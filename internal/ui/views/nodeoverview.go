package views

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"

	"github.com/1homsi/kubecurses/internal/model"
	"github.com/1homsi/kubecurses/internal/ui"
)

// rowKind distinguishes display row types.
type rowKind int

const (
	rkPod     rowKind = iota
	rkNode            // node section header
	rkReason         // pending-pod explainer sub-row
	rkWarning         // scheduling imbalance banner
)

// ovRow is a flat display row.
type ovRow struct {
	kind     rowKind
	node     model.Node
	pod      model.Pod
	podCount int    // for rkNode: pods scheduled on this node
	text     string // for rkReason / rkWarning
}

// ovCacheKey groups all inputs that affect the row model.
type ovCacheKey struct {
	podGen  uint64
	nodeGen uint64
	query   string
	ns      string
}

// NodeOverviewView renders nodes as section headers with their pods nested below.
type NodeOverviewView struct {
	rows         []ovRow
	scrollOffset int
	cacheValid   bool
	cacheKey     ovCacheKey
	cachedRows   []ovRow
}

// dynCols computes dynamic column widths from row width w.
// Right block: STATUS(10)+READY(6)+REST(5)+AGE(11) = 32 chars, right-anchored.
// Remaining space split 2:1 between NAME and NAMESPACE/VERSION columns.
func dynCols(w int) (nameW, nsW, statusAt int) {
	statusAt = w - 32
	if statusAt < 40 {
		statusAt = 40
	}
	avail := statusAt - 4 // 4-char indent (icon + spaces)
	nameW = avail * 2 / 3
	if nameW < 20 {
		nameW = 20
	}
	nsW = avail - nameW
	if nsW < 10 {
		nsW = 10
		nameW = avail - nsW
	}
	return
}

// buildRows groups pods under their node, injects warning/reason rows.
func (v *NodeOverviewView) buildRows(state *model.AppState, query string) []ovRow {
	byNode := make(map[string][]model.Pod, len(state.Nodes))
	var unscheduled []model.Pod

	for _, p := range state.Pods {
		if state.Namespace != "" && p.Namespace != state.Namespace {
			continue
		}
		if query != "" && !podMatchesQuery(p, query) {
			continue
		}
		if p.Node == "" {
			unscheduled = append(unscheduled, p)
		} else {
			byNode[p.Node] = append(byNode[p.Node], p)
		}
	}

	// Sort: NotReady first, then by pod count descending.
	nodes := make([]model.Node, len(state.Nodes))
	copy(nodes, state.Nodes)
	sort.Slice(nodes, func(i, j int) bool {
		iReady := nodes[i].Status == "Ready"
		jReady := nodes[j].Status == "Ready"
		if iReady != jReady {
			return !iReady
		}
		return len(byNode[nodes[i].Name]) > len(byNode[nodes[j].Name])
	})

	// ── scheduling imbalance detection ──────────────────────────────────
	var warningRows []ovRow
	if len(nodes) > 1 {
		totalPods := 0
		maxPods := 0
		heaviest := ""
		for _, n := range nodes {
			c := len(byNode[n.Name])
			totalPods += c
			if c > maxPods {
				maxPods = c
				heaviest = n.Name
			}
		}
		if totalPods > 0 && maxPods > 0 {
			avg := totalPods / len(nodes)
			if avg > 0 && maxPods >= avg*2 {
				pct := maxPods * 100 / totalPods
				msg := fmt.Sprintf(
					"⚠  Scheduling imbalance: %s has %d%% of pods (%d/%d total)",
					truncate(heaviest, 30), pct, maxPods, totalPods,
				)
				warningRows = append(warningRows, ovRow{kind: rkWarning, text: msg})
			}
		}
	}

	rows := make([]ovRow, 0, len(nodes)+len(state.Pods))
	for _, n := range nodes {
		pods := byNode[n.Name]
		if query != "" && len(pods) == 0 && !nodeMatchesQuery(n, query) {
			continue
		}
		rows = append(rows, ovRow{kind: rkNode, node: n, podCount: len(pods)})
		for _, p := range pods {
			rows = append(rows, ovRow{kind: rkPod, pod: p})
			// ── pending pod explainer ────────────────────────────────────
			if p.Status == "Pending" && p.PendingReason != "" {
				rows = append(rows, ovRow{kind: rkReason, text: p.PendingReason})
			}
		}
	}

	if len(unscheduled) > 0 {
		rows = append(rows, ovRow{
			kind: rkNode,
			node: model.Node{Name: "<unscheduled>", Status: "Unknown"},
		})
		for _, p := range unscheduled {
			rows = append(rows, ovRow{kind: rkPod, pod: p})
			if p.Status == "Pending" && p.PendingReason != "" {
				rows = append(rows, ovRow{kind: rkReason, text: p.PendingReason})
			}
		}
	}

	return append(warningRows, rows...)
}

