package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/1homsi/kubecurses/internal/k8s"
	"github.com/1homsi/kubecurses/internal/model"
	"github.com/1homsi/kubecurses/internal/ui"
	"github.com/1homsi/kubecurses/internal/ui/views"
)

// ErrCancelled is returned when the user quits the context picker.
var ErrCancelled = errors.New("cancelled")

// rowCounter is satisfied by any overview view that can report its row count.
type rowCounter interface {
	RowCount() int
}

const maxLogLines = 5000

// App is the top-level application struct.
type App struct {
	cfg            Config
	screen         *ui.Screen
	state          model.AppState
	watcher        *k8s.Watcher
	watcherCancel  context.CancelFunc // cancels only the watcher goroutines (not Run itself)
	runCtx         context.Context    // Run()'s context, used when restarting the watcher
	views          [5]views.View
	nodeOverview   rowCounter            // kept for RowCount access; swap impl to change overview style
	xrayView       *views.XrayView       // typed for SelectedRef access
	nodeDetailView *views.NodeDetailView // heatmap node drill-down
	events         chan tcell.Event      // fed by a single long-lived goroutine
	logLines       chan string           // fed by the active log-streaming goroutine; nil when inactive
	logsCancel     context.CancelFunc
}

// New creates and initialises a new App from the given Config.
func New(cfg Config) (*App, error) {
	// Screen must come up first so the cluster picker can run inside it.
	scr, err := ui.NewScreen()
	if err != nil {
		return nil, fmt.Errorf("screen init: %w", err)
	}

	// Show context picker when no --context flag was given.
	if cfg.Context == "" {
		contexts, current, err := k8s.ListContexts(cfg.Kubeconfig)
		if err == nil && len(contexts) > 1 {
			chosen, quit := ui.PickContext(scr, contexts, current)
			if quit {
				scr.Fini()
				return nil, ErrCancelled
			}
			cfg.Context = chosen
			// Persist immediately so kubectl sees the chosen context.
			if chosen != current {
				_ = k8s.PersistCurrentContext(cfg.Kubeconfig, chosen)
			}
		}
	}

	cs, restCfg, err := k8s.NewClientAndConfig(k8s.ClientOptions{
		KubeconfigPath: cfg.Kubeconfig,
		ContextName:    cfg.Context,
		RequestTimeout: cfg.RequestTimeout,
		QPS:            cfg.KubeAPIQPS,
		Burst:          cfg.KubeAPIBurst,
	})
	if err != nil {
		scr.Fini()
		return nil, fmt.Errorf("kubernetes client: %w", err)
	}

	watcherOpts := k8s.WatcherOptions{
		Watch:           cfg.Watch,
		PollInterval:    cfg.PollInterval,
		MetricsInterval: cfg.MetricsInterval,
		EnableMetrics:   cfg.EnableMetrics,
		MaxPods:         cfg.MaxPods,
	}
	watcher := k8s.NewWatcher(cs, restCfg, cfg.Namespace, watcherOpts)
	ov := &views.NodeOverviewView{}
	xv := &views.XrayView{}
	ndv := &views.NodeDetailView{}

	app := &App{
		cfg:            cfg,
		screen:         scr,
		watcher:        watcher,
		nodeOverview:   ov,
		xrayView:       xv,
		nodeDetailView: ndv,
		state: model.AppState{
			Namespace:      cfg.Namespace,
			Context:        cfg.Context,
			LogsAutoScroll: true,
			NoIcons:        cfg.NoIcons,
		},
		views: [5]views.View{
			model.TabNodeOverview: ov,
			model.TabPods:         xv,
			model.TabDeployments:  &views.DeploymentsView{},
			model.TabNamespaces:   &views.NamespacesView{},
			model.TabHeatmap:      &views.HeatmapView{},
		},
	}
	return app, nil
}

