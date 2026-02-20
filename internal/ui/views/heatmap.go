package views

import (
	"fmt"
	"sort"

	"github.com/gdamore/tcell/v2"

	"github.com/1homsi/kubecurses/internal/model"
	"github.com/1homsi/kubecurses/internal/ui"
)

// Honeycomb layout constants for the pod grid inside each node box.
// Each pod = one ⬢ glyph + 1 space = 2 cols wide, 1 row tall.
// Odd hex-rows are indented by hexStagger to produce the brick offset.
const (
	hexCellW     = 2 // ⬢ (1 col) + 1 space gap
	hexStagger   = 1 // odd-row left-indent
	maxHexPerRow = 7 // cap per even row
)

var (
	hexChar      = "⬢"
	hexCharASCII = "#"
)

// HeatmapView renders per-node hexagonal boxes of pod cells coloured by status.
//
// Layout: nodes are arranged in a honeycomb stagger — even rows have `cols`
// nodes, odd rows have `cols-1` nodes offset right by boxW/2.  Each node box
// is a real hexagon (DrawHexBorder: indented cap + full-width shoulder + body
// + shoulder + indented cap).  Rows tile with zero gap so the caps interlock
// visually like a honeycomb.
type HeatmapView struct {
	cachedGen      uint64
	cachedNodePods map[string][]model.Pod
}