func (v *NodeOverviewView) Draw(s *ui.Screen, r ui.Rect, state *model.AppState) {
	key := ovCacheKey{state.PodGeneration, state.NodeGeneration, state.SearchQuery, state.Namespace}
	if !v.cacheValid || key != v.cacheKey {
		v.cachedRows = v.buildRows(state, state.SearchQuery)
		v.cacheKey = key
		v.cacheValid = true
	}
	v.rows = v.cachedRows
	sel := state.Selection[model.TabNodeOverview]

	if sel >= len(v.rows) && len(v.rows) > 0 {
		sel = len(v.rows) - 1
		state.Selection[model.TabNodeOverview] = sel
	}

	v.drawHeader(s, r.X, r.Y, r.W)
	content := ui.Rect{X: r.X, Y: r.Y + 1, W: r.W, H: r.H - 1}

	if len(v.rows) > 0 {
		if sel < v.scrollOffset {
			v.scrollOffset = sel
		}
		if sel >= v.scrollOffset+content.H {
			v.scrollOffset = sel - content.H + 1
		}
		if v.scrollOffset < 0 {
			v.scrollOffset = 0
		}
	}

	for i := 0; i < content.H; i++ {
		rowIdx := v.scrollOffset + i
		y := content.Y + i
		if rowIdx >= len(v.rows) {
			s.FillRect(ui.Rect{X: content.X, Y: y, W: content.W, H: 1}, ' ', ui.StyleDefault)
			continue
		}
		v.drawRow(s, content.X, y, content.W, v.rows[rowIdx], rowIdx == sel)
	}
}

// drawHeader renders the sticky column-label row with dynamic column widths.
func (v *NodeOverviewView) drawHeader(s *ui.Screen, x, y, w int) {
	nameW, nsW, statusAt := dynCols(w)
	readyAt := statusAt + 10
	restAt := readyAt + 6
	ageAt := restAt + 5

	s.FillRect(ui.Rect{X: x, Y: y, W: w, H: 1}, ' ', ui.StyleHeader)
	s.DrawText(x+4,        y, ui.StyleHeader, fmt.Sprintf("%-*s", nameW, "NAME"))
	s.DrawText(x+4+nameW,  y, ui.StyleHeader, fmt.Sprintf("%-*s", nsW, "NAMESPACE"))
	s.DrawText(x+statusAt, y, ui.StyleHeader, fmt.Sprintf("%-10s", "STATUS"))
	s.DrawText(x+readyAt,  y, ui.StyleHeader, fmt.Sprintf("%-6s", "READY"))
	s.DrawText(x+restAt,   y, ui.StyleHeader, fmt.Sprintf("%-5s", "REST"))
	s.DrawText(x+ageAt,    y, ui.StyleHeader, "AGE")
}

// RowCount returns the current number of display rows.
func (v *NodeOverviewView) RowCount() int { return len(v.rows) }

// SelectedPodRef returns the namespace and pod name for the row at idx.
// Returns ("", "") when idx is out of range or the row is not a pod row.
func (v *NodeOverviewView) SelectedPodRef(idx int) (ns, pod string) {
	if idx < 0 || idx >= len(v.rows) {
		return "", ""
	}
	row := v.rows[idx]
	if row.kind != rkPod {
		return "", ""
	}
	return row.pod.Namespace, row.pod.Name
}

