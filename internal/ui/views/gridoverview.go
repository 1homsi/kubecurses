package views

import (
	"fmt"
	"strings"

	"github.com/1homsi/kubecurses/internal/model"
	"github.com/1homsi/kubecurses/internal/ui"
)

// GridOverviewView renders a 2×2 grid of node cards.
// Each card shows the node header and its pods. j/k moves the selected card;
// PgDn/PgUp pages through groups of 4 nodes.
type GridOverviewView struct {
	nodeCount int // cached from last Draw, used by RowCount
}

// RowCount returns the number of navigable node slots (one per node).
func (v *GridOverviewView) RowCount() int { return v.nodeCount }

func (v *GridOverviewView) Draw(s *ui.Screen, r ui.Rect, state *model.AppState) {
	nodes := state.Nodes
	v.nodeCount = len(nodes)
	byNode := v.groupPods(state)

	sel := state.Selection[model.TabNodeOverview]
	if sel < 0 {
		sel = 0
	}
	if len(nodes) > 0 && sel >= len(nodes) {
		sel = len(nodes) - 1
		state.Selection[model.TabNodeOverview] = sel
	}

	// Which page of 4 contains the selected node?
	pageOffset := (sel / 4) * 4

	// ── separators ────────────────────────────────────────────────────────
	sepX := r.X + r.W/2
	sepY := r.Y + r.H/2

	for y := r.Y; y < r.Y+r.H; y++ {
		s.DrawText(sepX, y, ui.StyleSeparator, "│")
	}
	for x := r.X; x < r.X+r.W; x++ {
		if x == sepX {
			continue
		}
		s.DrawText(x, sepY, ui.StyleSeparator, "─")
	}
	s.DrawText(sepX, sepY, ui.StyleSeparator, "┼")

	// ── four cells ────────────────────────────────────────────────────────
	leftW := r.W / 2
	rightW := r.W - leftW - 1 // -1 for separator column
	topH := r.H / 2
	botH := r.H - topH - 1 // -1 for separator row

	cells := [4]ui.Rect{
		{X: r.X, Y: r.Y, W: leftW, H: topH},                         // 0 top-left
		{X: sepX + 1, Y: r.Y, W: rightW, H: topH},                   // 1 top-right
		{X: r.X, Y: sepY + 1, W: leftW, H: botH},                    // 2 bottom-left
		{X: sepX + 1, Y: sepY + 1, W: rightW, H: botH},              // 3 bottom-right
	}

	for i, cell := range cells {
		nodeIdx := pageOffset + i
		selected := nodeIdx == sel
		if nodeIdx < len(nodes) {
			v.drawCard(s, cell, nodes[nodeIdx], byNode[nodes[nodeIdx].Name], selected)
		} else {
			v.drawEmptyCard(s, cell)
		}
	}

	// Page indicator when there are more than 4 nodes.
	if len(nodes) > 4 {
		totalPages := (len(nodes) + 3) / 4
		currentPage := pageOffset/4 + 1
		indicator := fmt.Sprintf(" %d/%d pg ", currentPage, totalPages)
		s.DrawText(r.X+r.W-len([]rune(indicator)), r.Y+r.H-1, ui.StyleDim, indicator)
	}
}

func (v *GridOverviewView) drawCard(s *ui.Screen, cell ui.Rect, n model.Node, pods []model.Pod, selected bool) {
	if cell.W <= 0 || cell.H <= 0 {
		return
	}
	s.FillRect(cell, ' ', ui.StyleDefault)

	// ── node header row ───────────────────────────────────────────────────
	headerStyle := ui.StyleNodeHeader
	dotStyle := ui.StyleNodeReadyDot
	if n.Status != "Ready" {
		dotStyle = ui.StyleNodeNotReadyDot
	}
	if selected {
		headerStyle = ui.StyleSelected
		dotStyle = ui.StyleSelected
	}
	s.FillRect(ui.Rect{X: cell.X, Y: cell.Y, W: cell.W, H: 1}, ' ', headerStyle)
	s.DrawText(cell.X, cell.Y, dotStyle, "●")

	// Node name truncated to leave room for the status badge.
	badge := fmt.Sprintf(" %s ", n.Status)
	badgeLen := len([]rune(badge))
	nameMax := cell.W - 3 - badgeLen
	if nameMax < 1 {
		nameMax = 1
	}
	// Use bright sky-blue for node name, fall back to selection style when selected.
	nameStyle := ui.StyleNodeName
	if selected {
		nameStyle = ui.StyleSelected
	}
	s.DrawText(cell.X+2, cell.Y, nameStyle, truncate(n.Name, nameMax))

	// Status badge right-aligned in the header.
	badgeX := cell.X + cell.W - badgeLen
	if badgeX > cell.X+2 {
		s.DrawText(badgeX, cell.Y, dotStyle, badge)
	}

	// ── pod rows ──────────────────────────────────────────────────────────
	maxPodRows := cell.H - 1
	for i, p := range pods {
		if i >= maxPodRows {
			break
		}
		rowY := cell.Y + 1 + i
		// Last visible row but more pods exist → show overflow count.
		if i == maxPodRows-1 && len(pods) > maxPodRows {
			remaining := len(pods) - maxPodRows
			s.DrawText(cell.X+2, rowY, ui.StyleDim, fmt.Sprintf("… %d more", remaining))
			break
		}
		v.drawPodLine(s, cell.X+2, rowY, cell.W-2, p)
	}
}

func (v *GridOverviewView) drawPodLine(s *ui.Screen, x, y, w int, p model.Pod) {
	if w <= 0 {
		return
	}
	statusStyle := podBaseStyle(p.Status)

	// Name(lavender) | status(colored) | restarts(warn color) — 50/40/10 split.
	nameW := w * 5 / 10
	statusW := w * 4 / 10
	restW := w - nameW - statusW

	s.DrawTextTrunc(x, y, nameW, ui.StylePodName, p.Name)
	s.DrawTextTrunc(x+nameW, y, statusW, statusStyle, p.Status)
	if restW > 1 && p.Restarts > 0 {
		s.DrawTextTrunc(x+nameW+statusW, y, restW, restartCountStyle(p.Restarts), fmt.Sprintf("%dr", p.Restarts))
	}
}

func (v *GridOverviewView) drawEmptyCard(s *ui.Screen, cell ui.Rect) {
	s.FillRect(cell, ' ', ui.StyleDefault)
}

func (v *GridOverviewView) groupPods(state *model.AppState) map[string][]model.Pod {
	q := strings.ToLower(state.SearchQuery[model.TabNodeOverview])
	byNode := make(map[string][]model.Pod, len(state.Nodes))
	for _, p := range state.Pods {
		if state.Namespace != "" && p.Namespace != state.Namespace {
			continue
		}
		if q != "" && !podMatchesQuery(p, q) {
			continue
		}
		byNode[p.Node] = append(byNode[p.Node], p)
	}
	return byNode
}

// Compile-time assertion that GridOverviewView satisfies the View interface.
var _ View = (*GridOverviewView)(nil)
