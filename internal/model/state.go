package model

// Tab identifies which resource view is active.
type Tab int

const (
	TabHeatmap      Tab = iota // per-node honeycomb grid — first tab
	TabNodeOverview            // nodes with pods grouped
	TabPods                    // flat pod list (Xray)
	TabDeployments
	TabNamespaces
	tabCount // sentinel — keep last
)

// TabNames maps Tab constants to display strings.
var TabNames = [tabCount]string{
	TabHeatmap:      "Heatmap",
	TabNodeOverview: "Overview",
	TabPods:         "Xray",
	TabDeployments:  "Deployments",
	TabNamespaces:   "Namespaces",
}

// AppState is the single source of truth for all mutable UI state.
// It lives exclusively in the main goroutine — no mutex needed.
type AppState struct {
	ActiveTab   Tab
	Selection   [tabCount]int // selected row per tab
	NoIcons     bool          // when true, views use text fallbacks instead of icons
	Pods        []Pod
	Nodes       []Node
	Namespaces  []Namespace
	Deployments []Deployment
	LastErr     error
	Namespace   string // active namespace filter ("" = all)
	Context     string // active kubernetes context name
	SearchMode  bool   // true while the user is typing a search query
	SearchQuery string // live filter applied to the active view
	HelpMode    bool   // true while the help overlay is shown

	// Logs overlay state — active when LogsMode is true.
	LogsMode       bool
	LogsNamespace  string
	LogsPod        string
	LogsContainer  string   // empty = first container / all
	LogsLines      []string // buffered log output
	LogsAutoScroll bool     // follow the tail of the log
	LogsOffset     int      // manual scroll position (ignored when AutoScroll is on)

	// Cluster picker overlay state — active when ClusterPickerMode is true.
	ClusterPickerMode bool
	ClusterPickerList []string
	ClusterPickerSel  int
	ClusterPickerCurr string // currently connected context

	// Incremented each time Pods is replaced — used by views to detect stale caches.
	PodGeneration uint64

	// Heatmap navigation state.
	NodesLoaded       bool   // true once the first nodes update has been received
	PodsLoaded        bool   // true once the first pods update has been received
	HeatmapCols       int    // grid column count — written by HeatmapView.Draw each frame
	HeatmapScroll     int    // first visible box-row index
	HeatmapNodeDetail bool   // true = node-detail overlay is active
	HeatmapDetailNode string // name of the node being detailed
	HeatmapDetailSel  int    // selected pod index within the node-detail view
	HeatmapRowPlan    []int  // symmetric row-width plan written by HeatmapView.Draw each frame
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
		s.PodGeneration++
		s.PodsLoaded = true
		s.clampSelection(TabPods, len(s.Pods))
	case UpdateNodes:
		s.Nodes = u.Nodes
		s.NodesLoaded = true
		s.clampSelection(TabHeatmap, len(s.Nodes))
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

// ── Heatmap honeycomb layout helpers ──────────────────────────────────────────

// HeatmapRowCols returns the number of nodes in box-row rowIdx.
// Even rows have cols nodes; odd rows have cols-1 (minimum 1).
func HeatmapRowCols(rowIdx, cols int) int {
	if rowIdx%2 == 1 {
		c := cols - 1
		if c < 1 {
			c = 1
		}
		return c
	}
	return cols
}

// HeatmapNodeToRowCol converts a flat node index to (boxRow, colInRow).
func HeatmapNodeToRowCol(nodeIdx, cols int) (row, col int) {
	remaining := nodeIdx
	row = 0
	for {
		n := HeatmapRowCols(row, cols)
		if remaining < n {
			return row, remaining
		}
		remaining -= n
		row++
	}
}

// HeatmapRowColToNode converts (boxRow, colInRow) to a flat node index.
func HeatmapRowColToNode(row, col, cols int) int {
	idx := 0
	for r := 0; r < row; r++ {
		idx += HeatmapRowCols(r, cols)
	}
	return idx + col
}

// ── Symmetric hex-cluster plan helpers ────────────────────────────────────────

// HeatmapPlanRows returns a symmetric hex-cluster row plan for n nodes.
// The plan is a slice of per-row node counts [a, a+1, …, c, …, a+1, a] that
// forms a hexagonal-cluster silhouette. The widest row is at most maxCols.
// For very large n (exceeding any pure symmetric shape), additional center rows
// of width maxCols are inserted between the two tapering halves.
func HeatmapPlanRows(n, maxCols int) []int {
	if n <= 0 {
		return nil
	}
	if maxCols < 1 {
		maxCols = 1
	}

	// Find the symmetric shape [a, a+1, …, c, …, a+1, a] with minimum waste.
	// Tiebreak on fewer rows (larger a relative to c = less taper = flatter shape).
	bestWaste := -1
	bestRows := -1
	bestA, bestC := 1, 1
	for c := 1; c <= maxCols; c++ {
		// Iterate a from c down to 1; stop at the first a where cap >= n.
		for a := c; a >= 1; a-- {
			cap := heatmapHexCap(a, c)
			if cap >= n {
				waste := cap - n
				rows := 2*(c-a) + 1
				if bestWaste < 0 || waste < bestWaste || (waste == bestWaste && rows < bestRows) {
					bestWaste = waste
					bestRows = rows
					bestA, bestC = a, c
				}
				break
			}
		}
		if bestWaste == 0 && bestRows == 1 {
			break // single-row perfect fit — can't do better
		}
	}

	if bestWaste >= 0 {
		return heatmapBuildPlan(bestA, bestC, 0)
	}

	// n exceeds the capacity of any pure symmetric shape at maxCols.
	// Extend by inserting extra full-width center rows.
	pureCap := heatmapHexCap(1, maxCols)
	extra := (n - pureCap + maxCols - 1) / maxCols
	return heatmapBuildPlan(1, maxCols, extra)
}

// heatmapHexCap returns the total node capacity of shape [a, a+1, …, c, …, a+1, a].
func heatmapHexCap(a, c int) int {
	return (c-a)*(c+a-1) + c
}

// heatmapBuildPlan constructs [a, …, c, (extra × c), …, a].
func heatmapBuildPlan(a, c, extra int) []int {
	steps := c - a
	rows := make([]int, 0, 2*steps+1+extra)
	for w := a; w <= c; w++ {
		rows = append(rows, w)
	}
	for i := 0; i < extra; i++ {
		rows = append(rows, c)
	}
	for w := c - 1; w >= a; w-- {
		rows = append(rows, w)
	}
	return rows
}

// HeatmapNodeToRowColPlan converts a flat node index to (row, col) using a plan.
func HeatmapNodeToRowColPlan(plan []int, nodeIdx int) (row, col int) {
	for r, rowCols := range plan {
		if nodeIdx < rowCols {
			return r, nodeIdx
		}
		nodeIdx -= rowCols
	}
	// Clamp to last valid position.
	if len(plan) > 0 {
		return len(plan) - 1, plan[len(plan)-1] - 1
	}
	return 0, 0
}

// HeatmapRowColToNodePlan converts (row, col) to a flat node index using a plan.
func HeatmapRowColToNodePlan(plan []int, row, col int) int {
	idx := 0
	for r := 0; r < row && r < len(plan); r++ {
		idx += plan[r]
	}
	return idx + col
}
