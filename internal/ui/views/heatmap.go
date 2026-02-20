package views

import (
	"fmt"
	"sort"

	"github.com/gdamore/tcell/v2"

	"github.com/1homsi/kubecurses/internal/model"
	"github.com/1homsi/kubecurses/internal/ui"
)

// Honeycomb layout: each pod = one ⬢ glyph + 1 space gap = 2 cols wide, 1 row tall.
// Odd hex-rows are indented by hexStagger (1 col) to produce the classic brick offset.
//
// Visual (4 even + 3 odd, all green):
//
//   ⬢ ⬢ ⬢ ⬢     ← even row
//    ⬢ ⬢ ⬢      ← odd row (stagger 1)
//   ⬢ ⬢ ⬢ ⬢     ← even row

const (
	hexCellW      = 2  // ⬢ (1 col) + 1 space gap
	hexStagger    = 1  // odd-row left-indent
	maxHexPerRow  = 10 // cap per even row so wrapping always produces honeycomb rows
)

var (
	hexChar      = "⬢"
	hexCharASCII = "#"
)

// HeatmapView renders per-node honeycomb grids of pod cells coloured by status.
// Selection tracks the active NODE. h/j/k/l navigate the grid; Enter opens the node.
type HeatmapView struct{}

func (v *HeatmapView) Draw(s *ui.Screen, r ui.Rect, state *model.AppState) {
	// Clear background so stale content from previous renders doesn't bleed through.
	s.FillRect(r, ' ', ui.StyleDefault)

	if len(state.Nodes) == 0 {
		msg := "Loading..."
		if state.NodesLoaded {
			msg = "No nodes available."
		}
		s.DrawText(r.X+2, r.Y+1, ui.StyleDim, msg)
		return
	}

	cellChar := hexChar
	if state.NoIcons {
		cellChar = hexCharASCII
	}

	// ── node → pods ───────────────────────────────────────────────────────────
	nodePods := make(map[string][]model.Pod, len(state.Nodes))
	for _, p := range state.Pods {
		nodePods[p.Node] = append(nodePods[p.Node], p)
	}
	podOrder := func(status string) int {
		switch status {
		case "Failed", "CrashLoopBackOff", "OOMKilled", "ImagePullBackOff",
			"ErrImagePull", "CreateContainerConfigError", "InvalidImageName":
			return 0
		case "Pending":
			return 1
		case "Running":
			return 2
		case "Terminating":
			return 3
		}
		return 4
	}
	for node := range nodePods {
		pods := nodePods[node]
		sort.Slice(pods, func(i, j int) bool {
			oi, oj := podOrder(pods[i].Status), podOrder(pods[j].Status)
			if oi != oj {
				return oi < oj
			}
			return pods[i].Name < pods[j].Name
		})
		nodePods[node] = pods
	}

	// ── box width & column count ──────────────────────────────────────────────
	minBoxW := 30
	availW := r.W - 2
	if availW < minBoxW {
		availW = minBoxW
	}
	// Prefer 3 columns, fall back to 2 then 1.
	cols := 1
	for _, c := range []int{3, 2, 1} {
		if availW/c >= minBoxW {
			cols = c
			break
		}
	}
	boxW := availW / cols
	state.HeatmapCols = cols

	// ── pre-pass: uniform box height (global max across all nodes) ────────────
	globalBoxH := 4
	for _, node := range state.Nodes {
		nPods := len(nodePods[node.Name])
		termRows := heatmapPodRows(nPods, boxW-2)
		h := 2 + termRows
		if h < 4 {
			h = 4
		}
		if h > globalBoxH {
			globalBoxH = h
		}
	}
	nodeBoxH := make([]int, len(state.Nodes))
	for i := range nodeBoxH {
		nodeBoxH[i] = globalBoxH
	}

	nBoxRows := (len(state.Nodes) + cols - 1) / cols
	boxRowH := make([]int, nBoxRows)
	for i := range boxRowH {
		boxRowH[i] = globalBoxH
	}

	// cumulative y for scroll math
	boxRowY := make([]int, nBoxRows+1)
	for i, h := range boxRowH {
		boxRowY[i+1] = boxRowY[i] + h + 2
	}

	// ── scroll: keep selected node visible ────────────────────────────────────
	sel := state.Selection[model.TabHeatmap]
	if sel >= len(state.Nodes) {
		sel = 0
	}
	selBoxRow := sel / cols
	scroll := state.HeatmapScroll
	if selBoxRow < scroll {
		scroll = selBoxRow
	}
	for scroll < selBoxRow {
		if boxRowY[selBoxRow]-boxRowY[scroll]+boxRowH[selBoxRow]+2 <= r.H {
			break
		}
		scroll++
	}
	state.HeatmapScroll = scroll

	// ── render ────────────────────────────────────────────────────────────────
	firstNode := scroll * cols
	y := r.Y
	curBoxRow := scroll

	for nodeIdx := firstNode; nodeIdx < len(state.Nodes); nodeIdx++ {
		col := nodeIdx % cols
		if nodeIdx > firstNode && col == 0 {
			y += boxRowH[curBoxRow] + 2
			curBoxRow++
		}
		if y >= r.Y+r.H {
			break
		}

		x := r.X + col*boxW
		node := state.Nodes[nodeIdx]
		pods := nodePods[node.Name]
		gridW := boxW - 2
		boxH := nodeBoxH[nodeIdx]
		isSelected := nodeIdx == sel

		// border
		borderStyle := ui.StyleNodeHeader
		if isSelected {
			borderStyle = ui.StyleHeatmapNodeSel
		}
		ui.DrawBorderOnly(s, x, y, boxW, boxH+1, borderStyle)

		// ready dot on top border
		dotStyle := ui.StyleNodeReadyDot
		if node.Status != "Ready" {
			dotStyle = ui.StyleNodeNotReadyDot
		}
		s.DrawText(x+2, y, borderStyle, "●")
		s.DrawText(x+3, y, dotStyle, "●")

		// header row: name (overwritten by metrics when available)
		nameMaxW := boxW - 6
		s.DrawText(x+2, y+1, ui.StyleNodeName,
			fmt.Sprintf("%-*s", nameMaxW, truncate(node.Name, nameMaxW)))
		if node.MetricsOK {
			cpuPct, memPct := int64(0), int64(0)
			if node.AllocCPUm > 0 {
				cpuPct = node.UsedCPUm * 100 / node.AllocCPUm
			}
			if node.AllocMemMi > 0 {
				memPct = node.UsedMemMi * 100 / node.AllocMemMi
			}
			s.DrawTextTrunc(x+2, y+1, boxW-4, ui.StyleNodeMeta,
				fmt.Sprintf("CPU %3d%% MEM %3d%%", cpuPct, memPct))
		}

		// ── honeycomb pod grid ─────────────────────────────────────────────────
		// Each logical hex-row = 1 terminal row.
		// Even rows flush at x+1; odd rows indented by hexStagger (1 col).
		// Stride = hexCellW (2): ⬢ glyph + 1 space.
		innerY := y + 2
		pIdx := 0
		hexRow := 0
		for pIdx < len(pods) {
			evenRow := hexRow%2 == 0
			cellsInRow := gridW / hexCellW
			if cellsInRow > maxHexPerRow {
				cellsInRow = maxHexPerRow
			}
			if !evenRow {
				odd := (gridW - hexStagger) / hexCellW
				if odd > maxHexPerRow-1 {
					odd = maxHexPerRow - 1
				}
				cellsInRow = odd
			}
			if cellsInRow < 1 {
				cellsInRow = 1
			}

			xOff := x + 1
			if !evenRow {
				xOff += hexStagger
			}
			rowY := innerY + hexRow

			for ci := 0; ci < cellsInRow && pIdx < len(pods); ci++ {
				p := pods[pIdx]
				cellX := xOff + ci*hexCellW
				style := heatmapPodStyle(p.Status)
				s.DrawText(cellX, rowY, style, cellChar)
				pIdx++
			}
			hexRow++
		}
	}

	// ── legend ────────────────────────────────────────────────────────────────
	legendY := r.Y + r.H - 2
	if curBoxRow < len(boxRowH) {
		if ly := y + boxRowH[curBoxRow] + 1; ly < legendY {
			legendY = ly
		}
	}
	if legendY >= r.Y && legendY < r.Y+r.H-1 {
		drawHeatmapLegend(s, r.X+2, legendY, r.W-4, state.NoIcons)
	}
	drawHeatmapHint(s, r.X+2, r.Y+r.H-1, r.W-4)
}

