package views

import (
	"fmt"
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

	rows := make([]ovRow, 0, len(state.Nodes)+len(state.Pods))
	for _, n := range state.Nodes {
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

	// Adjust scroll to keep sel visible.
	if len(v.rows) > 0 {
		if sel < v.scrollOffset {
			v.scrollOffset = sel
		}
		if sel >= v.scrollOffset+r.H {
			v.scrollOffset = sel - r.H + 1
		}
		if v.scrollOffset < 0 {
			v.scrollOffset = 0
		}
	}

	for i := 0; i < r.H; i++ {
		rowIdx := v.scrollOffset + i
		y := r.Y + i
		if rowIdx >= len(v.rows) {
			s.FillRect(ui.Rect{X: r.X, Y: y, W: r.W, H: 1}, ' ', ui.StyleDefault)
			continue
		}
		v.drawRow(s, r.X, y, r.W, v.rows[rowIdx], rowIdx == sel)
	}
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

func (v *NodeOverviewView) drawNodeRow(s *ui.Screen, x, y, w int, n model.Node, selected bool) {
	base := ui.StyleNodeHeader
	dotStyle := ui.StyleNodeReadyDot
	metaStyle := ui.StyleNodeMeta
	if n.Status != "Ready" {
		dotStyle = ui.StyleNodeNotReadyDot
	}
	if selected {
		base = ui.StyleSelected
		dotStyle = ui.StyleSelected
		metaStyle = ui.StyleSelected
	}
	s.FillRect(ui.Rect{X: x, Y: y, W: w, H: 1}, ' ', base)
	s.DrawText(x, y, dotStyle, "●")
	s.DrawText(x+2, y, base, fmt.Sprintf("%-28s", truncate(n.Name, 28)))
	s.DrawText(x+30, y, dotStyle, fmt.Sprintf("%-12s", n.Status))
	s.DrawText(x+42, y, metaStyle, fmt.Sprintf("%-10s", formatDuration(n.Age)))
	s.DrawText(x+52, y, metaStyle, truncate(n.Version, w-52))
}

func (v *NodeOverviewView) drawPodRow(s *ui.Screen, x, y, w int, p model.Pod, selected bool) {
	rowStyle := podBaseStyle(p.Status)
	if selected {
		rowStyle = ui.StyleSelected
	}
	s.FillRect(ui.Rect{X: x, Y: y, W: w, H: 1}, ' ', ui.StyleDefault)
	if selected {
		s.FillRect(ui.Rect{X: x, Y: y, W: w, H: 1}, ' ', ui.StyleSelected)
	}

	// Columns: indent=4, name=28, namespace=16, status=10, ready=6, restarts=5, age
	s.DrawText(x+4, y, rowStyle, fmt.Sprintf("%-28s", truncate(p.Name, 28)))
	s.DrawText(x+32, y, rowStyle, fmt.Sprintf("%-16s", truncate(p.Namespace, 16)))
	s.DrawText(x+48, y, rowStyle, fmt.Sprintf("%-10s", p.Status))
	s.DrawText(x+58, y, rowStyle, fmt.Sprintf("%-6s", p.Ready))

	// Restart count — coloured by severity when not selected.
	restartStr := fmt.Sprintf("%-5d", p.Restarts)
	restartStyle := rowStyle
	if !selected {
		restartStyle = restartCountStyle(p.Restarts)
	}
	s.DrawText(x+64, y, restartStyle, restartStr)
	s.DrawText(x+69, y, rowStyle, formatDuration(p.Age))
}

// podBaseStyle returns the default (non-selected) style for a pod row.
func podBaseStyle(status string) tcell.Style {
	switch status {
	case "Running":
		return ui.StylePodRunning
	case "Pending":
		return ui.StylePodPending
	case "Failed", "Error", "OOMKilled", "CrashLoopBackOff":
		return ui.StylePodFailed
	}
	return ui.StylePodDefault
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
