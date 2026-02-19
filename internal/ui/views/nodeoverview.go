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
type NodeOverviewView struct {
	rows         []ovRow
	scrollOffset int
}

// dynCols computes dynamic column widths from the available row width w.
// The right block (STATUS 10 + READY 6 + REST 5 + AGE 11 = 32) is right-anchored.
// The remaining space is split 2:1 between the NAME and NAMESPACE/VERSION columns.
func dynCols(w int) (nameW, nsW, statusAt int) {
	statusAt = w - 32
	if statusAt < 40 {
		statusAt = 40
	}
	avail := statusAt - 4 // 4-char left indent (icon + spaces)
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

	rows := make([]ovRow, 0, len(nodes)+len(state.Pods))
	for _, n := range nodes {
		pods := byNode[n.Name]
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
	s.DrawText(x+4,          y, ui.StyleHeader, fmt.Sprintf("%-*s", nameW, "NAME"))
	s.DrawText(x+4+nameW,    y, ui.StyleHeader, fmt.Sprintf("%-*s", nsW, "NAMESPACE"))
	s.DrawText(x+statusAt,   y, ui.StyleHeader, fmt.Sprintf("%-10s", "STATUS"))
	s.DrawText(x+readyAt,    y, ui.StyleHeader, fmt.Sprintf("%-6s", "READY"))
	s.DrawText(x+restAt,     y, ui.StyleHeader, fmt.Sprintf("%-5s", "REST"))
	s.DrawText(x+ageAt,      y, ui.StyleHeader, "AGE")
}

// RowCount returns the current number of display rows.
func (v *NodeOverviewView) RowCount() int { return len(v.rows) }

func (v *NodeOverviewView) drawRow(s *ui.Screen, x, y, w int, row ovRow, selected bool) {
	if row.isNode {
		v.drawNodeRow(s, x, y, w, row.node, selected)
	} else {
		v.drawPodRow(s, x, y, w, row.pod, selected)
	}
}

func (v *NodeOverviewView) drawNodeRow(s *ui.Screen, x, y, w int, n model.Node, selected bool) {
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
	s.DrawText(x,           y, dotStyle,    "●")
	s.DrawText(x+4,         y, nameStyle,   fmt.Sprintf("%-*s", nameW, truncate(n.Name, nameW)))
	s.DrawText(x+4+nameW,   y, metaStyle,   fmt.Sprintf("%-*s", nsW, truncate(n.Version, nsW)))
	s.DrawText(x+statusAt,  y, statusStyle, fmt.Sprintf("%-10s", n.Status))
	s.DrawText(x+ageAt,     y, metaStyle,   formatDuration(n.Age))
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

	// Status icon within the 4-char left indent.
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
