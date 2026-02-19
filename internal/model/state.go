package model

// Tab identifies which resource view is active.
type Tab int

const (
	TabNodeOverview Tab = iota // nodes with pods grouped — the main view
	TabPods                    // flat pod list
	TabDeployments
	TabNamespaces
	tabCount // sentinel — keep last
)

// TabNames maps Tab constants to display strings.
var TabNames = [tabCount]string{
	TabNodeOverview: "Overview",
	TabPods:         "Pods",
	TabDeployments:  "Deployments",
	TabNamespaces:   "Namespaces",
}

// AppState is the single source of truth for all mutable UI state.
// It lives exclusively in the main goroutine — no mutex needed.
type AppState struct {
	ActiveTab   Tab
	Selection   [tabCount]int // selected row per tab
	Pods        []Pod
	Nodes       []Node
	Namespaces  []Namespace
	Deployments []Deployment
	LastErr     error
	Namespace   string // active namespace filter ("" = all)
	SearchMode  bool   // true while the user is typing a search query
	SearchQuery string // live filter applied to the active view
}

// ApplyUpdate merges an incoming watcher update into state.
func (s *AppState) ApplyUpdate(u Update) {
	if u.Err != nil {
		s.LastErr = u.Err
		return
	}
	s.LastErr = nil
	switch u.Kind {
	case UpdatePods:
		s.Pods = u.Pods
		s.clampSelection(TabPods, len(s.Pods))
	case UpdateNodes:
		s.Nodes = u.Nodes
		// TabNodeOverview rows = nodes + their pods; clamped by activeLen in app
	case UpdateNamespaces:
		s.Namespaces = u.Namespaces
		s.clampSelection(TabNamespaces, len(s.Namespaces))
	case UpdateDeployments:
		s.Deployments = u.Deployments
		s.clampSelection(TabDeployments, len(s.Deployments))
	}
}

// clampSelection ensures the selection index for a tab stays in bounds.
func (s *AppState) clampSelection(t Tab, length int) {
	if length == 0 {
		s.Selection[t] = 0
		return
	}
	if s.Selection[t] >= length {
		s.Selection[t] = length - 1
	}
}

// MoveSelection moves the selection for the active tab by delta rows,
// clamped to [0, length).
func (s *AppState) MoveSelection(delta, length int) {
	if length == 0 {
		return
	}
	sel := s.Selection[s.ActiveTab] + delta
	if sel < 0 {
		sel = 0
	}
	if sel >= length {
		sel = length - 1
	}
	s.Selection[s.ActiveTab] = sel
}

// NextTab advances to the next tab (wraps around).
func (s *AppState) NextTab() {
	s.ActiveTab = (s.ActiveTab + 1) % tabCount
}

// PrevTab goes to the previous tab (wraps around).
func (s *AppState) PrevTab() {
	s.ActiveTab = (s.ActiveTab + tabCount - 1) % tabCount
}

// SetTab sets a specific tab (no-op if out of range).
func (s *AppState) SetTab(t Tab) {
	if t >= 0 && t < tabCount {
		s.ActiveTab = t
	}
}
