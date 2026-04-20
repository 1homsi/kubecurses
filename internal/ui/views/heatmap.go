package views

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/1homsi/kubecurses/internal/model"
	"github.com/1homsi/kubecurses/internal/ui"
)

// hexCellW is the column width of one pod cell: ⬢ glyph (1 col) + 1 space gap.
const hexCellW = 2

// heatmapBoxGap is the horizontal gap in columns between adjacent tiles.
const heatmapBoxGap = 2

// Reserve one full body row for node metrics / summary before the pod grid.
const heatmapHeaderBodyRows = 1

var (
	hexChar      = "⬢"
	hexCharASCII = "#"
)

// HeatmapView renders per-node hexagonal boxes of pod cells coloured by status.
type HeatmapView struct {
	cachedGen      uint64
	cachedNodePods map[string][]model.Pod
	// Cached layout plans to avoid recomputing every frame.
	cachedNodePlanKey [2]int // [nodeCount, cols]
	cachedNodePlan    []int
	cachedPodPlans    map[[3]int][]int // [nPods, maxPerRow, useHex] → plan
}

// honeycombPlan builds a simple alternating [C, C-1, C, C-1, …] row plan for
// n nodes. Even rows have cols tiles; odd rows have max(cols-1,1). The last
// row uses however many nodes remain. No symmetry requirement.
func honeycombPlan(n, cols int) []int {
	if n <= 0 {
		return nil
	}
	if cols < 1 {
		cols = 1
	}
	var plan []int
	remaining := n
	for remaining > 0 {
		c := cols
		if len(plan)%2 == 1 {
			c = cols - 1
			if c < 1 {
				c = 1
			}
		}
		if c > remaining {
			c = remaining
		}
		plan = append(plan, c)
		remaining -= c
	}
	return plan
}

