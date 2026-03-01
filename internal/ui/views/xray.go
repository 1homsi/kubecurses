package views

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"

	"github.com/1homsi/kubecurses/internal/model"
	"github.com/1homsi/kubecurses/internal/ui"
)

type xrayRowKind int

const (
	xrkNs        xrayRowKind = iota
	xrkPod
	xrkContainer
	xrkDetail
)

type xrayRow struct {
	kind       xrayRowKind
	ns         string
	nsPodCnt   int
	pod        model.Pod
	container  model.Container
	podIsLast  bool
	contIsLast bool
	detailText string
}

// xrayCacheKey groups all inputs that affect the row model.
type xrayCacheKey struct {
	podGen   uint64
	query    string
	ns       string
	nsFilter string
}

// XrayView renders a namespace → pod → container tree.
type XrayView struct {
	rows         []xrayRow
	scrollOffset int
	cacheValid   bool
	cacheKey     xrayCacheKey
	cachedRows   []xrayRow
}

// RowCount returns the current number of display rows (all kinds).
func (v *XrayView) RowCount() int { return len(v.rows) }

// SelectedRef returns the namespace, pod name, and container name for the row
// at idx. container is empty when the selection is on a pod or namespace row.
// Returns ("", "", "") when idx is out of range or on a namespace header.
func (v *XrayView) SelectedRef(idx int) (ns, name, container string) {
	if idx < 0 || idx >= len(v.rows) {
		return "", "", ""
	}
	row := v.rows[idx]
	switch row.kind {
	case xrkPod:
		return row.pod.Namespace, row.pod.Name, ""
	case xrkContainer, xrkDetail:
		return row.pod.Namespace, row.pod.Name, row.container.Name
	}
	return "", "", ""
}

func (v *XrayView) buildRows(state *model.AppState, query string) []xrayRow {
	nsMap := make(map[string][]model.Pod)
	nsSeen := make(map[string]bool)
	var nsOrder []string

	for _, p := range state.Pods {
		if state.Namespace != "" && p.Namespace != state.Namespace {
			continue
		}
		if state.NamespaceFilter != "" && p.Namespace != state.NamespaceFilter {
			continue
		}
		if query != "" && !podMatchesQuery(p, query) {
			continue
		}
		if !nsSeen[p.Namespace] {
			nsSeen[p.Namespace] = true
			nsOrder = append(nsOrder, p.Namespace)
		}
		nsMap[p.Namespace] = append(nsMap[p.Namespace], p)
	}
	sort.Strings(nsOrder)

	var rows []xrayRow
	for _, ns := range nsOrder {
		pods := nsMap[ns]
		rows = append(rows, xrayRow{kind: xrkNs, ns: ns, nsPodCnt: len(pods)})
		for pi, pod := range pods {
			podIsLast := pi == len(pods)-1
			rows = append(rows, xrayRow{
				kind:      xrkPod,
				pod:       pod,
				podIsLast: podIsLast,
			})
			for ci, cont := range pod.Containers {
				isLast := ci == len(pod.Containers)-1
				hasDetail := cont.Message != ""
				rows = append(rows, xrayRow{
					kind:       xrkContainer,
					pod:        pod,
					container:  cont,
					podIsLast:  podIsLast,
					contIsLast: isLast && !hasDetail,
				})
				if hasDetail {
					rows = append(rows, xrayRow{
						kind:       xrkDetail,
						pod:        pod,
						container:  cont,
						podIsLast:  podIsLast,
						contIsLast: isLast,
						detailText: cont.Message,
					})
				}
			}
		}
	}
	return rows
}

