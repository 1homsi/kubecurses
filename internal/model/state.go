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
	// Incremented each time Nodes is replaced — used by views to detect stale caches.
	NodeGeneration uint64

	// Logs wrapping cache — lazily populated by DrawLogsView / scroll handlers.
	// Both width and line count must match for the cache to be valid.
	LogsWrapped   []string
	LogsWrapWidth int
	LogsWrapCount int // len(LogsLines) when LogsWrapped was computed

	// Exec overlay state — active when ExecMode is true.
	ExecMode       bool
	ExecNamespace  string
	ExecPod        string
	ExecContainer  string   // empty = first container / Kubernetes default
	ExecLines      []string // buffered command output
	ExecAutoScroll bool
	ExecOffset     int // manual scroll position (ignored when ExecAutoScroll is on)

	// Exec wrapping cache — same pattern as logs cache.
	ExecWrapped   []string
	ExecWrapWidth int
	ExecWrapCount int // len(ExecLines) when ExecWrapped was computed

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
		s.NodeGeneration++
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

// HeatmapPlanRowsMin returns a symmetric hex-like row plan for n items.
// The returned plan always sums to exactly n, keeps adjacent row deltas small
// (|delta| <= 1), and prefers a compact honeycomb silhouette over a pyramid.
// minA is a soft hint for preferred narrow-row width (not a hard constraint).
func HeatmapPlanRowsMin(n, maxCols, minA int) []int {
	if n <= 0 {
		return nil
	}
	if maxCols < 1 {
		maxCols = 1
	}
	if minA < 1 {
		minA = 1
	}

	minRows := (n + maxCols - 1) / maxCols
	if minRows < 1 {
		minRows = 1
	}
	// Search a compact window first; fall back to wider if needed.
	maxRows := minRows + 8
	if maxRows > n {
		maxRows = n
	}

	targetRows := minRows + 1
	if targetRows > n {
		targetRows = n
	}

	bestScore := int(^uint(0) >> 1) // max int
	var bestPlan []int

	tryRows := func(from, to int) {
		for rows := from; rows <= to; rows++ {
			plan, ok := heatmapBuildBalancedPlan(n, rows, maxCols)
			if !ok || len(plan) == 0 {
				continue
			}
			edge := plan[0]
			if n > 1 && edge == 1 {
				continue // avoid orphan shoulders where possible
			}

			peak := plan[0]
			for _, v := range plan {
				if v > peak {
					peak = v
				}
			}
			centerWidth := 0
			midL := (len(plan) - 1) / 2
			midR := len(plan) / 2
			for i := midL; i >= 0 && plan[i] == peak; i-- {
				centerWidth++
			}
			for i := midR + 1; i < len(plan) && plan[i] == peak; i++ {
				centerWidth++
			}

			flat := 0
			if len(plan) > 1 && peak == edge {
				flat = 1
			}

			softMinPenalty := 0
			if edge < minA {
				softMinPenalty = minA - edge
			}

			deltaPenalty := 0
			for i := 1; i < len(plan); i++ {
				d := plan[i] - plan[i-1]
				if d < 0 {
					d = -d
				}
				if d > 1 {
					deltaPenalty += (d - 1) * 20
				}
			}

			score := 0
			score += flat * 300
			score += (peak - edge) * 10
			if centerWidth >= 2 {
				// Prefer a short center plateau for smoother shoulders.
				score += (centerWidth - 2) * 4
			} else {
				score += 8
			}
			if len(plan) > targetRows {
				score += (len(plan) - targetRows) * 3
			} else {
				score += (targetRows - len(plan)) * 6
			}
			score += softMinPenalty * 2
			score += deltaPenalty

			if bestPlan == nil || score < bestScore {
				bestScore = score
				bestPlan = plan
			}
		}
	}

	tryRows(minRows, maxRows)
	if bestPlan == nil {
		// Broader fallback for pathological cases.
		tryRows(1, n)
	}
	if bestPlan != nil {
		return bestPlan
	}
	return []int{n}
}

// HeatmapPlanRows returns a symmetric hex-cluster row plan for n nodes.
// The plan is a slice of per-row node counts [a, a+1, …, c, …, a+1, a] that
// forms a hexagonal-cluster silhouette. The widest row is at most maxCols.
// For very large n (exceeding any pure symmetric shape), additional center rows
// of width maxCols are inserted between the two tapering halves.
func HeatmapPlanRows(n, maxCols int) []int {
	return HeatmapPlanRowsMin(n, maxCols, 1)
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

// heatmapBuildBalancedPlan creates an exact-sum symmetric plan with a fixed row
// count and row deltas constrained to <= 1.
func heatmapBuildBalancedPlan(n, rows, maxCols int) ([]int, bool) {
	if n <= 0 || rows <= 0 || rows > n || maxCols < 1 {
		return nil, false
	}
	base := n / rows
	if base < 1 || base > maxCols {
		return nil, false
	}
	rem := n - base*rows
	plan := make([]int, rows)
	for i := range plan {
		plan[i] = base
	}

	for rem > 0 {
		progressed := false

		// Center row first (odd row counts only).
		if rows%2 == 1 {
			c := rows / 2
			if rem > 0 && heatmapCanIncSym(plan, c, c, maxCols) {
				plan[c]++
				rem--
				progressed = true
				if rem == 0 {
					break
				}
			}
		}

		// Then grow shoulders from the center outwards in mirrored pairs.
		maxDist := rows / 2
		for d := 0; d < maxDist && rem >= 2; d++ {
			l := (rows-1)/2 - d
			r := rows/2 + d
			if l == r {
				continue
			}
			if heatmapCanIncSym(plan, l, r, maxCols) {
				plan[l]++
				plan[r]++
				rem -= 2
				progressed = true
			}
		}

		if !progressed {
			return nil, false
		}
	}

	for i := 1; i < len(plan); i++ {
		d := plan[i] - plan[i-1]
		if d < 0 {
			d = -d
		}
		if d > 1 {
			return nil, false
		}
		if plan[i] > maxCols {
			return nil, false
		}
	}
	if plan[0] < 1 || plan[len(plan)-1] < 1 {
		return nil, false
	}
	return plan, true
}

func heatmapCanIncSym(plan []int, l, r, maxCols int) bool {
	if l < 0 || r < 0 || l >= len(plan) || r >= len(plan) {
		return false
	}
	tmp := make([]int, len(plan))
	copy(tmp, plan)

	tmp[l]++
	if l != r {
		tmp[r]++
	}
	if tmp[l] > maxCols || tmp[r] > maxCols {
		return false
	}
	// Preserve smooth shoulders.
	for i := 1; i < len(tmp); i++ {
		d := tmp[i] - tmp[i-1]
		if d < 0 {
			d = -d
		}
		if d > 1 {
			return false
		}
	}
	// Preserve symmetry.
	for i := 0; i < len(tmp)/2; i++ {
		if tmp[i] != tmp[len(tmp)-1-i] {
			return false
		}
	}
	return true
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