func (v *HeatmapView) Render(width, height int, state *model.AppState) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	if !state.NodesLoaded || !state.PodsLoaded {
		lines := []string{ui.StyleDim.Render(ui.PadRight("  Loading...", width))}
		for len(lines) < height {
			lines = append(lines, ui.FillWidth(width, ui.StyleDefault))
		}
		return strings.Join(lines, "\n")
	}
	if len(state.Nodes) == 0 {
		lines := []string{ui.StyleDim.Render(ui.PadRight("  No nodes available.", width))}
		for len(lines) < height {
			lines = append(lines, ui.FillWidth(width, ui.StyleDefault))
		}
		return strings.Join(lines, "\n")
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
		v.cachedPodPlans = nil
		v.cachedGen = state.PodGeneration
	}
	nodePods := v.cachedNodePods

	// ── box width & column count ──────────────────────────────────────────────
	// minBoxW = 30: at this width the 3-row shoulder taper (ui.HexShoulderRows)
	// produces a clearly hexagonal silhouette.
	const minBoxW = 30
	availW := width - 2
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

	screenPodCap := (width / hexCellW) * (height - 3)
	useHexPodLayout := len(state.Pods) < screenPodCap

	// ── simple alternating row plan (cached) ─────────────────────────────────
	// [cols, cols-1, cols, cols-1, …] gives the stagger: full rows on even
	// indices, one-fewer tiles on odd indices.  The centering math in the
	// render loop naturally places odd rows at a half-pitch offset.
	planKey := [2]int{len(state.Nodes), cols}
	if v.cachedNodePlanKey != planKey || v.cachedNodePlan == nil {
		v.cachedNodePlan = honeycombPlan(len(state.Nodes), cols)
		v.cachedNodePlanKey = planKey
		v.cachedPodPlans = nil
	}
	plan := v.cachedNodePlan
	state.HeatmapRowPlan = plan
	nBoxRows := len(plan)

	// hexOverhead = top shoulders + one header body row + bottom shoulders.
	hexOverhead := 2*ui.HexShoulderRows + heatmapHeaderBodyRows
	globalBoxH := hexOverhead
	for _, node := range state.Nodes {
		nPods := len(nodePods[node.Name])
		termRows := len(v.cachedPodPlan(nPods, maxPodPerRow(boxW-2), useHexPodLayout))
		if h := termRows + hexOverhead; h > globalBoxH {
			globalBoxH = h
		}
	}

	boxRowH := make([]int, nBoxRows)
	for i := range boxRowH {
		boxRowH[i] = globalBoxH
	}

	// HexVertStep makes successive tile rows overlap by HexShoulderRows canvas
	// rows. The offset x-positions of odd rows mean the shoulder areas of
	// adjacent tile rows land at different columns, creating the interlocking
	// hex-mesh that reads as a honeycomb.
	rowStep := ui.HexVertStep(globalBoxH)
	boxRowTop := make([]int, nBoxRows+1)
	for i := 0; i < nBoxRows; i++ {
		boxRowTop[i+1] = boxRowTop[i] + rowStep
	}

	// ── scroll: keep selected node fully visible ───────────────────────────────
	availH := height - 3

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

	// ── 2D rune+style canvas ──────────────────────────────────────────────────
	canvasH := availH + globalBoxH
	if canvasH < height {
		canvasH = height
	}
	canvasRunes := make([][]rune, canvasH)
	canvasStyles := make([][]int, canvasH)
	var styles []lipgloss.Style
	styleIdx := make(map[string]int)

	getStyleIdx := func(s lipgloss.Style) int {
		key := s.Render("")
		if idx, ok := styleIdx[key]; ok {
			return idx
		}
		idx := len(styles)
		styles = append(styles, s)
		styleIdx[key] = idx
		return idx
	}
	defaultIdx := getStyleIdx(ui.StyleDefault)

	for i := range canvasRunes {
		runes := make([]rune, width)
		idxs := make([]int, width)
		for j := range runes {
			runes[j] = ' '
			idxs[j] = defaultIdx
		}
		canvasRunes[i] = runes
		canvasStyles[i] = idxs
	}

	setText := func(x, y int, style lipgloss.Style, text string) {
		idx := getStyleIdx(style)
		for _, ch := range text {
			if x >= width {
				break
			}
			if x >= 0 && y >= 0 && y < canvasH {
				canvasRunes[y][x] = ch
				canvasStyles[y][x] = idx
			}
			x++
		}
	}
	setTextTrunc := func(x, y, maxW int, style lipgloss.Style, text string) {
		runes := []rune(text)
		if len(runes) > maxW {
			runes = runes[:maxW]
		}
		setText(x, y, style, string(runes))
	}
	fillRect := func(x, y, w, h int, style lipgloss.Style) {
		idx := getStyleIdx(style)
		for ry := y; ry < y+h && ry < canvasH; ry++ {
			for rx := x; rx < x+w && rx < width; rx++ {
				if rx >= 0 && ry >= 0 {
					canvasRunes[ry][rx] = ' '
					canvasStyles[ry][rx] = idx
				}
			}
		}
	}

	// ── hex tile drawing helper ───────────────────────────────────────────────
	//
	// The tile silhouette is built entirely from background fills — no glyph
	// characters on the edges. The hex shape emerges from the 3-row shoulder
	// taper: each shoulder row narrows the tile by one column on each side
	// (ui.HexShoulderRows = 3), giving gradual angled corners against the
	// default background:
	//
	//    ██████████████     ← ry=0 shoulder, indent 3
	//   ████████████████    ← ry=1 shoulder, indent 2
	//  ██████████████████   ← ry=2 shoulder, indent 1
	//  ██████████████████   ← body (perim left/right, fill center)
	//  ██████████████████
	//   ████████████████    ← shoulder, indent 1
	//    ██████████████     ← shoulder, indent 2
	//     ████████████      ← shoulder, indent 3
	//
	drawHexTile := func(x, y, w, h int, fillStyle, perimStyle lipgloss.Style) {
		if w < ui.HexMinWidth {
			w = ui.HexMinWidth
		}
		if h < ui.HexMinHeight {
			h = ui.HexMinHeight
		}
		for ry := 0; ry < h; ry++ {
			indent := ui.HexShoulderIndent(ry, h)
			left := x + indent
			right := x + w - 1 - indent
			if left > right {
				continue
			}
			span := right - left + 1
			rowY := y + ry

			fillRect(left, rowY, span, 1, fillStyle)

			switch {
			case ry == 0:
				setText(left, rowY, perimStyle, "╱")
				if span > 2 {
					setText(left+1, rowY, perimStyle, strings.Repeat("─", span-2))
				}
				if span > 1 {
					setText(right, rowY, perimStyle, "╲")
				}
			case ry == h-1:
				setText(left, rowY, perimStyle, "╲")
				if span > 2 {
					setText(left+1, rowY, perimStyle, strings.Repeat("─", span-2))
				}
				if span > 1 {
					setText(right, rowY, perimStyle, "╱")
				}
			case ry < ui.HexShoulderRows:
				setText(left, rowY, perimStyle, "╱")
				if span > 1 {
					setText(right, rowY, perimStyle, "╲")
				}
			case (h - 1 - ry) < ui.HexShoulderRows:
				setText(left, rowY, perimStyle, "╲")
				if span > 1 {
					setText(right, rowY, perimStyle, "╱")
				}
			default:
				setText(left, rowY, perimStyle, "│")
				if span > 1 {
					setText(right, rowY, perimStyle, "│")
				}
			}
		}
	}

	// ── render boxes ─────────────────────────────────────────────────────────
	for curBoxRow := scroll; curBoxRow < nBoxRows; curBoxRow++ {
		y := boxRowTop[curBoxRow] - boxRowTop[scroll]
		if y+globalBoxH > availH {
			break
		}

		rowN := plan[curBoxRow]
		rowW := rowN*boxW + (rowN-1)*heatmapBoxGap
		// Centering shorter (odd) rows produces the natural half-pitch offset:
		// a row of (cols-1) tiles centered in availW starts at exactly
		// (boxW+gap)/2 — the honeycomb stagger — without extra math.
		nodeXBase := (availW - rowW) / 2

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

			perimStyle := ui.StyleHeatmapBorder
			if isSelected {
				perimStyle = ui.StyleHeatmapNodeSel
			}
			drawHexTile(x, y, boxW, nH, ui.StyleNodeHeader, perimStyle)

			// Title on the last top-shoulder row (ui.HexShoulderRows-1).
			// At this row HexShoulderIndent returns 1, so the hex spans
			// x+1..x+boxW-2; text safely starts at x+2.
			titleY := y + ui.HexShoulderRows - 1
			metaY := y + ui.HexShoulderRows
			dotStyle := ui.StyleNodeReadyDot
			if node.Status != "Ready" {
				dotStyle = ui.StyleNodeNotReadyDot
			}
			setText(x+2, titleY, ui.StyleNodeHeader, "●")
			setText(x+3, titleY, dotStyle, "●")

			nameMaxW := boxW - 7
			setTextTrunc(x+5, titleY, nameMaxW, ui.StyleNodeName,
				fmt.Sprintf("%-*s", nameMaxW, truncate(node.Name, nameMaxW)))

			running, pending, failing := heatmapPodCounts(pods)
			metaText := fmt.Sprintf("%d pods", len(pods))
			if node.MetricsOK {
				cpuPct, memPct := int64(0), int64(0)
				if node.AllocCPUm > 0 {
					cpuPct = node.UsedCPUm * 100 / node.AllocCPUm
				}
				if node.AllocMemMi > 0 {
					memPct = node.UsedMemMi * 100 / node.AllocMemMi
				}
				metaText = fmt.Sprintf("%d pods  C%02d M%02d", len(pods), cpuPct, memPct)
			}
			setTextTrunc(x+2, metaY, boxW-4, ui.StyleNodeMeta,
				fmt.Sprintf("%-*s", boxW-4, truncate(metaText, boxW-4)))

			statusX := x + 2
			if failing > 0 {
				statusLabel := fmt.Sprintf("%d bad", failing)
				statusX = x + boxW - 2 - len([]rune(statusLabel))
				setText(statusX, metaY, ui.StyleHeatmapFailed, statusLabel)
			} else if pending > 0 {
				statusLabel := fmt.Sprintf("%d pend", pending)
				statusX = x + boxW - 2 - len([]rune(statusLabel))
				setText(statusX, metaY, ui.StyleHeatmapPending, statusLabel)
			} else if running > 0 {
				statusLabel := fmt.Sprintf("%d run", running)
				statusX = x + boxW - 2 - len([]rune(statusLabel))
				setText(statusX, metaY, ui.StyleHeatmapRunning, statusLabel)
			}

			// Pod grid starts after the dedicated header body row.
			innerY := y + ui.HexShoulderRows + heatmapHeaderBodyRows
			podPlan := v.cachedPodPlan(len(pods), maxPodPerRow(gridW), useHexPodLayout)
			pIdx := 0
			for hexRow, rowCells := range podPlan {
				if pIdx >= len(pods) {
					break
				}
				podRowW := rowCells * hexCellW
				podXOff := x + 1 + (gridW-podRowW)/2
				rowY := innerY + hexRow
				for ci := 0; ci < rowCells && pIdx < len(pods); ci++ {
					p := pods[pIdx]
					style := heatmapPodStyle(p.Status)
					setText(podXOff+ci*hexCellW, rowY, style, cellChar)
					pIdx++
				}
			}
		}
	}

	// ── legend and hint ──────────────────────────────────────────────────────
	legendY := availH
	hintY := availH + 1
	drawLegend := func(y int) {
		if y < 0 || y >= canvasH {
			return
		}
		type entry struct {
			style lipgloss.Style
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
		if state.NoIcons {
			swatch = "#"
		}
		cx := 2
		for _, e := range entries {
			if cx >= width-2 {
				break
			}
			setText(cx, y, e.style, swatch)
			cx++
			setText(cx, y, ui.StyleDim, e.label)
			cx += len([]rune(e.label))
		}
	}
	drawLegend(legendY)
	setText(2, hintY, ui.StyleDim, "h/j/k/l: navigate  Enter: node detail  Esc: back")

	// ── render canvas to string ──────────────────────────────────────────────
	var lines []string
	for row := 0; row < canvasH && row < height; row++ {
		var b strings.Builder
		i := 0
		for i < width {
			sIdx := canvasStyles[row][i]
			j := i + 1
			for j < width && canvasStyles[row][j] == sIdx {
				j++
			}
			b.WriteString(styles[sIdx].Render(string(canvasRunes[row][i:j])))
			i = j
		}
		lines = append(lines, b.String())
	}
	for len(lines) < height {
		lines = append(lines, ui.FillWidth(width, ui.StyleDefault))
	}
	if len(lines) > height {
		lines = lines[:height]
	}

	return strings.Join(lines, "\n")
}