// Run starts the watcher goroutines and enters the main event loop.
// It returns when the user quits or an unrecoverable error occurs.
func (a *App) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Store for switchCluster, which needs to derive a new watcher context.
	a.runCtx = ctx

	// Watchers run in their own child context so they can be cancelled and
	// restarted independently when the user switches clusters.
	watcherCtx, watcherCancel := context.WithCancel(ctx)
	a.watcherCancel = watcherCancel
	a.watcher.Start(watcherCtx)

	// Single persistent goroutine feeds all tcell events onto a.events.
	// This avoids goroutine accumulation / PollEvent deadlock on tab resume.
	a.events = make(chan tcell.Event, 16)
	go func() {
		for {
			ev := a.screen.PollEvent()
			if ev == nil {
				return // screen closed
			}
			select {
			case a.events <- ev:
			case <-ctx.Done():
				return
			}
		}
	}()

	a.draw()

	// Rate-limit redraws triggered by data/log updates to ~30 fps so that
	// high-frequency log streams don't saturate the terminal with renders.
	// Key/mouse events always draw immediately for responsive feel.
	const dataDrawInterval = 33 * time.Millisecond
	var lastDataDraw time.Time
	dataDraw := func() {
		if time.Since(lastDataDraw) >= dataDrawInterval {
			a.draw()
			lastDataDraw = time.Now()
		}
	}

	for {
		select {
		case <-ctx.Done():
			a.screen.Fini()
			return ctx.Err()

		case update := <-a.watcher.Updates():
			a.state.ApplyUpdate(update)
			dataDraw()

		case line := <-a.logLines:
			// Append incoming log line; cap buffer to avoid unbounded growth.
			a.state.LogsLines = append(a.state.LogsLines, line)
			if len(a.state.LogsLines) > maxLogLines {
				a.state.LogsLines = a.state.LogsLines[len(a.state.LogsLines)-maxLogLines:]
			}
			dataDraw()

		case tcellEv := <-a.events:
			if tcellEv == nil {
				continue
			}
			if quit := a.handleEvent(tcellEv); quit {
				a.screen.Fini()
				return nil
			}
			a.draw()
		}
	}
}

// handleEvent dispatches to the appropriate mode handler.
// Returns true if the application should quit.
func (a *App) handleEvent(ev tcell.Event) bool {
	// Terminal resize / resume — re-sync the screen to avoid display freeze.
	if _, ok := ev.(*tcell.EventResize); ok {
		a.screen.Sync()
		return false
	}
	// Mouse events are handled regardless of overlay state.
	if evMouse, ok := ev.(*tcell.EventMouse); ok {
		return a.handleMouseEvent(evMouse)
	}
	// Any key dismisses the help overlay.
	if a.state.HelpMode {
		a.state.HelpMode = false
		return false
	}
	if a.state.LogsMode {
		return a.applyLogsAction(ui.EventToAction(ev))
	}
	if a.state.ClusterPickerMode {
		return a.applyPickerAction(ui.EventToAction(ev))
	}
	if a.state.SearchMode {
		return a.applySearchAction(ui.SearchEventToAction(ev), ev)
	}
	return a.applyAction(ui.EventToAction(ev))
}

// handleMouseEvent translates tcell mouse events into scroll/navigation actions.
func (a *App) handleMouseEvent(ev *tcell.EventMouse) bool {
	btn := ev.Buttons()
	w, h := a.screen.Size()
	// Logs box content height accounts for tab bar, status bar, box borders, and status strip.
	logsContentH := h - 5
	switch {
	case btn&tcell.WheelUp != 0:
		switch {
		case a.state.LogsMode:
			a.logsScrollBy(-3, logsContentH, len(ui.CachedWrapLogs(&a.state, w-6)))
		case a.state.ClusterPickerMode:
			if a.state.ClusterPickerSel > 0 {
				a.state.ClusterPickerSel--
			}
		case a.state.HeatmapNodeDetail:
			if a.state.HeatmapDetailSel > 0 {
				a.state.HeatmapDetailSel -= 3
				if a.state.HeatmapDetailSel < 0 {
					a.state.HeatmapDetailSel = 0
				}
			}
		default:
			a.state.MoveSelection(-3, a.activeLen())
		}
	case btn&tcell.WheelDown != 0:
		switch {
		case a.state.LogsMode:
			a.logsScrollBy(3, logsContentH, len(ui.CachedWrapLogs(&a.state, w-6)))
		case a.state.ClusterPickerMode:
			if a.state.ClusterPickerSel < len(a.state.ClusterPickerList)-1 {
				a.state.ClusterPickerSel++
			}
		case a.state.HeatmapNodeDetail:
			n := a.heatmapDetailPodCount()
			a.state.HeatmapDetailSel += 3
			if a.state.HeatmapDetailSel >= n {
				a.state.HeatmapDetailSel = n - 1
			}
		default:
			a.state.MoveSelection(3, a.activeLen())
		}
	}
	return false
}