// heatmapPodRows returns the number of terminal rows consumed by nPods cells.
// Each logical hex-row = 1 terminal row.
func heatmapPodRows(nPods, gridW int) int {
	if gridW < hexCellW {
		gridW = hexCellW
	}
	hexRows, remaining, row := 0, nPods, 0
	for remaining > 0 {
		var c int
		if row%2 == 0 {
			c = gridW / hexCellW
			if c > maxHexPerRow {
				c = maxHexPerRow
			}
		} else {
			c = (gridW - hexStagger) / hexCellW
			if c > maxHexPerRow-1 {
				c = maxHexPerRow - 1
			}
		}
		if c < 1 {
			c = 1
		}
		remaining -= c
		hexRows++
		row++
	}
	return hexRows // each logical hex-row = 1 terminal row
}

// heatmapPodStyle returns the style for a hex pod cell.
func heatmapPodStyle(status string) tcell.Style {
	switch status {
	case "Running":
		return ui.StyleHeatmapRunning
	case "Pending":
		return ui.StyleHeatmapPending
	case "Terminating":
		return ui.StyleHeatmapTerm
	case "Failed", "CrashLoopBackOff", "OOMKilled", "ImagePullBackOff",
		"ErrImagePull", "CreateContainerConfigError", "InvalidImageName":
		return ui.StyleHeatmapFailed
	}
	return ui.StyleHeatmapDefault
}

// drawHeatmapLegend renders a colour-coded legend row.
func drawHeatmapLegend(s *ui.Screen, x, y, maxW int, noIcons bool) {
	type entry struct {
		style tcell.Style
		label string
	}
	entries := []entry{
		{ui.StyleHeatmapRunning, " Running  "},
		{ui.StyleHeatmapPending, " Pending  "},
		{ui.StyleHeatmapFailed, " Failed  "},
		{ui.StyleHeatmapTerm, " Terminating  "},
		{ui.StyleHeatmapDefault, " Unknown  "},
	}
	swatch := "⬢"
	if noIcons {
		swatch = "#"
	}
	cx := x
	for _, e := range entries {
		if cx >= x+maxW {
			break
		}
		s.DrawText(cx, y, e.style, swatch)
		cx++
		rem := x + maxW - cx
		if rem <= 0 {
			break
		}
		runes := []rune(e.label)
		if len(runes) > rem {
			runes = runes[:rem]
		}
		s.DrawTextTrunc(cx, y, len(runes), ui.StyleDim, string(runes))
		cx += len(runes)
	}
}

func drawHeatmapHint(s *ui.Screen, x, y, maxW int) {
	s.DrawTextTrunc(x, y, maxW, ui.StyleDim,
		"h/j/k/l: navigate  Enter: node detail  Esc: back")
}