func maxPodPerRow(gridW int) int {
	if gridW < hexCellW {
		gridW = hexCellW
	}
	m := gridW / hexCellW
	if m < 4 {
		m = 4
	}
	return m
}

func (v *HeatmapView) cachedPodPlan(nPods, maxPerRow int, useHex bool) []int {
	hexInt := 0
	if useHex {
		hexInt = 1
	}
	key := [3]int{nPods, maxPerRow, hexInt}
	if v.cachedPodPlans != nil {
		if plan, ok := v.cachedPodPlans[key]; ok {
			return plan
		}
	}
	plan := heatmapPodPlan(nPods, maxPerRow, useHex)
	if v.cachedPodPlans == nil {
		v.cachedPodPlans = make(map[[3]int][]int)
	}
	v.cachedPodPlans[key] = plan
	return plan
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

func heatmapPodStyle(status string) lipgloss.Style {
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

func heatmapPodCounts(pods []model.Pod) (running, pending, failing int) {
	for _, pod := range pods {
		switch pod.Status {
		case "Running":
			running++
		case "Pending":
			pending++
		case "Failed", "CrashLoopBackOff", "OOMKilled", "ImagePullBackOff",
			"ErrImagePull", "CreateContainerConfigError", "InvalidImageName":
			failing++
		}
	}
	return running, pending, failing
}