func (v *HeatmapView) Draw(s *ui.Screen, r ui.Rect, state *model.AppState) {
	s.FillRect(r, ' ', ui.StyleDefault)

	if !state.NodesLoaded || !state.PodsLoaded {
		s.DrawText(r.X+2, r.Y+1, ui.StyleDim, "Loading...")
		return
	}
	if len(state.Nodes) == 0 {
		s.DrawText(r.X+2, r.Y+1, ui.StyleDim, "No nodes available.")
		return
	}

	cellChar := hexChar
	if state.NoIcons {
		cellChar = hexCharASCII
	}

	// ── node → pods (cached) ──────────────────────────────────────────────────
	if v.cachedGen != state.PodGeneration || v.cachedNodePods == nil {
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
		v.cachedNodePods = nodePods
		v.cachedGen = state.PodGeneration
	}
	nodePods := v.cachedNodePods

	// ── box width & column count ──────────────────────────────────────────────
	minBoxW := 20
	availW := r.W - 2
	if availW < minBoxW {
		availW = minBoxW
	}
	cols := 1
	for _, c := range []int{8, 7, 6, 5, 4, 3, 2, 1} {
		if availW/c >= minBoxW {
			cols = c
			break
		}
	}
	boxW := availW / cols
	state.HeatmapCols = cols

	// ── hex border taper depth ────────────────────────────────────────────────
	// Must match DrawHexBorder's own numTaper calculation so content rows line up.
	numTaper := 2
	// ── uniform box height: global max across all nodes ───────────────────────
	// All boxes the same size — height driven purely by the busiest node.
	// Overhead = 2*numTaper (taper rows) + 1 (title body row).
	hexOverhead := 2*numTaper + 1
	globalBoxH := hexOverhead
	for _, node := range state.Nodes {
		nPods := len(nodePods[node.Name])
		termRows := heatmapPodRows(nPods, boxW-2)
		h := termRows + hexOverhead
		if h > globalBoxH {
			globalBoxH = h
		}
	}
	nodeBoxH := make([]int, len(state.Nodes))
	for i := range nodeBoxH {
		nodeBoxH[i] = globalBoxH
	}

	// All box-rows share the same global height.
	nBoxRows := heatmapTotalBoxRows(len(state.Nodes), cols)
	boxRowH := make([]int, nBoxRows)
	for i := range boxRowH {
		boxRowH[i] = globalBoxH
	}

	// Cumulative y — zero gap between rows so hex caps interlock like a real honeycomb.
	boxRowY := make([]int, nBoxRows+1)
	for i, h := range boxRowH {
		boxRowY[i+1] = boxRowY[i] + h
	}

	// ── scroll: keep selected node visible ────────────────────────────────────
	availH := r.H - 3

	sel := state.Selection[model.TabHeatmap]
	if sel >= len(state.Nodes) {
		sel = 0
	}
	selBoxRow, _ := model.HeatmapNodeToRowCol(sel, cols)
	scroll := state.HeatmapScroll
	if selBoxRow < scroll {
		scroll = selBoxRow
	}
	for scroll < selBoxRow {
		if boxRowY[selBoxRow]-boxRowY[scroll]+boxRowH[selBoxRow] <= availH {
			break
		}
		scroll++
	}
	state.HeatmapScroll = scroll

	// ── render ────────────────────────────────────────────────────────────────
	y := r.Y
	curBoxRow := scroll
	for curBoxRow < nBoxRows {
		if y >= r.Y+availH {
			break
		}

		rowN := model.HeatmapRowCols(curBoxRow, cols)
		// Odd rows are offset right by half a box width — the honeycomb stagger.
		nodeXBase := r.X
		if curBoxRow%2 == 1 {
			nodeXBase += boxW / 2
		}
		rowMaxH := boxRowH[curBoxRow]

		for col := 0; col < rowN; col++ {
			nodeIdx := model.HeatmapRowColToNode(curBoxRow, col, cols)
			if nodeIdx >= len(state.Nodes) {
				break
			}

			x := nodeXBase + col*boxW
			node := state.Nodes[nodeIdx]
			pods := nodePods[node.Name]
			gridW := boxW - 2
			isSelected := nodeIdx == sel
			nH := nodeBoxH[nodeIdx] // each node drawn at its own height

			// hexagonal border
			borderStyle := ui.StyleNodeHeader
			if isSelected {
				borderStyle = ui.StyleHeatmapNodeSel
			}
			ui.DrawHexBorder(s, x, y, boxW, nH, borderStyle)

			// Title on the first body row (y+numTaper).
			// All taper rows above/below are left empty so the hex silhouette shows.
			titleY := y + numTaper
			dotStyle := ui.StyleNodeReadyDot
			if node.Status != "Ready" {
				dotStyle = ui.StyleNodeNotReadyDot
			}
			s.DrawText(x+2, titleY, borderStyle, "●")
			s.DrawText(x+3, titleY, dotStyle, "●")

			nameMaxW := boxW - 7
			s.DrawText(x+5, titleY, ui.StyleNodeName,
				fmt.Sprintf("%-*s", nameMaxW, truncate(node.Name, nameMaxW)))
			if node.MetricsOK {
				cpuPct, memPct := int64(0), int64(0)
				if node.AllocCPUm > 0 {
					cpuPct = node.UsedCPUm * 100 / node.AllocCPUm
				}
				if node.AllocMemMi > 0 {
					memPct = node.UsedMemMi * 100 / node.AllocMemMi
				}
				s.DrawTextTrunc(x+5, titleY, boxW-7, ui.StyleNodeMeta,
					fmt.Sprintf("CPU %3d%% MEM %3d%%", cpuPct, memPct))
			}

			// ── honeycomb pod grid (starts one row after title) ────────────────
			innerY := y + numTaper + 1
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

				podXOff := x + 1
				if !evenRow {
					podXOff += hexStagger
				}
				rowY := innerY + hexRow

				for ci := 0; ci < cellsInRow && pIdx < len(pods); ci++ {
					p := pods[pIdx]
					cellX := podXOff + ci*hexCellW
					style := heatmapPodStyle(p.Status)
					s.DrawText(cellX, rowY, style, cellChar)
					pIdx++
				}
				hexRow++
			}
		}

		// 1-row gap between box rows so caps don't scatter into adjacent rows.
		y += rowMaxH + 1
		curBoxRow++
	}

	// ── legend ────────────────────────────────────────────────────────────────
	legendY := r.Y + r.H - 2
	if ly := y + 1; ly < legendY {
		legendY = ly
	}
	if legendY >= r.Y && legendY < r.Y+r.H-1 {
		drawHeatmapLegend(s, r.X+2, legendY, r.W-4, state.NoIcons)
	}
	drawHeatmapHint(s, r.X+2, r.Y+r.H-1, r.W-4)
}

// heatmapTotalBoxRows returns the number of honeycomb box-rows needed for nNodes.
func heatmapTotalBoxRows(nNodes, cols int) int {
	rows := 0
	remaining := nNodes
	for remaining > 0 {
		rowCols := model.HeatmapRowCols(rows, cols)
		if rowCols > remaining {
			rowCols = remaining
		}
		remaining -= rowCols
		rows++
	}
	return rows
}

// heatmapPodRows returns the number of terminal rows consumed by nPods cells.
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
	return hexRows
}

// heatmapPodStyle returns the tcell style for a pod hex cell.
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