func (v *XrayView) Draw(s *ui.Screen, r ui.Rect, state *model.AppState) {
	key := xrayCacheKey{state.PodGeneration, state.SearchQuery[model.TabPods], state.Namespace, state.NamespaceFilter}
	if !v.cacheValid || key != v.cacheKey {
		v.cachedRows = v.buildRows(state, state.SearchQuery[model.TabPods])
		v.cacheKey = key
		v.cacheValid = true
	}
	v.rows = v.cachedRows
	sel := state.Selection[model.TabPods]

	if sel >= len(v.rows) && len(v.rows) > 0 {
		sel = len(v.rows) - 1
		state.Selection[model.TabPods] = sel
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

func (v *XrayView) drawHeader(s *ui.Screen, x, y, w int) {
	_, _, statusAt := dynCols(w)
	readyAt := statusAt + 10
	restAt := readyAt + 6
	ageAt := restAt + 5

	s.FillRect(ui.Rect{X: x, Y: y, W: w, H: 1}, ' ', ui.StyleHeader)
	nameW := statusAt - 5
	if nameW < 4 {
		nameW = 4
	}
	s.DrawText(x+5, y, ui.StyleHeader, fmt.Sprintf("%-*s", nameW, "NAME"))
	s.DrawText(x+statusAt, y, ui.StyleHeader, fmt.Sprintf("%-10s", "STATUS"))
	s.DrawText(x+readyAt, y, ui.StyleHeader, fmt.Sprintf("%-6s", "READY"))
	s.DrawText(x+restAt, y, ui.StyleHeader, fmt.Sprintf("%-5s", "REST"))
	s.DrawText(x+ageAt, y, ui.StyleHeader, "AGE")
}

func (v *XrayView) drawRow(s *ui.Screen, x, y, w int, row xrayRow, selected bool) {
	switch row.kind {
	case xrkNs:
		v.drawNsRow(s, x, y, w, row, selected)
	case xrkPod:
		v.drawPodRow(s, x, y, w, row, selected)
	case xrkContainer:
		v.drawContainerRow(s, x, y, w, row, selected)
	case xrkDetail:
		v.drawDetailRow(s, x, y, w, row, selected)
	}
}

func (v *XrayView) drawNsRow(s *ui.Screen, x, y, w int, row xrayRow, selected bool) {
	bg := ui.StyleXrayNsHeader
	if selected {
		bg = ui.StyleSelected
	}
	s.FillRect(ui.Rect{X: x, Y: y, W: w, H: 1}, ' ', bg)
	label := fmt.Sprintf("▸ %s  (%d pods)", row.ns, row.nsPodCnt)
	s.DrawText(x, y, bg, truncate(label, w-2))
}

func (v *XrayView) drawPodRow(s *ui.Screen, x, y, w int, row xrayRow, selected bool) {
	_, _, statusAt := dynCols(w)
	readyAt := statusAt + 10
	restAt := readyAt + 6
	ageAt := restAt + 5

	p := row.pod
	connector := "  ├─ "
	if row.podIsLast {
		connector = "  └─ "
	}
	nameW := statusAt - 5
	if nameW < 10 {
		nameW = 10
	}

	statusStyle := podBaseStyle(p.Status)
	nameStyle := ui.StylePodName
	connStyle := ui.StyleXrayConnector
	if selected {
		statusStyle = ui.StyleSelected
		nameStyle = ui.StyleSelected
		connStyle = ui.StyleSelected
	}
	bg := ui.StyleDefault
	if selected {
		bg = ui.StyleSelected
	}
	s.FillRect(ui.Rect{X: x, Y: y, W: w, H: 1}, ' ', bg)
	s.DrawText(x, y, connStyle, connector)
	s.DrawText(x+5, y, nameStyle, fmt.Sprintf("%-*s", nameW, truncate(p.Name, nameW)))
	s.DrawText(x+statusAt, y, statusStyle, fmt.Sprintf("%-10s", podStatusShort(p.Status)))
	s.DrawText(x+readyAt, y, statusStyle, fmt.Sprintf("%-6s", p.Ready))

	restartStyle := statusStyle
	if !selected {
		restartStyle = restartCountStyle(p.Restarts)
	}
	s.DrawText(x+restAt, y, restartStyle, fmt.Sprintf("%-5d", p.Restarts))
	s.DrawText(x+ageAt, y, statusStyle, formatDuration(p.Age))
}

func (v *XrayView) drawContainerRow(s *ui.Screen, x, y, w int, row xrayRow, selected bool) {
	c := row.container

	var connector string
	switch {
	case row.podIsLast && row.contIsLast:
		connector = "     └─ "
	case row.podIsLast && !row.contIsLast:
		connector = "     ├─ "
	case !row.podIsLast && row.contIsLast:
		connector = "  │  └─ "
	default:
		connector = "  │  ├─ "
	}

	iconStyle := containerStatusStyle(c.Status)
	nameStyle := ui.StyleDim
	connStyle := ui.StyleXrayConnector
	if selected {
		iconStyle = ui.StyleSelected
		nameStyle = ui.StyleSelected
		connStyle = ui.StyleSelected
	}
	bg := ui.StyleDefault
	if selected {
		bg = ui.StyleSelected
	}
	s.FillRect(ui.Rect{X: x, Y: y, W: w, H: 1}, ' ', bg)
	s.DrawText(x, y, connStyle, connector)

	const rightBlock = 17
	nameStart := x + 8

	resText := ""
	if w >= 100 && (c.CPURequestM > 0 || c.MemRequestMi > 0) {
		resText = fmt.Sprintf("cpu:%s/%s mem:%s/%s",
			formatCPU(c.CPURequestM), formatCPU(c.CPULimitM),
			formatMem(c.MemRequestMi), formatMem(c.MemLimitMi))
	}

	nameMaxW := w - 8 - rightBlock
	if resText != "" {
		resW := len([]rune(resText)) + 2
		if nameMaxW-resW >= 10 {
			nameMaxW -= resW
		} else {
			resText = ""
		}
	}
	if nameMaxW < 8 {
		nameMaxW = 8
	}
	rightAt := x + w - rightBlock

	s.DrawText(nameStart, y, nameStyle, fmt.Sprintf("%-*s", nameMaxW, truncate(c.Name, nameMaxW)))

	if c.Image != "" && !selected {
		img := "[" + imageShort(c.Image) + "]"
		nameLen := len([]rune(c.Name))
		if nameLen+1+len([]rune(img)) <= nameMaxW {
			s.DrawText(nameStart+nameLen+1, y, ui.StyleDim, img)
		}
	}

	if resText != "" {
		resStyle := ui.StyleDim
		if selected {
			resStyle = bg
		}
		s.DrawText(nameStart+nameMaxW+1, y, resStyle, resText)
	}

	s.DrawText(rightAt, y, iconStyle, containerIcon(c.Status))
	s.DrawText(rightAt+2, y, iconStyle, fmt.Sprintf("%-10s", containerStatusShort(c.Status)))

	restartStyle := iconStyle
	if !selected {
		restartStyle = restartCountStyle(c.Restarts)
	}
	s.DrawText(rightAt+12, y, restartStyle, fmt.Sprintf("%d", c.Restarts))
}

func (v *XrayView) drawDetailRow(s *ui.Screen, x, y, w int, row xrayRow, selected bool) {
	var prefix string
	switch {
	case row.podIsLast && row.contIsLast:
		prefix = "        "
	case row.podIsLast && !row.contIsLast:
		prefix = "     │  "
	case !row.podIsLast && row.contIsLast:
		prefix = "  │     "
	default:
		prefix = "  │  │  "
	}

	bg := ui.StyleDefault
	if selected {
		bg = ui.StyleSelected
	}
	s.FillRect(ui.Rect{X: x, Y: y, W: w, H: 1}, ' ', bg)
	s.DrawText(x, y, ui.StyleXrayConnector, prefix)

	msgStyle := ui.StyleDim
	if selected {
		msgStyle = ui.StyleSelected
	}
	if w > 8 {
		s.DrawTextTrunc(x+8, y, w-8, msgStyle, row.detailText)
	}
}

func imageShort(image string) string {
	if i := strings.LastIndex(image, "/"); i >= 0 {
		image = image[i+1:]
	}
	return image
}

func containerStatusStyle(status string) tcell.Style {
	switch status {
	case "Running":
		return ui.StylePodRunning
	case "Waiting":
		return ui.StylePodPending
	case "CrashLoopBackOff", "ImagePullBackOff", "ErrImagePull",
		"CreateContainerConfigError", "InvalidImageName", "OOMKilled":
		return ui.StylePodFailed
	case "Terminated":
		return ui.StylePodDefault
	}
	return ui.StylePodDefault
}

func containerIcon(status string) string {
	switch status {
	case "Running":
		return "✔ "
	case "Waiting":
		return "↻ "
	case "CrashLoopBackOff":
		return "↻ "
	case "Terminated":
		return "⊘ "
	case "OOMKilled", "ImagePullBackOff", "ErrImagePull",
		"CreateContainerConfigError", "InvalidImageName":
		return "✖ "
	}
	return "· "
}

func containerStatusShort(status string) string {
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
