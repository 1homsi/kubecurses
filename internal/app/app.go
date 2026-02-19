package app

import (
	"context"
	"errors"
	"fmt"
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

// App is the top-level application struct.
type App struct {
	cfg          Config
	screen       *ui.Screen
	state        model.AppState
	watcher      *k8s.Watcher
	views        [4]views.View
	nodeOverview rowCounter // kept for RowCount access; swap impl to change overview style
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
		}
	}

	cs, err := k8s.NewClient(cfg.Kubeconfig, cfg.Context)
	if err != nil {
		scr.Fini()
		return nil, fmt.Errorf("kubernetes client: %w", err)
	}

	watcher := k8s.NewWatcher(cs, cfg.Namespace)
	ov := &views.NodeOverviewView{}

	app := &App{
		cfg:          cfg,
		screen:       scr,
		watcher:      watcher,
		nodeOverview: ov,
		state: model.AppState{
			Namespace: cfg.Namespace,
			Context:   cfg.Context,
		},
		views: [4]views.View{
			model.TabNodeOverview: ov,
			model.TabPods:         &views.PodsView{},
			model.TabDeployments:  &views.DeploymentsView{},
			model.TabNamespaces:   &views.NamespacesView{},
		},
	}
	return app, nil
}

// Run starts the watcher goroutines and enters the main event loop.
// It returns when the user quits or an unrecoverable error occurs.
func (a *App) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	a.watcher.Start(ctx, a.cfg.PollInterval)

	ticker := time.NewTicker(a.cfg.PollInterval)
	defer ticker.Stop()

	a.draw()

	for {
		select {
		case <-ctx.Done():
			a.screen.Fini()
			return ctx.Err()

		case update := <-a.watcher.Updates():
			a.state.ApplyUpdate(update)
			a.draw()

		case <-ticker.C:
			a.watcher.TriggerRefresh()

		case tcellEv := <-a.pollEvent(ctx):
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

// pollEvent returns a channel that yields the next tcell event.
func (a *App) pollEvent(ctx context.Context) <-chan tcell.Event {
	ch := make(chan tcell.Event, 1)
	go func() {
		ev := a.screen.PollEvent()
		select {
		case ch <- ev:
		case <-ctx.Done():
		}
	}()
	return ch
}

// handleEvent dispatches to help, search, or normal mode event handling.
// Returns true if the application should quit.
func (a *App) handleEvent(ev tcell.Event) bool {
	// Any key dismisses the help overlay.
	if a.state.HelpMode {
		a.state.HelpMode = false
		return false
	}
	if a.state.SearchMode {
		return a.applySearchAction(ui.SearchEventToAction(ev), ev)
	}
	return a.applyAction(ui.EventToAction(ev))
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
	switch action {
	case ui.ActionQuit:
		return true
	case ui.ActionNextTab:
		a.state.NextTab()
	case ui.ActionPrevTab:
		a.state.PrevTab()
	case ui.ActionTab1:
		a.state.SetTab(model.TabNodeOverview)
	case ui.ActionTab2:
		a.state.SetTab(model.TabPods)
	case ui.ActionTab3:
		a.state.SetTab(model.TabDeployments)
	case ui.ActionTab4:
		a.state.SetTab(model.TabNamespaces)
	case ui.ActionMoveUp:
		a.state.MoveSelection(a.upDelta(), a.activeLen())
	case ui.ActionMoveDown:
		a.state.MoveSelection(a.downDelta(), a.activeLen())
	case ui.ActionMoveLeft:
		a.gridMoveHorizontal(-1)
	case ui.ActionMoveRight:
		a.gridMoveHorizontal(1)
	case ui.ActionPageUp:
		a.state.MoveSelection(-4, a.activeLen())
	case ui.ActionPageDown:
		a.state.MoveSelection(4, a.activeLen())
	case ui.ActionRefresh:
		a.watcher.TriggerRefresh()
	case ui.ActionSearchOpen:
		a.state.SearchMode = true
		a.state.SearchQuery = ""
	case ui.ActionSearchCancel:
		a.state.SearchQuery = ""
	case ui.ActionHelp:
		a.state.HelpMode = true
	}
	return false
}

func (a *App) upDelta() int   { return -1 }
func (a *App) downDelta() int { return 1 }

// gridMoveHorizontal moves the selected node card left (dir=-1) or right
// (dir=+1) within the 2-column grid. Ignores movement that would wrap across
// row boundaries.
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

// activeLen returns the number of navigable rows in the currently active view.
func (a *App) activeLen() int {
	switch a.state.ActiveTab {
	case model.TabNodeOverview:
		// NodeOverviewView knows its actual row count after the last draw.
		return a.nodeOverview.RowCount()
	case model.TabPods:
		return len(a.state.Pods)
	case model.TabNamespaces:
		return len(a.state.Namespaces)
	case model.TabDeployments:
		return len(a.state.Deployments)
	}
	return 0
}

// draw renders the entire screen.
func (a *App) draw() {
	w, h := a.screen.Size()
	a.screen.Clear()
	ui.DrawTabBar(a.screen, w, a.state.ActiveTab, &a.state)
	contentRect := ui.ContentRect(w, h)
	a.views[a.state.ActiveTab].Draw(a.screen, contentRect, &a.state)
	ui.DrawStatusBar(a.screen, w, h, &a.state)
	if a.state.HelpMode {
		ui.DrawHelp(a.screen, w, h)
	}
	a.screen.Show()
}
