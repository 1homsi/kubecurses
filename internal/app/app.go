package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/1homsi/kubecurses/internal/k8s"
	"github.com/1homsi/kubecurses/internal/model"
	"github.com/1homsi/kubecurses/internal/ui"
	"github.com/1homsi/kubecurses/internal/ui/views"
)

// ErrCancelled is returned when the user quits the context picker.
var ErrCancelled = errors.New("cancelled")

const maxLogLines = 5000

// ── Bubble Tea message types ─────────────────────────────────────────────────

type watcherMsg model.Update
type logLineMsg string
type logStreamEndMsg struct{}
type execDoneMsg struct {
	err    error
	stderr string
}

// ── App (tea.Model) ──────────────────────────────────────────────────────────

// App is the top-level Bubble Tea model.
type App struct {
	cfg             Config
	width, height   int
	state           model.AppState
	watcher         *k8s.Watcher
	watcherCancel   context.CancelFunc
	runCtx          context.Context
	runCancel       context.CancelFunc
	views           [6]views.View
	nodeOverview    *views.NodeOverviewView
	xrayView        *views.XrayView
	nodeDetailView  *views.NodeDetailView
	deploymentsView *views.DeploymentsView
	namespacesView  *views.NamespacesView
	eventsView      *views.EventsView
	logLines        chan string
	logsCancel      context.CancelFunc
	spinner         spinner.Model
	loading         bool // true until both nodes and pods have arrived
}

// New creates and initialises a new App from the given Config.
// The context picker is handled by the caller before New is called.
func New(cfg Config) (*App, error) {
	cs, restCfg, err := k8s.NewClientAndConfig(k8s.ClientOptions{
		KubeconfigPath: cfg.Kubeconfig,
		ContextName:    cfg.Context,
		RequestTimeout: cfg.RequestTimeout,
		QPS:            cfg.KubeAPIQPS,
		Burst:          cfg.KubeAPIBurst,
	})
	if err != nil {
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
	dv := &views.DeploymentsView{}
	nsv := &views.NamespacesView{}
	ev := &views.EventsView{}

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#82BEFF"))

	app := &App{
		cfg:             cfg,
		watcher:         watcher,
		nodeOverview:    ov,
		xrayView:        xv,
		nodeDetailView:  ndv,
		deploymentsView: dv,
		namespacesView:  nsv,
		eventsView:      ev,
		spinner:         sp,
		loading:         true,
		state: model.AppState{
			Namespace:      cfg.Namespace,
			Context:        cfg.Context,
			LogsAutoScroll: true,
			NoIcons:        cfg.NoIcons,
		},
		views: [6]views.View{
			model.TabNodeOverview: ov,
			model.TabPods:         xv,
			model.TabDeployments:  dv,
			model.TabNamespaces:   nsv,
			model.TabHeatmap:      &views.HeatmapView{},
			model.TabEvents:       ev,
		},
	}
	return app, nil
}

// ── tea.Model interface ──────────────────────────────────────────────────────

func (a *App) Init() tea.Cmd {
	ctx, cancel := context.WithCancel(context.Background())
	a.runCtx = ctx
	a.runCancel = cancel

	watcherCtx, watcherCancel := context.WithCancel(ctx)
	a.watcherCancel = watcherCancel
	a.watcher.Start(watcherCtx)

	return tea.Batch(a.waitForWatcherUpdate(), a.spinner.Tick)
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil

	case tea.KeyMsg:
		cmd := a.handleKey(msg)
		return a, cmd

	case tea.MouseMsg:
		a.handleMouse(msg)
		return a, nil

	case spinner.TickMsg:
		if a.loading {
			var cmd tea.Cmd
			a.spinner, cmd = a.spinner.Update(msg)
			return a, cmd
		}
		return a, nil

	case watcherMsg:
		a.state.ApplyUpdate(model.Update(msg))
		if a.loading && a.state.NodesLoaded && a.state.PodsLoaded {
			a.loading = false
		}
		return a, a.waitForWatcherUpdate()

	case logLineMsg:
		a.state.LogsLines = append(a.state.LogsLines, string(msg))
		if len(a.state.LogsLines) > maxLogLines {
			a.state.LogsLines = a.state.LogsLines[len(a.state.LogsLines)-maxLogLines:]
		}
		return a, a.waitForLogLine()

	case logStreamEndMsg:
		// Stream ended; stop waiting for more lines.
		return a, nil

	case execDoneMsg:
		if msg.err != nil && a.runCtx.Err() == nil {
			errMsg := strings.TrimSpace(msg.stderr)
			if errMsg == "" {
				errMsg = msg.err.Error()
			}
			a.state.LastErr = fmt.Errorf("exec: %s", errMsg)
		}
		// Drain any watcher updates that arrived while exec was running.
		for {
			select {
			case update := <-a.watcher.Updates():
				a.state.ApplyUpdate(update)
			default:
				return a, a.waitForWatcherUpdate()
			}
		}
	}
	return a, nil
}