// applyLogsAction handles key events while the logs view is active.
func (a *App) applyLogsAction(action ui.Action) bool {
	w, h := a.screen.Size()
	// tab(1) + status(1) + box-top-border(1) + status-strip(1) + box-bottom-border(1)
	contentH := h - 5
	totalLines := len(ui.CachedWrapLogs(&a.state, w-6))

	switch action {
	case ui.ActionQuit:
		return true
	case ui.ActionSearchCancel: // Esc
		a.closeLogs()
	case ui.ActionMoveDown:
		a.logsScrollBy(1, contentH, totalLines)
	case ui.ActionMoveUp:
		a.logsScrollBy(-1, contentH, totalLines)
	case ui.ActionPageDown:
		a.logsScrollBy(contentH, contentH, totalLines)
	case ui.ActionPageUp:
		a.logsScrollBy(-contentH, contentH, totalLines)
	case ui.ActionLogsToggleScroll:
		a.state.LogsAutoScroll = !a.state.LogsAutoScroll
	}
	return false
}

// logsScrollBy scrolls the log view by delta lines (negative = up, positive = down).
// It handles the autoscroll→manual transition correctly:
//   - Any upward delta from autoscroll mode snaps to the current bottom first, so
//     the user starts scrolling from where they were visually, not from offset 0.
//   - Offset is clamped to [0, maxOffset] after every operation so trackpad bursts
//     never create drift that requires equal unwinding to recover.
//   - Scrolling to the very bottom re-engages autoscroll.
func (a *App) logsScrollBy(delta, contentH, totalLines int) {
	maxOff := totalLines - contentH
	if maxOff < 0 {
		maxOff = 0
	}

	if delta < 0 {
		// Upward scroll: if autoscroll was on, snap to the real bottom first.
		if a.state.LogsAutoScroll {
			a.state.LogsOffset = maxOff
			a.state.LogsAutoScroll = false
		}
		a.state.LogsOffset += delta // delta is negative
		if a.state.LogsOffset < 0 {
			a.state.LogsOffset = 0
		}
	} else {
		// Downward scroll: turn off autoscroll and clamp to max.
		a.state.LogsAutoScroll = false
		a.state.LogsOffset += delta
		if a.state.LogsOffset >= maxOff {
			a.state.LogsOffset = maxOff
			a.state.LogsAutoScroll = true // re-engage autoscroll at the bottom
		}
	}
}

// applyPickerAction handles key events while the cluster picker overlay is open.
func (a *App) applyPickerAction(action ui.Action) bool {
	list := a.state.ClusterPickerList
	switch action {
	case ui.ActionQuit:
		return true
	case ui.ActionSearchCancel: // Esc — dismiss without switching
		a.state.ClusterPickerMode = false
	case ui.ActionConfirm: // Enter — switch to selected context
		if a.state.ClusterPickerSel < len(list) {
			chosen := list[a.state.ClusterPickerSel]
			a.state.ClusterPickerMode = false
			a.switchCluster(chosen)
		}
	case ui.ActionMoveDown:
		if a.state.ClusterPickerSel < len(list)-1 {
			a.state.ClusterPickerSel++
		}
	case ui.ActionMoveUp:
		if a.state.ClusterPickerSel > 0 {
			a.state.ClusterPickerSel--
		}
	}
	return false
}

