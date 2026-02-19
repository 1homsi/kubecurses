package views

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"

	"github.com/1homsi/kubecurses/internal/model"
	"github.com/1homsi/kubecurses/internal/ui"
)

// ovRow is a flat display row — either a node section header or a pod entry.
type ovRow struct {
	isNode bool
	node   model.Node
	pod    model.Pod
}

// NodeOverviewView renders nodes as section headers with their pods nested below.
// This is the main/default view.
type NodeOverviewView struct {
	rows         []ovRow
	scrollOffset int
}

// buildRows groups pods under their node, respecting the active namespace filter.
// Nodes are sorted: NotReady first, then by pod count descending.
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

	// Sort: NotReady nodes first, then by pod count descending.
	nodes := make([]model.Node, len(state.Nodes))
	copy(nodes, state.Nodes)
	sort.Slice(nodes, func(i, j int) bool {
		iReady := nodes[i].Status == "Ready"
		jReady := nodes[j].Status == "Ready"
		if iReady != jReady {
			return !iReady // NotReady comes first
		}
		return len(byNode[nodes[i].Name]) > len(byNode[nodes[j].Name])
	})

	rows := make([]ovRow, 0, len(nodes)+len(state.Pods))
	for _, n := range nodes {
		pods := byNode[n.Name]
		// When searching, hide nodes with no matching pods.
		if query != "" && len(pods) == 0 && !nodeMatchesQuery(n, query) {
			continue
		}
		rows = append(rows, ovRow{isNode: true, node: n})
		for _, p := range pods {
			rows = append(rows, ovRow{pod: p})
		}
	}

	if len(unscheduled) > 0 {
		rows = append(rows, ovRow{isNode: true, node: model.Node{Name: "<unscheduled>", Status: "Unknown"}})
		for _, p := range unscheduled {
			rows = append(rows, ovRow{pod: p})
		}
	}
	return rows
}

func (v *NodeOverviewView) Draw(s *ui.Screen, r ui.Rect, state *model.AppState) {
	v.rows = v.buildRows(state, state.SearchQuery)
	sel := state.Selection[model.TabNodeOverview]

	// Clamp selection to actual row count.
	if sel >= len(v.rows) && len(v.rows) > 0 {
		sel = len(v.rows) - 1
		state.Selection[model.TabNodeOverview] = sel
	}

	// ── sticky column header ──────────────────────────────────────────────
	v.drawHeader(s, r.X, r.Y, r.W)

	// Content area is one row shorter (header takes r.Y).
	content := ui.Rect{X: r.X, Y: r.Y + 1, W: r.W, H: r.H - 1}

	// Adjust scroll to keep sel visible.
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

// drawHeader renders the sticky column-label row.
// Columns: indent(4) name(28) version/ns(16) status(10) ready(6) restarts(5) age
func (v *NodeOverviewView) drawHeader(s *ui.Screen, x, y, w int) {
	s.FillRect(ui.Rect{X: x, Y: y, W: w, H: 1}, ' ', ui.StyleHeader)
	s.DrawText(x+4,  y, ui.StyleHeader, fmt.Sprintf("%-28s", "NAME"))
	s.DrawText(x+32, y, ui.StyleHeader, fmt.Sprintf("%-16s", "NAMESPACE"))
	s.DrawText(x+48, y, ui.StyleHeader, fmt.Sprintf("%-10s", "STATUS"))
	s.DrawText(x+58, y, ui.StyleHeader, fmt.Sprintf("%-6s",  "READY"))
	s.DrawText(x+64, y, ui.StyleHeader, fmt.Sprintf("%-5s",  "REST"))
	s.DrawText(x+69, y, ui.StyleHeader, "AGE")
}

// RowCount returns the current number of display rows (used by app for MoveSelection).
func (v *NodeOverviewView) RowCount() int { return len(v.rows) }

func (v *NodeOverviewView) drawRow(s *ui.Screen, x, y, w int, row ovRow, selected bool) {
	if row.isNode {
		v.drawNodeRow(s, x, y, w, row.node, selected)
	} else {
		v.drawPodRow(s, x, y, w, row.pod, selected)
	}
}

// drawNodeRow renders a node section header aligned to the same column grid as pod rows.
// Columns used: NAME(x+4,28) | VERSION(x+32,16) | STATUS(x+48,10) | AGE(x+69)
func (v *NodeOverviewView) drawNodeRow(s *ui.Screen, x, y, w int, n model.Node, selected bool) {
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
	s.DrawText(x, y, dotStyle, "●")
	s.DrawText(x+4,  y, nameStyle,   fmt.Sprintf("%-28s", truncate(n.Name, 28)))
	s.DrawText(x+32, y, metaStyle,   fmt.Sprintf("%-16s", truncate(n.Version, 16)))
	s.DrawText(x+48, y, statusStyle, fmt.Sprintf("%-10s", n.Status))
	s.DrawText(x+69, y, metaStyle,   formatDuration(n.Age))
}

func (v *NodeOverviewView) drawPodRow(s *ui.Screen, x, y, w int, p model.Pod, selected bool) {
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

	// Status icon at x+1 (within the 4-char indent).
	iconStyle := podBaseStyle(p.Status)
	if selected {
		iconStyle = ui.StyleSelected
	}
	s.DrawText(x+1, y, iconStyle, podStatusIcon(p.Status))

	// Columns: indent=4, name=28, namespace=16, status=10, ready=6, restarts=5, age
	s.DrawText(x+4,  y, nameStyle,   fmt.Sprintf("%-28s", truncate(p.Name, 28)))
	s.DrawText(x+32, y, nsStyle,     fmt.Sprintf("%-16s", truncate(p.Namespace, 16)))
	s.DrawText(x+48, y, statusStyle, fmt.Sprintf("%-10s", podStatusShort(p.Status)))
	s.DrawText(x+58, y, statusStyle, fmt.Sprintf("%-6s", p.Ready))

	// Restart count — coloured by severity when not selected.
	restartStr := fmt.Sprintf("%-5d", p.Restarts)
	restartStyle := statusStyle
	if !selected {
		restartStyle = restartCountStyle(p.Restarts)
	}
	s.DrawText(x+64, y, restartStyle, restartStr)
	s.DrawText(x+69, y, statusStyle, formatDuration(p.Age))
}

// podBaseStyle returns the default (non-selected) style for a pod row based on status.
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

// podStatusShort returns an abbreviated status string that fits within 10 chars.
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

// podStatusIcon returns a 1-char icon representing the pod's status.
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

// restartCountStyle returns a warning/critical style based on restart count.
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
