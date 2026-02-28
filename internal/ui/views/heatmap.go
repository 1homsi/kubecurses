package views

import (
	"fmt"
	"sort"

	"github.com/gdamore/tcell/v2"

	"github.com/1homsi/kubecurses/internal/model"
	"github.com/1homsi/kubecurses/internal/ui"
)

// hexCellW is the column width of one pod cell: ⬢ glyph (1 col) + 1 space gap.
const hexCellW = 2
const heatmapBoxGap = 1

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
		w := availW - (c-1)*heatmapBoxGap
		if w < c {
			continue
		}
		if w/c >= minBoxW {
			cols = c
			break
		}
	}
	boxW := (availW - (cols-1)*heatmapBoxGap) / cols
	state.HeatmapCols = cols

	// If total pods are dense enough to fill the viewport, prefer a compact
	// rectangular pod grid to maximize density and reduce visual noise.
	screenPodCap := (r.W / hexCellW) * (r.H - 3)
	useHexPodLayout := len(state.Pods) < screenPodCap

	// ── symmetric hex-cluster row plan ────────────────────────────────────────
	plan := model.HeatmapPlanRowsMin(len(state.Nodes), cols, 2)
	state.HeatmapRowPlan = plan
	nBoxRows := len(plan)

	// ── hex border taper depth ────────────────────────────────────────────────
	// Must match DrawHexBorder's own numTaper calculation so content rows line up.
	numTaper := 2
	// ── uniform box height: global max across all nodes ───────────────────────
	// All boxes the same size — height driven purely by the busiest node.
	// Overhead = 2*numTaper taper rows; title sits on the last shoulder row (free).
	hexOverhead := 2 * numTaper
	globalBoxH := hexOverhead
	for _, node := range state.Nodes {
		nPods := len(nodePods[node.Name])
		termRows := heatmapPodRows(nPods, boxW-2, useHexPodLayout)
		h := termRows + hexOverhead
		if h > globalBoxH {
			globalBoxH = h
		}
	}

	// All box-rows share the same global height.
	boxRowH := make([]int, nBoxRows)
	for i := range boxRowH {
		boxRowH[i] = globalBoxH
	}

	// Tighten vertical composition by overlapping one terminal row between box rows.
	rowStep := globalBoxH - 1
	if rowStep < 1 {
		rowStep = 1
	}
	boxRowTop := make([]int, nBoxRows+1)
	for i := 0; i < nBoxRows; i++ {
		boxRowTop[i+1] = boxRowTop[i] + rowStep
	}

	// ── scroll: keep selected node visible ────────────────────────────────────
	availH := r.H - 3

	sel := state.Selection[model.TabHeatmap]
	if sel >= len(state.Nodes) {
		sel = 0
	}
	selBoxRow, _ := model.HeatmapNodeToRowColPlan(plan, sel)
	scroll := state.HeatmapScroll
	if selBoxRow < scroll {
		scroll = selBoxRow
	}
	for scroll < selBoxRow {
		if boxRowTop[selBoxRow]-boxRowTop[scroll]+boxRowH[selBoxRow] <= availH {
			break
		}
		scroll++
	}
	state.HeatmapScroll = scroll

	// ── render ────────────────────────────────────────────────────────────────
	for curBoxRow := scroll; curBoxRow < nBoxRows; curBoxRow++ {
		y := r.Y + (boxRowTop[curBoxRow] - boxRowTop[scroll])
		if y+globalBoxH > r.Y+availH {
			break
		}

		rowN := plan[curBoxRow]
		// Center each row horizontally — symmetric hex-cluster layout.
		rowW := rowN*boxW + (rowN-1)*heatmapBoxGap
		nodeXBase := r.X + (availW-rowW)/2

		for col := 0; col < rowN; col++ {
			nodeIdx := model.HeatmapRowColToNodePlan(plan, curBoxRow, col)
			if nodeIdx >= len(state.Nodes) {
				break
			}

			x := nodeXBase + col*(boxW+heatmapBoxGap)
			node := state.Nodes[nodeIdx]
			pods := nodePods[node.Name]
			gridW := boxW - 2
			isSelected := nodeIdx == sel
			nH := globalBoxH

			// ── filled hex mask ───────────────────────────────────────────────
			// Interior: solid colorNodeBg fill defines the hex shape.
			// Perimeter: same fill normally; golden/amber ring when selected.
			perimStyle := ui.StyleNodeHeader // seamless with fill when not selected
			if isSelected {
				perimStyle = ui.StyleHeatmapNodeSel // amber background ring
			}
			ui.DrawHexFill(s, x, y, boxW, nH, ui.StyleNodeHeader, perimStyle)

			// Title on the last shoulder row (y+numTaper-1) — no wasted body row.
			titleY := y + numTaper - 1
			dotStyle := ui.StyleNodeReadyDot
			if node.Status != "Ready" {
				dotStyle = ui.StyleNodeNotReadyDot
			}
			s.DrawText(x+2, titleY, ui.StyleNodeHeader, "●")
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

			// ── pod grid: symmetric hex-cluster layout ────────────────────────
			innerY := y + numTaper
			maxPerRow := gridW / hexCellW
			if maxPerRow < 4 {
				maxPerRow = 4
			}
			podPlan := heatmapPodPlan(len(pods), maxPerRow, useHexPodLayout)
			pIdx := 0
			for hexRow, rowCells := range podPlan {
				if pIdx >= len(pods) {
					break
				}
				// Center this pod row within the inner grid width.
				rowW := rowCells * hexCellW
				podXOff := x + 1 + (gridW-rowW)/2
				rowY := innerY + hexRow
				for ci := 0; ci < rowCells && pIdx < len(pods); ci++ {
					p := pods[pIdx]
					style := heatmapPodStyle(p.Status)
					s.DrawText(podXOff+ci*hexCellW, rowY, style, cellChar)
					pIdx++
				}
			}
		}

	}

	// ── legend — always pinned to the bottom of the viewport ─────────────────
	legendY := r.Y + r.H - 2
	drawHeatmapLegend(s, r.X+2, legendY, r.W-4, state.NoIcons)
	drawHeatmapHint(s, r.X+2, r.Y+r.H-1, r.W-4)
}

// heatmapPodRows returns the number of terminal rows consumed by nPods cells
// using the same symmetric hex-cluster plan as the pod rendering.
func heatmapPodRows(nPods, gridW int, useHex bool) int {
	if gridW < hexCellW {
		gridW = hexCellW
	}
	maxPerRow := gridW / hexCellW
	if maxPerRow < 4 {
		maxPerRow = 4
	}
	return len(heatmapPodPlan(nPods, maxPerRow, useHex))
}

func heatmapPodPlan(nPods, maxPerRow int, useHex bool) []int {
	if nPods <= 0 {
		return nil
	}
	if maxPerRow < 1 {
		maxPerRow = 1
	}
	if useHex {
		return model.HeatmapPlanRowsMin(nPods, maxPerRow, 4)
	}
	rows := make([]int, 0, (nPods+maxPerRow-1)/maxPerRow)
	remaining := nPods
	for remaining > 0 {
		cells := maxPerRow
		if remaining < cells {
			cells = remaining
		}
		rows = append(rows, cells)
		remaining -= cells
	}
	return rows
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