// switchCluster reconnects to a different Kubernetes context without restarting Run.
func (a *App) switchCluster(newContext string) {
	// Always persist the chosen context so kubectl sees the same cluster after
	// kubecurses exits — even when we're already connected (e.g. re-confirming the
	// current context after a startup-picker selection that wasn't yet persisted).
	if err := k8s.PersistCurrentContext(a.cfg.Kubeconfig, newContext); err != nil {
		a.state.LastErr = fmt.Errorf("persist context: %w", err)
	}

	if newContext == a.cfg.Context {
		return // already connected, no reconnect needed
	}
	// Stop existing watcher goroutines.
	if a.watcherCancel != nil {
		a.watcherCancel()
	}
	// Build a new client for the chosen context.
	cs, restCfg, err := k8s.NewClientAndConfig(k8s.ClientOptions{
		KubeconfigPath: a.cfg.Kubeconfig,
		ContextName:    newContext,
		RequestTimeout: a.cfg.RequestTimeout,
		QPS:            a.cfg.KubeAPIQPS,
		Burst:          a.cfg.KubeAPIBurst,
	})
	if err != nil {
		a.state.LastErr = fmt.Errorf("switch cluster: %w", err)
		return
	}
	// Close any active log overlays before swapping the watcher so their
	// goroutines don't outlive the old connection.
	a.closeLogs()
	// Swap the watcher.
	watcherOpts := k8s.WatcherOptions{
		Watch:           a.cfg.Watch,
		PollInterval:    a.cfg.PollInterval,
		MetricsInterval: a.cfg.MetricsInterval,
		EnableMetrics:   a.cfg.EnableMetrics,
		MaxPods:         a.cfg.MaxPods,
	}
	a.watcher = k8s.NewWatcher(cs, restCfg, a.cfg.Namespace, watcherOpts)
	watcherCtx, watcherCancel := context.WithCancel(a.runCtx)
	a.watcherCancel = watcherCancel
	a.watcher.Start(watcherCtx)
	// Update config and visible state; clear stale data.
	a.cfg.Context = newContext
	a.state.Context = newContext
	a.state.Pods = nil
	a.state.Nodes = nil
	a.state.Namespaces = nil
	a.state.Deployments = nil
}

// openLogs starts streaming logs for the given pod/container.
func (a *App) openLogs(ns, pod, container string) {
	// Cancel any in-progress stream first.
	if a.logsCancel != nil {
		a.logsCancel()
		a.logsCancel = nil
	}

	logCtx, cancel := context.WithCancel(context.Background())
	a.logsCancel = cancel

	// New buffered channel — nil-ing the old one means any goroutine blocked
	// on the previous channel's send will exit via ctx.Done().
	a.logLines = make(chan string, 200)
	a.state.LogsLines = nil
	a.state.LogsOffset = 0
	a.state.LogsAutoScroll = true
	a.state.LogsNamespace = ns
	a.state.LogsPod = pod
	a.state.LogsContainer = container
	a.state.LogsMode = true

	cs := a.watcher.Clientset()
	ch := a.logLines
	go k8s.StreamLogs(logCtx, cs, ns, pod, container, a.cfg.LogTailLines, ch)
}

// closeLogs stops the active log stream and hides the logs view.
func (a *App) closeLogs() {
	if a.logsCancel != nil {
		a.logsCancel()
		a.logsCancel = nil
	}
	a.logLines = nil // nil channel never fires in select
	a.state.LogsMode = false
	a.state.LogsLines = nil
}