func (a *App) View() string {
	if a.width == 0 || a.height == 0 {
		return ""
	}

	w, h := a.width, a.height
	tabBar := ui.RenderTabBar(w, a.state.ActiveTab, &a.state)
	contentRect := ui.ContentRect(w, h)

	var content string
	switch {
	case a.state.LogsMode:
		content = ui.RenderLogsView(contentRect.W, contentRect.H, &a.state)
	case a.state.DescribeMode:
		content = ui.RenderDescribeOverlay(contentRect.W, contentRect.H, &a.state)
	case a.state.HeatmapNodeDetail:
		content = a.nodeDetailView.Render(contentRect.W, contentRect.H, &a.state)
	default:
		content = a.views[a.state.ActiveTab].Render(contentRect.W, contentRect.H, &a.state)
	}

	// Ensure content is exactly contentRect.H lines so the status bar
	// always appears at the bottom of the terminal.
	contentLines := strings.Split(content, "\n")
	for len(contentLines) < contentRect.H {
		contentLines = append(contentLines, ui.FillWidth(w, ui.StyleDefault))
	}
	if len(contentLines) > contentRect.H {
		contentLines = contentLines[:contentRect.H]
	}
	content = strings.Join(contentLines, "\n")

	statusBar := ui.RenderStatusBar(w, &a.state)
	base := tabBar + "\n" + content + "\n" + statusBar

	if a.loading {
		spinnerStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#82BEFF")).
			Background(ui.StyleDefault.GetBackground())
		loadingText := spinnerStyle.Render(a.spinner.View() + " Loading cluster data…")
		base = lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, loadingText,
			lipgloss.WithWhitespaceBackground(ui.StyleDefault.GetBackground()))
	}

	if a.state.HelpMode {
		overlay := ui.RenderHelp(w, h)
		base = lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, overlay,
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(ui.StyleDefault.GetForeground()),
			lipgloss.WithWhitespaceBackground(ui.StyleDefault.GetBackground()))
	}
	if a.state.ClusterPickerMode {
		overlay := ui.RenderClusterPicker(&a.state, w, h)
		base = lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, overlay,
			lipgloss.WithWhitespaceBackground(ui.StyleDefault.GetBackground()))
	}
	if a.state.NamespacePickerMode {
		overlay := ui.RenderNamespacePicker(&a.state, w, h)
		base = lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, overlay,
			lipgloss.WithWhitespaceBackground(ui.StyleDefault.GetBackground()))
	}

	return base
}

// ── Async commands ───────────────────────────────────────────────────────────

func (a *App) waitForWatcherUpdate() tea.Cmd {
	ch := a.watcher.Updates()
	done := a.watcher.Done()
	return func() tea.Msg {
		select {
		case update, ok := <-ch:
			if !ok {
				return nil
			}
			return watcherMsg(update)
		case <-done:
			return nil
		}
	}
}

func (a *App) waitForLogLine() tea.Cmd {
	ch := a.logLines
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		line, ok := <-ch
		if !ok {
			return logStreamEndMsg{}
		}
		return logLineMsg(line)
	}
}

// ── Key handling ─────────────────────────────────────────────────────────────