func (v *NodeOverviewView) drawRow(s *ui.Screen, x, y, w int, row ovRow, selected bool) {
	switch row.kind {
	case rkNode:
		v.drawNodeRow(s, x, y, w, row.node, row.podCount, selected)
	case rkReason:
		v.drawReasonRow(s, x, y, w, row.text)
	case rkWarning:
		v.drawWarningRow(s, x, y, w, row.text)
	default:
		v.drawPodRow(s, x, y, w, row.pod, selected)
	}
}

// drawNodeRow renders a node section header aligned to the same column grid as pod rows.
// When metrics-server data is available, the NAMESPACE column shows cpu/mem/pods;
// otherwise it shows the k8s version.
func (v *NodeOverviewView) drawNodeRow(s *ui.Screen, x, y, w int, n model.Node, podCount int, selected bool) {
	nameW, nsW, statusAt := dynCols(w)
	ageAt := statusAt + 21 // +10 status +6 ready +5 rest

	base := ui.StyleNodeHeader
	dotStyle := ui.StyleNodeReadyDot
	nameStyle := ui.StyleNodeName
	metaStyle := ui.StyleNodeMeta
	statusStyle := ui.StyleNodeReadyDot
	if n.Status != "Ready" {
		dotStyle = ui.StyleNodeNotReadyDot
		statusStyle = ui.StyleNodeNotReadyDot
	}
	if selected {
		base = ui.StyleSelected
		dotStyle = ui.StyleSelected
		nameStyle = ui.StyleSelected
		metaStyle = ui.StyleSelected
		statusStyle = ui.StyleSelected
	}
	s.FillRect(ui.Rect{X: x, Y: y, W: w, H: 1}, ' ', base)
	s.DrawText(x,          y, dotStyle,    "●")
	s.DrawText(x+4,        y, nameStyle,   fmt.Sprintf("%-*s", nameW, truncate(n.Name, nameW)))
	s.DrawText(x+statusAt, y, statusStyle, fmt.Sprintf("%-10s", n.Status))
	s.DrawText(x+ageAt,    y, metaStyle,   formatDuration(n.Age))

	// NAMESPACE column: metrics when available, version otherwise.
	if n.MetricsOK && n.AllocCPUm > 0 {
		cpuPct := int(n.UsedCPUm * 100 / n.AllocCPUm)
		memPct := int(n.UsedMemMi * 100 / n.AllocMemMi)

		cpuStyle, memStyle := metaStyle, metaStyle
		if !selected {
			cpuStyle = metricStyle(cpuPct)
			memStyle = metricStyle(memPct)
		}

		col := x + 4 + nameW
		cpu := fmt.Sprintf("cpu:%d%% ", cpuPct)
		mem := fmt.Sprintf("mem:%d%% ", memPct)
		s.DrawText(col, y, cpuStyle, truncate(cpu, nsW/2))
		col += len([]rune(cpu))
		if col < x+4+nameW+nsW {
			s.DrawText(col, y, memStyle, truncate(mem, (x+4+nameW+nsW)-col))
		}
		col += len([]rune(mem))
		if n.AllocPods > 0 && col < x+4+nameW+nsW {
			pods := fmt.Sprintf("%d/%d pods", podCount, n.AllocPods)
			s.DrawText(col, y, metaStyle, truncate(pods, (x+4+nameW+nsW)-col))
		}
	} else {
		s.DrawText(x+4+nameW, y, metaStyle, fmt.Sprintf("%-*s", nsW, truncate(n.Version, nsW)))
	}
}

// metricStyle returns a colour-coded style for a percentage value.
func metricStyle(pct int) tcell.Style {
	switch {
	case pct >= 85:
		return ui.StyleMetricsCrit
	case pct >= 70:
		return ui.StyleMetricsWarn
	}
	return ui.StyleMetricsOK
}

// drawReasonRow renders a pending-pod explainer sub-row.
func (v *NodeOverviewView) drawReasonRow(s *ui.Screen, x, y, w int, reason string) {
	s.FillRect(ui.Rect{X: x, Y: y, W: w, H: 1}, ' ', ui.StyleDefault)
	s.DrawText(x+4, y, ui.StylePendingReason, truncate("→ "+reason, w-6))
}

// drawWarningRow renders a scheduling imbalance banner.
func (v *NodeOverviewView) drawWarningRow(s *ui.Screen, x, y, w int, msg string) {
	s.FillRect(ui.Rect{X: x, Y: y, W: w, H: 1}, ' ', ui.StyleWarning)
	s.DrawText(x+2, y, ui.StyleWarning, truncate(msg, w-4))
}