// runInteractiveExec suspends the TUI, delegates to `kubectl exec -it` for the
// actual shell session, then resumes the TUI when the session ends.
//
// Delegating to kubectl guarantees the exact same authentication flow the user
// gets from their terminal: kubectl refreshes the exec-auth-plugin token on
// every invocation, respects AWS_PROFILE / assumed roles in the kubeconfig, and
// handles raw-mode setup, SIGWINCH forwarding, and PTY teardown internally.
//
// Flow:
//  1. Suspend tcell — terminal is returned to normal (cooked) mode.
//  2. Run `kubectl exec -it` with stdin/stdout/stderr wired to the real terminal.
//  3. Resume tcell, sync, and redraw.
func (a *App) runInteractiveExec(ns, pod, container string) {
	// 1. Hand the terminal back to the OS.
	if err := a.screen.Suspend(); err != nil {
		a.state.LastErr = fmt.Errorf("exec suspend: %w", err)
		return
	}

	// 2. Build and run `kubectl exec -it <pod> -n <ns> [--context …] [--kubeconfig …] [-c <container>] -- /bin/sh`.
	args := []string{"exec", "-it", pod, "-n", ns}
	if a.cfg.Context != "" {
		args = append(args, "--context", a.cfg.Context)
	}
	if a.cfg.Kubeconfig != "" {
		args = append(args, "--kubeconfig", a.cfg.Kubeconfig)
	}
	if container != "" {
		args = append(args, "-c", container)
	}
	args = append(args, "--", "/bin/sh")

	cmd := exec.Command("kubectl", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	execErr := cmd.Run()

	// 3. Bring tcell back up.
	if err := a.screen.Resume(); err != nil {
		a.state.LastErr = fmt.Errorf("exec resume: %w", err)
		return
	}
	a.screen.Sync()

	// Surface unexpected errors; normal shell exit (exit code ≥ 1) is not an error.
	if execErr != nil && a.runCtx.Err() == nil {
		var exitErr *exec.ExitError
		if !errors.As(execErr, &exitErr) {
			a.state.LastErr = fmt.Errorf("exec: %w", execErr)
		}
	}

	// Drain any watcher updates that arrived while exec was running, then redraw.
	for {
		select {
		case update := <-a.watcher.Updates():
			a.state.ApplyUpdate(update)
		default:
			a.draw()
			return
		}
	}
}

// handleExecOpen starts an interactive exec for the currently selected pod.
func (a *App) handleExecOpen() {
	if a.state.ActiveTab == model.TabPods { // Xray tab
		ns, pod, container := a.xrayView.SelectedRef(a.state.Selection[model.TabPods])
		if pod != "" {
			a.runInteractiveExec(ns, pod, container)
		}
	}
}

// handleExecOpenForDetail starts an interactive exec for the selected pod in
// the node-detail view.
func (a *App) handleExecOpenForDetail() {
	pods := views.NodeDetailPods(&a.state)
	if a.state.HeatmapDetailSel < len(pods) {
		p := pods[a.state.HeatmapDetailSel]
		a.runInteractiveExec(p.Namespace, p.Name, "")
	}
}

// applySearchAction handles key events while the search bar is active.
func (a *App) applySearchAction(action ui.Action, ev tcell.Event) bool {
	switch action {
	case ui.ActionQuit:
		return true
	case ui.ActionSearchCancel:
		a.state.SearchMode = false
		a.state.SearchQuery = ""
	case ui.ActionSearchCommit:
		a.state.SearchMode = false
	case ui.ActionSearchBack:
		q := []rune(a.state.SearchQuery)
		if len(q) > 0 {
			a.state.SearchQuery = string(q[:len(q)-1])
		}
	case ui.ActionSearchAppend:
		if evKey, ok := ev.(*tcell.EventKey); ok {
			a.state.SearchQuery += string(evKey.Rune())
		}
	}
	return false
}

// applyAction mutates app state based on the given action.
// Returns true if the application should quit.
func (a *App) applyAction(action ui.Action) bool {
	onHeatmap := a.state.ActiveTab == model.TabHeatmap
	inDetail := a.state.HeatmapNodeDetail

	switch action {
	case ui.ActionQuit:
		return true

	case ui.ActionNextTab:
		a.state.HeatmapNodeDetail = false
		a.state.NextTab()
	case ui.ActionPrevTab:
		a.state.HeatmapNodeDetail = false
		a.state.PrevTab()
	case ui.ActionTab1:
		a.state.HeatmapNodeDetail = false
		a.state.SetTab(model.TabHeatmap)
	case ui.ActionTab2:
		a.state.HeatmapNodeDetail = false
		a.state.SetTab(model.TabNodeOverview)
	case ui.ActionTab3:
		a.state.HeatmapNodeDetail = false
		a.state.SetTab(model.TabPods)
	case ui.ActionTab4:
		a.state.HeatmapNodeDetail = false
		a.state.SetTab(model.TabDeployments)
	case ui.ActionTab5:
		a.state.HeatmapNodeDetail = false
		a.state.SetTab(model.TabNamespaces)

	case ui.ActionMoveUp:
		if onHeatmap && !inDetail {
			a.heatmapMoveUp()
		} else if inDetail {
			if a.state.HeatmapDetailSel > 0 {
				a.state.HeatmapDetailSel--
			}
		} else {
			a.state.MoveSelection(a.upDelta(), a.activeLen())
		}

	case ui.ActionMoveDown:
		if onHeatmap && !inDetail {
			a.heatmapMoveDown()
		} else if inDetail {
			if a.state.HeatmapDetailSel < a.heatmapDetailPodCount()-1 {
				a.state.HeatmapDetailSel++
			}
		} else {
			a.state.MoveSelection(a.downDelta(), a.activeLen())
		}

	case ui.ActionMoveLeft: // 'h'
		if onHeatmap && !inDetail {
			a.heatmapMoveLeft()
		} else {
			a.gridMoveHorizontal(-1)
		}

	case ui.ActionMoveRight: // right-arrow
		if onHeatmap && !inDetail {
			a.heatmapMoveRight()
		} else {
			a.gridMoveHorizontal(1)
		}

	case ui.ActionPageUp:
		if onHeatmap && !inDetail {
			plan := a.state.HeatmapRowPlan
			sel := a.state.Selection[model.TabHeatmap]
			row, col := model.HeatmapNodeToRowColPlan(plan, sel)
			newRow := row - 3
			if newRow < 0 {
				newRow = 0
			}
			newCol := col
			if len(plan) > newRow && newCol >= plan[newRow] {
				newCol = plan[newRow] - 1
			}
			if newSel := model.HeatmapRowColToNodePlan(plan, newRow, newCol); newSel >= 0 && newSel < len(a.state.Nodes) {
				a.state.Selection[model.TabHeatmap] = newSel
			}
		} else if inDetail {
			a.state.HeatmapDetailSel -= 10
			if a.state.HeatmapDetailSel < 0 {
				a.state.HeatmapDetailSel = 0
			}
		} else {
			a.state.MoveSelection(-4, a.activeLen())
		}

	case ui.ActionPageDown:
		if onHeatmap && !inDetail {
			plan := a.state.HeatmapRowPlan
			sel := a.state.Selection[model.TabHeatmap]
			row, col := model.HeatmapNodeToRowColPlan(plan, sel)
			newRow := row + 3
			if newRow >= len(plan) {
				newRow = len(plan) - 1
			}
			newCol := col
			if len(plan) > newRow && newCol >= plan[newRow] {
				newCol = plan[newRow] - 1
			}
			if newSel := model.HeatmapRowColToNodePlan(plan, newRow, newCol); newSel >= 0 && newSel < len(a.state.Nodes) {
				a.state.Selection[model.TabHeatmap] = newSel
			}
		} else if inDetail {
			n := a.heatmapDetailPodCount()
			a.state.HeatmapDetailSel += 10
			if a.state.HeatmapDetailSel >= n {
				a.state.HeatmapDetailSel = n - 1
			}
		} else {
			a.state.MoveSelection(4, a.activeLen())
		}

	case ui.ActionConfirm: // Enter
		if onHeatmap && !inDetail {
			// Drill into the selected node.
			if sel := a.state.Selection[model.TabHeatmap]; sel < len(a.state.Nodes) {
				a.state.HeatmapDetailNode = a.state.Nodes[sel].Name
				a.state.HeatmapDetailSel = 0
				a.state.HeatmapNodeDetail = true
			}
		} else if a.state.ActiveTab == model.TabNamespaces {
			a.drillIntoNamespace()
		}

	case ui.ActionSearchCancel: // Esc
		if inDetail {
			a.state.HeatmapNodeDetail = false
		} else {
			a.state.SearchQuery = ""
		}

	case ui.ActionRefresh:
		a.watcher.TriggerRefresh()

	case ui.ActionSearchOpen:
		if !onHeatmap {
			a.state.SearchMode = true
			a.state.SearchQuery = ""
		}

	case ui.ActionHelp:
		a.state.HelpMode = true

	case ui.ActionLogsOpen: // 'l'
		if onHeatmap && !inDetail {
			// l = move right within the heatmap grid
			a.heatmapMoveRight()
		} else if inDetail {
			// l = open logs for the selected pod in the detail view
			a.handleLogsOpenForDetail()
		} else {
			a.handleLogsOpen()
		}

	case ui.ActionExecOpen: // 'e'
		if inDetail {
			a.handleExecOpenForDetail()
		} else {
			a.handleExecOpen()
		}

	case ui.ActionSwitchCluster:
		a.handleClusterPickerOpen()
	}
	return false
}

// drillIntoNamespace filters the Xray view to the selected namespace and
// navigates to it so the user sees all pods/containers in that namespace.
func (a *App) drillIntoNamespace() {
	sel := a.state.Selection[model.TabNamespaces]
	name := views.SelectedNamespace(sel, &a.state)
	if name == "" {
		return
	}
	a.state.SearchQuery = name
	a.state.SearchMode = false
	a.state.Selection[model.TabPods] = 0
	a.state.SetTab(model.TabPods)
}

// handleClusterPickerOpen fetches available contexts and opens the in-app picker.
func (a *App) handleClusterPickerOpen() {
	contexts, current, err := k8s.ListContexts(a.cfg.Kubeconfig)
	if err != nil || len(contexts) == 0 {
		return
	}
	sel := 0
	for i, c := range contexts {
		if c == current {
			sel = i
			break
		}
	}
	a.state.ClusterPickerList = contexts
	a.state.ClusterPickerCurr = current
	a.state.ClusterPickerSel = sel
	a.state.ClusterPickerMode = true
}

// handleLogsOpen opens logs for the currently selected pod/container.
func (a *App) handleLogsOpen() {
	switch a.state.ActiveTab {
	case model.TabPods: // Xray tab
		ns, pod, container := a.xrayView.SelectedRef(a.state.Selection[model.TabPods])
		if pod != "" {
			a.openLogs(ns, pod, container)
		}
	}
}

func (a *App) upDelta() int   { return -1 }
func (a *App) downDelta() int { return 1 }

// gridMoveHorizontal moves the selected node card left (dir=-1) or right
// (dir=+1) within the 2-column NodeOverview grid.
func (a *App) gridMoveHorizontal(dir int) {
	if a.state.ActiveTab != model.TabNodeOverview {
		return
	}
	sel := a.state.Selection[model.TabNodeOverview]
	col := sel % 2 // 0 = left column, 1 = right column
	if dir == -1 && col == 0 {
		return // already in left column
	}
	if dir == 1 && col == 1 {
		return // already in right column
	}
	a.state.MoveSelection(dir, a.activeLen())
}

// heatmapMoveUp moves the heatmap selection up by one grid row.
func (a *App) heatmapMoveUp() {
	plan := a.state.HeatmapRowPlan
	if len(a.state.Nodes) == 0 || len(plan) == 0 {
		return
	}
	sel := a.state.Selection[model.TabHeatmap]
	row, col := model.HeatmapNodeToRowColPlan(plan, sel)
	if row == 0 {
		return
	}
	newRow := row - 1
	newCol := col
	if newCol >= plan[newRow] {
		newCol = plan[newRow] - 1
	}
	newSel := model.HeatmapRowColToNodePlan(plan, newRow, newCol)
	if newSel >= 0 && newSel < len(a.state.Nodes) {
		a.state.Selection[model.TabHeatmap] = newSel
	}
}

// heatmapMoveDown moves the heatmap selection down by one grid row.
func (a *App) heatmapMoveDown() {
	plan := a.state.HeatmapRowPlan
	if len(a.state.Nodes) == 0 || len(plan) == 0 {
		return
	}
	sel := a.state.Selection[model.TabHeatmap]
	row, col := model.HeatmapNodeToRowColPlan(plan, sel)
	newRow := row + 1
	if newRow >= len(plan) {
		return
	}
	newCol := col
	if newCol >= plan[newRow] {
		newCol = plan[newRow] - 1
	}
	newSel := model.HeatmapRowColToNodePlan(plan, newRow, newCol)
	if newSel >= 0 && newSel < len(a.state.Nodes) {
		a.state.Selection[model.TabHeatmap] = newSel
	}
}

// heatmapMoveLeft moves the heatmap selection left by one cell within its row.
func (a *App) heatmapMoveLeft() {
	plan := a.state.HeatmapRowPlan
	if len(a.state.Nodes) == 0 || len(plan) == 0 {
		return
	}
	sel := a.state.Selection[model.TabHeatmap]
	_, col := model.HeatmapNodeToRowColPlan(plan, sel)
	if col > 0 {
		a.state.Selection[model.TabHeatmap]--
	}
}

// heatmapMoveRight moves the heatmap selection right by one cell within its row.
func (a *App) heatmapMoveRight() {
	plan := a.state.HeatmapRowPlan
	if len(a.state.Nodes) == 0 || len(plan) == 0 {
		return
	}
	sel := a.state.Selection[model.TabHeatmap]
	row, col := model.HeatmapNodeToRowColPlan(plan, sel)
	next := sel + 1
	if col+1 < plan[row] && next < len(a.state.Nodes) {
		a.state.Selection[model.TabHeatmap] = next
	}
}

// handleLogsOpenForDetail opens logs for the selected pod in the node-detail view.
func (a *App) handleLogsOpenForDetail() {
	pods := views.NodeDetailPods(&a.state)
	if a.state.HeatmapDetailSel < len(pods) {
		p := pods[a.state.HeatmapDetailSel]
		a.openLogs(p.Namespace, p.Name, "")
	}
}

// activeLen returns the number of navigable rows in the currently active view.
func (a *App) activeLen() int {
	switch a.state.ActiveTab {
	case model.TabNodeOverview:
		return a.nodeOverview.RowCount()
	case model.TabPods:
		return a.xrayView.RowCount()
	case model.TabNamespaces:
		return len(a.state.Namespaces)
	case model.TabDeployments:
		return len(a.state.Deployments)
	case model.TabHeatmap:
		if a.state.HeatmapNodeDetail {
			return a.heatmapDetailPodCount()
		}
		return len(a.state.Nodes)
	}
	return 0
}

// heatmapDetailPodCount returns the number of pods on the node being detailed.
func (a *App) heatmapDetailPodCount() int {
	count := 0
	for _, p := range a.state.Pods {
		if p.Node == a.state.HeatmapDetailNode {
			count++
		}
	}
	return count
}

// draw renders the entire screen.
func (a *App) draw() {
	w, h := a.screen.Size()
	a.screen.Clear()
	ui.DrawTabBar(a.screen, w, a.state.ActiveTab, &a.state)
	contentRect := ui.ContentRect(w, h)
	switch {
	case a.state.LogsMode:
		ui.DrawLogsView(a.screen, contentRect, &a.state)
	case a.state.HeatmapNodeDetail:
		a.nodeDetailView.Draw(a.screen, contentRect, &a.state)
	default:
		a.views[a.state.ActiveTab].Draw(a.screen, contentRect, &a.state)
	}
	ui.DrawStatusBar(a.screen, w, h, &a.state)
	if a.state.HelpMode {
		ui.DrawHelp(a.screen, w, h)
	}
	if a.state.ClusterPickerMode {
		ui.DrawClusterPicker(a.screen, &a.state)
	}
	a.screen.Show()
}