// handleKey dispatches key events to the appropriate mode handler.
// Returns a tea.Cmd (nil if no command needed, tea.Quit to exit).
func (a *App) handleKey(msg tea.KeyMsg) tea.Cmd {
	// Any key dismisses the help overlay.
	if a.state.HelpMode {
		a.state.HelpMode = false
		return nil
	}
	if a.state.LogsMode {
		return a.applyLogsAction(ui.KeyToAction(msg))
	}
	if a.state.DescribeMode {
		return a.applyDescribeAction(ui.KeyToAction(msg))
	}
	if a.state.ClusterPickerMode {
		return a.applyPickerAction(ui.KeyToAction(msg))
	}
	if a.state.NamespacePickerMode {
		return a.applyNamespacePickerAction(ui.KeyToAction(msg))
	}
	if a.state.SearchMode {
		return a.applySearchAction(ui.SearchKeyToAction(msg), msg)
	}
	return a.applyAction(ui.KeyToAction(msg))
}

// ── Mouse handling ───────────────────────────────────────────────────────────

func (a *App) handleMouse(msg tea.MouseMsg) {
	w := a.width
	h := a.height
	logsContentH := h - 5

	switch msg.Button {
	case tea.MouseButtonWheelUp:
		switch {
		case a.state.LogsMode:
			a.logsScrollBy(-3, logsContentH, len(ui.CachedWrapLogs(&a.state, w-6)))
		case a.state.DescribeMode:
			a.state.DescribeOffset -= 3
			if a.state.DescribeOffset < 0 {
				a.state.DescribeOffset = 0
			}
		case a.state.ClusterPickerMode:
			if a.state.ClusterPickerSel > 0 {
				a.state.ClusterPickerSel--
			}
		case a.state.NamespacePickerMode:
			if a.state.NamespacePickerSel > 0 {
				a.state.NamespacePickerSel--
			}
		case a.state.HeatmapNodeDetail:
			a.state.HeatmapDetailSel -= 3
			if a.state.HeatmapDetailSel < 0 {
				a.state.HeatmapDetailSel = 0
			}
		default:
			a.state.MoveSelection(-3, a.activeLen())
		}
	case tea.MouseButtonWheelDown:
		switch {
		case a.state.LogsMode:
			a.logsScrollBy(3, logsContentH, len(ui.CachedWrapLogs(&a.state, w-6)))
		case a.state.DescribeMode:
			a.state.DescribeOffset += 3
		case a.state.ClusterPickerMode:
			if a.state.ClusterPickerSel < len(a.state.ClusterPickerList)-1 {
				a.state.ClusterPickerSel++
			}
		case a.state.NamespacePickerMode:
			if a.state.NamespacePickerSel < len(a.state.NamespacePickerList)-1 {
				a.state.NamespacePickerSel++
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
}

// ── Logs action handler ──────────────────────────────────────────────────────

func (a *App) applyLogsAction(action ui.Action) tea.Cmd {
	w, h := a.width, a.height
	contentH := h - 5
	totalLines := len(ui.CachedWrapLogs(&a.state, w-6))

	switch action {
	case ui.ActionQuit:
		return tea.Quit
	case ui.ActionSearchCancel:
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
	return nil
}

func (a *App) logsScrollBy(delta, contentH, totalLines int) {
	maxOff := totalLines - contentH
	if maxOff < 0 {
		maxOff = 0
	}

	if delta < 0 {
		if a.state.LogsAutoScroll {
			a.state.LogsOffset = maxOff
			a.state.LogsAutoScroll = false
		}
		a.state.LogsOffset += delta
		if a.state.LogsOffset < 0 {
			a.state.LogsOffset = 0
		}
	} else {
		a.state.LogsAutoScroll = false
		a.state.LogsOffset += delta
		if a.state.LogsOffset >= maxOff {
			a.state.LogsOffset = maxOff
			a.state.LogsAutoScroll = true
		}
	}
}

// ── Picker action handler ────────────────────────────────────────────────────

func (a *App) applyPickerAction(action ui.Action) tea.Cmd {
	list := a.state.ClusterPickerList
	switch action {
	case ui.ActionQuit:
		return tea.Quit
	case ui.ActionSearchCancel:
		a.state.ClusterPickerMode = false
	case ui.ActionConfirm:
		if a.state.ClusterPickerSel < len(list) {
			chosen := list[a.state.ClusterPickerSel]
			a.state.ClusterPickerMode = false
			a.switchCluster(chosen)
			return a.waitForWatcherUpdate()
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
	return nil
}

// ── Describe action handler ──────────────────────────────────────────────────

func (a *App) applyDescribeAction(action ui.Action) tea.Cmd {
	h := a.height
	contentH := h - 3
	totalLines := len(a.state.DescribeLines)
	maxOff := totalLines - contentH
	if maxOff < 0 {
		maxOff = 0
	}
	switch action {
	case ui.ActionQuit:
		return tea.Quit
	case ui.ActionSearchCancel:
		a.state.DescribeMode = false
	case ui.ActionMoveDown:
		if a.state.DescribeOffset < maxOff {
			a.state.DescribeOffset++
		}
	case ui.ActionMoveUp:
		if a.state.DescribeOffset > 0 {
			a.state.DescribeOffset--
		}
	case ui.ActionPageDown:
		a.state.DescribeOffset += contentH
		if a.state.DescribeOffset > maxOff {
			a.state.DescribeOffset = maxOff
		}
	case ui.ActionPageUp:
		a.state.DescribeOffset -= contentH
		if a.state.DescribeOffset < 0 {
			a.state.DescribeOffset = 0
		}
	}
	return nil
}

// ── Namespace picker action handler ──────────────────────────────────────────

func (a *App) applyNamespacePickerAction(action ui.Action) tea.Cmd {
	list := a.state.NamespacePickerList
	switch action {
	case ui.ActionQuit:
		return tea.Quit
	case ui.ActionSearchCancel:
		a.state.NamespacePickerMode = false
	case ui.ActionConfirm:
		if a.state.NamespacePickerSel < len(list) {
			a.state.NamespaceFilter = list[a.state.NamespacePickerSel]
			a.state.NamespacePickerMode = false
		}
	case ui.ActionMoveDown:
		if a.state.NamespacePickerSel < len(list)-1 {
			a.state.NamespacePickerSel++
		}
	case ui.ActionMoveUp:
		if a.state.NamespacePickerSel > 0 {
			a.state.NamespacePickerSel--
		}
	}
	return nil
}

func (a *App) handleNamespacePickerOpen() {
	list := make([]string, 0, len(a.state.Namespaces)+1)
	list = append(list, "")
	for _, ns := range a.state.Namespaces {
		list = append(list, ns.Name)
	}
	sel := 0
	for i, ns := range list {
		if ns == a.state.NamespaceFilter {
			sel = i
			break
		}
	}
	a.state.NamespacePickerList = list
	a.state.NamespacePickerSel = sel
	a.state.NamespacePickerMode = true
}

// ── Search action handler ────────────────────────────────────────────────────

func (a *App) applySearchAction(action ui.Action, msg tea.KeyMsg) tea.Cmd {
	switch action {
	case ui.ActionQuit:
		return tea.Quit
	case ui.ActionSearchCancel:
		a.state.SearchMode = false
		a.state.SetActiveSearchQuery("")
	case ui.ActionSearchCommit:
		a.state.SearchMode = false
	case ui.ActionSearchBack:
		q := []rune(a.state.ActiveSearchQuery())
		if len(q) > 0 {
			a.state.SetActiveSearchQuery(string(q[:len(q)-1]))
		}
	case ui.ActionSearchAppend:
		if len(msg.Runes) > 0 {
			a.state.SetActiveSearchQuery(a.state.ActiveSearchQuery() + string(msg.Runes))
		}
	}
	return nil
}

// ── Main action handler ─────────────────────────────────────────────────────

func (a *App) applyAction(action ui.Action) tea.Cmd {
	onHeatmap := a.state.ActiveTab == model.TabHeatmap
	inDetail := a.state.HeatmapNodeDetail

	switch action {
	case ui.ActionQuit:
		return tea.Quit

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
	case ui.ActionTab6:
		a.state.HeatmapNodeDetail = false
		a.state.SetTab(model.TabEvents)

	case ui.ActionMoveUp:
		if onHeatmap && !inDetail {
			a.heatmapMoveUp()
		} else if inDetail {
			if a.state.HeatmapDetailSel > 0 {
				a.state.HeatmapDetailSel--
			}
		} else {
			a.state.MoveSelection(-1, a.activeLen())
		}

	case ui.ActionMoveDown:
		if onHeatmap && !inDetail {
			a.heatmapMoveDown()
		} else if inDetail {
			if a.state.HeatmapDetailSel < a.heatmapDetailPodCount()-1 {
				a.state.HeatmapDetailSel++
			}
		} else {
			a.state.MoveSelection(1, a.activeLen())
		}

	case ui.ActionMoveLeft:
		if onHeatmap && !inDetail {
			a.heatmapMoveLeft()
		} else {
			a.gridMoveHorizontal(-1)
		}

	case ui.ActionMoveRight:
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

	case ui.ActionConfirm:
		if onHeatmap && !inDetail {
			if sel := a.state.Selection[model.TabHeatmap]; sel < len(a.state.Nodes) {
				a.state.HeatmapDetailNode = a.state.Nodes[sel].Name
				a.state.HeatmapDetailSel = 0
				a.state.HeatmapNodeDetail = true
			}
		}

	case ui.ActionSearchCancel:
		if inDetail {
			a.state.HeatmapNodeDetail = false
		} else {
			a.state.SetActiveSearchQuery("")
		}

	case ui.ActionRefresh:
		a.watcher.TriggerRefresh()

	case ui.ActionSearchOpen:
		if !onHeatmap {
			a.state.SearchMode = true
			a.state.SetActiveSearchQuery("")
		}

	case ui.ActionHelp:
		a.state.HelpMode = true

	case ui.ActionLogsOpen:
		if onHeatmap && !inDetail {
			a.heatmapMoveRight()
		} else if inDetail {
			a.handleLogsOpenForDetail()
		} else {
			a.handleLogsOpen()
		}

	case ui.ActionExecOpen:
		if inDetail {
			return a.handleExecOpenForDetail()
		}
		return a.handleExecOpen()

	case ui.ActionSwitchCluster:
		a.handleClusterPickerOpen()

	case ui.ActionSwitchNamespace:
		a.handleNamespacePickerOpen()

	case ui.ActionDescribe:
		a.handleDescribeOpen()
	}
	return nil
}

// ── Cluster management ───────────────────────────────────────────────────────

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

func (a *App) switchCluster(newContext string) {
	if err := k8s.PersistCurrentContext(a.cfg.Kubeconfig, newContext); err != nil {
		a.state.LastErr = fmt.Errorf("persist context: %w", err)
	}

	if newContext == a.cfg.Context {
		return
	}
	if a.watcherCancel != nil {
		a.watcherCancel()
	}
	if a.watcher != nil {
		a.watcher.Close()
	}
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
	a.closeLogs()
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
	a.cfg.Context = newContext
	a.state.Context = newContext
	a.state.Pods = nil
	a.state.Nodes = nil
	a.state.Namespaces = nil
	a.state.Deployments = nil
	a.state.Events = nil
	// Bump generation counters so cached views invalidate stale data.
	a.state.PodGeneration++
	a.state.NodeGeneration++
	a.state.EventGeneration++
	a.state.NodesLoaded = false
	a.state.PodsLoaded = false
}

// ── Logs management ──────────────────────────────────────────────────────────

func (a *App) openLogs(ns, pod, container string) {
	if a.logsCancel != nil {
		a.logsCancel()
		a.logsCancel = nil
	}

	logCtx, cancel := context.WithCancel(context.Background())
	a.logsCancel = cancel
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

func (a *App) closeLogs() {
	if a.logsCancel != nil {
		a.logsCancel()
		a.logsCancel = nil
	}
	a.logLines = nil
	a.state.LogsMode = false
	a.state.LogsLines = nil
}

func (a *App) handleLogsOpen() {
	switch a.state.ActiveTab {
	case model.TabPods:
		ns, pod, container := a.xrayView.SelectedRef(a.state.Selection[model.TabPods])
		if pod != "" {
			a.openLogs(ns, pod, container)
		}
	case model.TabNodeOverview:
		ns, pod := a.nodeOverview.SelectedPodRef(a.state.Selection[model.TabNodeOverview])
		if pod != "" {
			a.openLogs(ns, pod, "")
		}
	}
}

func (a *App) handleLogsOpenForDetail() {
	pods := views.NodeDetailPods(&a.state)
	if a.state.HeatmapDetailSel < len(pods) {
		p := pods[a.state.HeatmapDetailSel]
		a.openLogs(p.Namespace, p.Name, "")
	}
}

// ── Exec management ──────────────────────────────────────────────────────────

// execCmd wraps exec.Command to implement tea.ExecCommand while capturing stderr.
type execCmd struct {
	cmd       *exec.Cmd
	stderrBuf *strings.Builder
}

func (e *execCmd) Run() error         { return e.cmd.Run() }
func (e *execCmd) SetStdin(r io.Reader) { e.cmd.Stdin = r }
func (e *execCmd) SetStdout(w io.Writer) { e.cmd.Stdout = w }
func (e *execCmd) SetStderr(w io.Writer) {
	e.cmd.Stderr = io.MultiWriter(w, e.stderrBuf)
}

func (a *App) buildExecCmd(ns, pod, container string) tea.Cmd {
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

	var stderrBuf strings.Builder
	ec := &execCmd{
		cmd:       exec.Command("kubectl", args...),
		stderrBuf: &stderrBuf,
	}
	return tea.Exec(ec, func(err error) tea.Msg {
		return execDoneMsg{err: err, stderr: stderrBuf.String()}
	})
}

func (a *App) handleExecOpen() tea.Cmd {
	switch a.state.ActiveTab {
	case model.TabPods:
		ns, pod, container := a.xrayView.SelectedRef(a.state.Selection[model.TabPods])
		if pod != "" {
			return a.buildExecCmd(ns, pod, container)
		}
	case model.TabNodeOverview:
		ns, pod := a.nodeOverview.SelectedPodRef(a.state.Selection[model.TabNodeOverview])
		if pod != "" {
			return a.buildExecCmd(ns, pod, "")
		}
	}
	return nil
}

func (a *App) handleExecOpenForDetail() tea.Cmd {
	pods := views.NodeDetailPods(&a.state)
	if a.state.HeatmapDetailSel < len(pods) {
		p := pods[a.state.HeatmapDetailSel]
		return a.buildExecCmd(p.Namespace, p.Name, "")
	}
	return nil
}

// ── Describe ─────────────────────────────────────────────────────────────────

func (a *App) handleDescribeOpen() {
	kind, ns, name := a.describeSubject()
	if name == "" {
		return
	}
	args := []string{"describe", kind, name}
	if ns != "" {
		args = append(args, "-n", ns)
	}
	if a.cfg.Context != "" {
		args = append(args, "--context", a.cfg.Context)
	}
	if a.cfg.Kubeconfig != "" {
		args = append(args, "--kubeconfig", a.cfg.Kubeconfig)
	}
	out, err := exec.Command("kubectl", args...).CombinedOutput()
	if err != nil && len(out) == 0 {
		a.state.LastErr = fmt.Errorf("describe: %w", err)
		return
	}
	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	a.state.DescribeLines = lines
	a.state.DescribeOffset = 0
	a.state.DescribeTitle = kind + " " + name
	if ns != "" {
		a.state.DescribeTitle = kind + " " + ns + "/" + name
	}
	a.state.DescribeMode = true
}

func (a *App) describeSubject() (kind, ns, name string) {
	switch a.state.ActiveTab {
	case model.TabPods:
		n, pod, _ := a.xrayView.SelectedRef(a.state.Selection[model.TabPods])
		return "pod", n, pod
	case model.TabNodeOverview:
		k, n, nm := a.nodeOverview.SelectedRef(a.state.Selection[model.TabNodeOverview])
		return k, n, nm
	case model.TabHeatmap:
		if a.state.HeatmapNodeDetail {
			pods := views.NodeDetailPods(&a.state)
			if a.state.HeatmapDetailSel < len(pods) {
				p := pods[a.state.HeatmapDetailSel]
				return "pod", p.Namespace, p.Name
			}
		} else {
			if sel := a.state.Selection[model.TabHeatmap]; sel < len(a.state.Nodes) {
				return "node", "", a.state.Nodes[sel].Name
			}
		}
	case model.TabDeployments:
		n, nm := a.deploymentsView.SelectedRef(a.state.Selection[model.TabDeployments])
		return "deployment", n, nm
	case model.TabNamespaces:
		nm := a.namespacesView.SelectedRef(a.state.Selection[model.TabNamespaces])
		return "namespace", "", nm
	}
	return "", "", ""
}

// ── Navigation helpers ───────────────────────────────────────────────────────

func (a *App) activeLen() int {
	switch a.state.ActiveTab {
	case model.TabNodeOverview:
		return a.nodeOverview.RowCount()
	case model.TabPods:
		return a.xrayView.RowCount()
	case model.TabNamespaces:
		return a.namespacesView.RowCount()
	case model.TabDeployments:
		return a.deploymentsView.RowCount()
	case model.TabEvents:
		return a.eventsView.RowCount()
	case model.TabHeatmap:
		if a.state.HeatmapNodeDetail {
			return a.heatmapDetailPodCount()
		}
		return len(a.state.Nodes)
	}
	return 0
}

func (a *App) heatmapDetailPodCount() int {
	count := 0
	for _, p := range a.state.Pods {
		if p.Node == a.state.HeatmapDetailNode {
			count++
		}
	}
	return count
}

func (a *App) gridMoveHorizontal(dir int) {
	if a.state.ActiveTab != model.TabNodeOverview {
		return
	}
	sel := a.state.Selection[model.TabNodeOverview]
	col := sel % 2
	if dir == -1 && col == 0 {
		return
	}
	if dir == 1 && col == 1 {
		return
	}
	a.state.MoveSelection(dir, a.activeLen())
}

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

// ── Startup context picker ───────────────────────────────────────────────────

// contextPickerModel is a mini tea.Model used for the startup context picker.
type contextPickerModel struct {
	contexts []string
	current  string
	sel      int
	chosen   string
	quit     bool
	width    int
	height   int
}

func (m contextPickerModel) Init() tea.Cmd { return nil }

func (m contextPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEscape:
			m.quit = true
			return m, tea.Quit
		case tea.KeyEnter:
			if m.sel < len(m.contexts) {
				m.chosen = m.contexts[m.sel]
			}
			return m, tea.Quit
		case tea.KeyUp:
			if m.sel > 0 {
				m.sel--
			}
		case tea.KeyDown:
			if m.sel < len(m.contexts)-1 {
				m.sel++
			}
		}
		switch msg.String() {
		case "q", "Q":
			m.quit = true
			return m, tea.Quit
		case "k":
			if m.sel > 0 {
				m.sel--
			}
		case "j":
			if m.sel < len(m.contexts)-1 {
				m.sel++
			}
		}
	}
	return m, nil
}

func (m contextPickerModel) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	state := &model.AppState{
		ClusterPickerList: m.contexts,
		ClusterPickerCurr: m.current,
		ClusterPickerSel:  m.sel,
	}
	overlay := ui.RenderClusterPicker(state, m.width, m.height)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		overlay,
		lipgloss.WithWhitespaceBackground(ui.StyleDefault.GetBackground()),
	)
}

// RunContextPicker runs a mini tea.Program to pick a Kubernetes context.
// Returns the chosen context name and whether the user quit.
func RunContextPicker(contexts []string, current string) (string, bool) {
	sel := 0
	for i, c := range contexts {
		if c == current {
			sel = i
			break
		}
	}
	m := contextPickerModel{
		contexts: contexts,
		current:  current,
		sel:      sel,
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		return "", true
	}
	final := result.(contextPickerModel)
	if final.quit {
		return "", true
	}
	return final.chosen, false
}