func (v *NodeOverviewView) drawPodRow(s *ui.Screen, x, y, w int, p model.Pod, selected bool) {
	nameW, nsW, statusAt := dynCols(w)
	readyAt := statusAt + 10
	restAt := readyAt + 6
	ageAt := restAt + 5

	statusStyle := podBaseStyle(p.Status)
	nameStyle := ui.StylePodName
	nsStyle := ui.StyleNamespace
	if selected {
		statusStyle = ui.StyleSelected
		nameStyle = ui.StyleSelected
		nsStyle = ui.StyleSelected
	}
	bg := ui.StyleDefault
	if selected {
		bg = ui.StyleSelected
	}
	s.FillRect(ui.Rect{X: x, Y: y, W: w, H: 1}, ' ', bg)

	iconStyle := podBaseStyle(p.Status)
	if selected {
		iconStyle = ui.StyleSelected
	}
	s.DrawText(x+1, y, iconStyle, podStatusIcon(p.Status))

	s.DrawText(x+4,        y, nameStyle,   fmt.Sprintf("%-*s", nameW, truncate(p.Name, nameW)))
	s.DrawText(x+4+nameW,  y, nsStyle,     fmt.Sprintf("%-*s", nsW, truncate(p.Namespace, nsW)))
	s.DrawText(x+statusAt, y, statusStyle, fmt.Sprintf("%-10s", podStatusShort(p.Status)))
	s.DrawText(x+readyAt,  y, statusStyle, fmt.Sprintf("%-6s", p.Ready))

	restartStyle := statusStyle
	if !selected {
		restartStyle = restartCountStyle(p.Restarts)
	}
	s.DrawText(x+restAt, y, restartStyle, fmt.Sprintf("%-5d", p.Restarts))
	s.DrawText(x+ageAt,  y, statusStyle,  formatDuration(p.Age))
}

func podBaseStyle(status string) tcell.Style {
	switch status {
	case "Running":
		return ui.StylePodRunning
	case "Pending", "Terminating":
		return ui.StylePodPending
	case "Failed", "Error", "OOMKilled", "CrashLoopBackOff",
		"ImagePullBackOff", "ErrImagePull", "CreateContainerConfigError", "InvalidImageName":
		return ui.StylePodFailed
	}
	return ui.StylePodDefault
}

func podStatusShort(status string) string {
	switch status {
	case "CrashLoopBackOff":
		return "CrashLoop"
	case "ImagePullBackOff":
		return "ImgPull"
	case "ErrImagePull":
		return "ImgPullErr"
	case "CreateContainerConfigError":
		return "CfgError"
	case "InvalidImageName":
		return "BadImage"
	}
	r := []rune(status)
	if len(r) > 10 {
		return string(r[:9]) + "…"
	}
	return status
}

func podStatusIcon(status string) string {
	switch status {
	case "Running":
		return "✔"
	case "Pending":
		return "↻"
	case "Terminating":
		return "⊘"
	case "CrashLoopBackOff":
		return "↻"
	case "Failed", "Error", "OOMKilled", "ErrImagePull",
		"CreateContainerConfigError", "InvalidImageName":
		return "✖"
	case "ImagePullBackOff":
		return "⚠"
	}
	return "·"
}

func restartCountStyle(restarts int32) tcell.Style {
	switch {
	case restarts >= 10:
		return ui.StyleRestartsCrit
	case restarts >= 3:
		return ui.StyleRestartsWarn
	}
	return ui.StylePodDefault
}

func podMatchesQuery(p model.Pod, q string) bool {
	q = strings.ToLower(q)
	return strings.Contains(strings.ToLower(p.Name), q) ||
		strings.Contains(strings.ToLower(p.Namespace), q) ||
		strings.Contains(strings.ToLower(p.Status), q)
}

func nodeMatchesQuery(n model.Node, q string) bool {
	q = strings.ToLower(q)
	return strings.Contains(strings.ToLower(n.Name), q)
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 1 {
		return "…"
	}
	return string(r[:max-1]) + "…"
}
