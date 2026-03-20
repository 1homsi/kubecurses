package views

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

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

func (v *XrayView) RowCount() int { return len(v.rows) }

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
			rows = append(rows, xrayRow{kind: xrkPod, pod: pod, podIsLast: podIsLast})
			for ci, cont := range pod.Containers {
				isLast := ci == len(pod.Containers)-1
				hasDetail := cont.Message != ""
				rows = append(rows, xrayRow{
					kind: xrkContainer, pod: pod, container: cont,
					podIsLast: podIsLast, contIsLast: isLast && !hasDetail,
				})
				if hasDetail {
					rows = append(rows, xrayRow{
						kind: xrkDetail, pod: pod, container: cont,
						podIsLast: podIsLast, contIsLast: isLast, detailText: cont.Message,
					})
				}
			}
		}
	}
	return rows
}

func (v *XrayView) Render(width, height int, state *model.AppState) string {
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

	var lines []string
	lines = append(lines, v.renderHeader(width))
	contentH := height - 1

	if len(v.rows) > 0 {
		if sel < v.scrollOffset {
			v.scrollOffset = sel
		}
		if sel >= v.scrollOffset+contentH {
			v.scrollOffset = sel - contentH + 1
		}
		if v.scrollOffset < 0 {
			v.scrollOffset = 0
		}
	}

	for i := 0; i < contentH; i++ {
		rowIdx := v.scrollOffset + i
		if rowIdx >= len(v.rows) {
			lines = append(lines, ui.FillWidth(width, ui.StyleDefault))
			continue
		}
		lines = append(lines, v.renderRow(width, v.rows[rowIdx], rowIdx == sel))
	}

	return strings.Join(lines, "\n")
}

func (v *XrayView) renderHeader(w int) string {
	_, _, statusAt := dynCols(w)
	readyAt := statusAt + 10
	restAt := readyAt + 6
	ageAt := restAt + 5
	nameW := statusAt - 5
	if nameW < 4 {
		nameW = 4
	}
	hdr := fmt.Sprintf("     %-*s%-10s%-6s%-5sAGE",
		nameW, "NAME", "STATUS", "READY", "REST")
	_ = ageAt
	return ui.StyleHeader.Render(ui.PadRight(hdr, w))
}

func (v *XrayView) renderRow(w int, row xrayRow, selected bool) string {
	switch row.kind {
	case xrkNs:
		return v.renderNsRow(w, row, selected)
	case xrkPod:
		return v.renderPodRow(w, row, selected)
	case xrkContainer:
		return v.renderContainerRow(w, row, selected)
	case xrkDetail:
		return v.renderDetailRow(w, row, selected)
	}
	return ""
}

func (v *XrayView) renderNsRow(w int, row xrayRow, selected bool) string {
	bg := ui.StyleXrayNsHeader
	if selected {
		bg = ui.StyleSelected
	}
	label := fmt.Sprintf("▸ %s  (%d pods)", row.ns, row.nsPodCnt)
	return bg.Render(ui.PadRight(truncate(label, w-2), w))
}

func (v *XrayView) renderPodRow(w int, row xrayRow, selected bool) string {
	_, _, statusAt := dynCols(w)
	readyAt := statusAt + 10
	restAt := readyAt + 6
	ageAt := restAt + 5
	_ = ageAt

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

	var b strings.Builder
	b.WriteString(connStyle.Render(connector))
	b.WriteString(nameStyle.Render(fmt.Sprintf("%-*s", nameW, truncate(p.Name, nameW))))
	b.WriteString(statusStyle.Render(fmt.Sprintf("%-10s", podStatusShort(p.Status))))
	b.WriteString(statusStyle.Render(fmt.Sprintf("%-6s", p.Ready)))

	restartStyle := statusStyle
	if !selected {
		restartStyle = restartCountStyle(p.Restarts)
	}
	b.WriteString(restartStyle.Render(fmt.Sprintf("%-5d", p.Restarts)))
	b.WriteString(statusStyle.Render(formatDuration(p.Age)))

	result := b.String()
	runes := []rune(result)
	if len(runes) < w {
		result += bg.Render(strings.Repeat(" ", w-len(runes)))
	}
	return result
}

func (v *XrayView) renderContainerRow(w int, row xrayRow, selected bool) string {
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

	const rightBlock = 17
	nameStart := 8

	resText := ""
	if w >= 100 && (c.CPURequestM > 0 || c.MemRequestMi > 0) {
		resText = fmt.Sprintf("cpu:%s/%s mem:%s/%s",
			formatCPU(c.CPURequestM), formatCPU(c.CPULimitM),
			formatMem(c.MemRequestMi), formatMem(c.MemLimitMi))
	}

	nameMaxW := w - nameStart - rightBlock
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

	var b strings.Builder
	b.WriteString(connStyle.Render(connector))

	nameText := fmt.Sprintf("%-*s", nameMaxW, truncate(c.Name, nameMaxW))
	if c.Image != "" && !selected {
		img := "[" + imageShort(c.Image) + "]"
		nameLen := len([]rune(c.Name))
		if nameLen+1+len([]rune(img)) <= nameMaxW {
			nameText = c.Name + " " + img
			nameText = fmt.Sprintf("%-*s", nameMaxW, nameText)
		}
	}
	b.WriteString(nameStyle.Render(nameText))

	if resText != "" {
		resStyle := ui.StyleDim
		if selected {
			resStyle = bg
		}
		b.WriteString(resStyle.Render(" " + resText))
	}

	rightAt := w - rightBlock
	_ = rightAt
	b.WriteString(iconStyle.Render(containerIcon(c.Status)))
	b.WriteString(iconStyle.Render(fmt.Sprintf("%-10s", containerStatusShort(c.Status))))

	restartStyle := iconStyle
	if !selected {
		restartStyle = restartCountStyle(c.Restarts)
	}
	b.WriteString(restartStyle.Render(fmt.Sprintf("%d", c.Restarts)))

	result := b.String()
	runes := []rune(result)
	if len(runes) < w {
		result += bg.Render(strings.Repeat(" ", w-len(runes)))
	}
	return result
}

func (v *XrayView) renderDetailRow(w int, row xrayRow, selected bool) string {
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
	connStyle := ui.StyleXrayConnector
	if selected {
		connStyle = ui.StyleSelected
	}
	msgStyle := ui.StyleDim
	if selected {
		msgStyle = ui.StyleSelected
	}

	var b strings.Builder
	b.WriteString(connStyle.Render(prefix))
	if w > 8 {
		b.WriteString(msgStyle.Render(truncate(row.detailText, w-8)))
	}

	result := b.String()
	runes := []rune(result)
	if len(runes) < w {
		result += bg.Render(strings.Repeat(" ", w-len(runes)))
	}
	return result
}

func imageShort(image string) string {
	if i := strings.LastIndex(image, "/"); i >= 0 {
		image = image[i+1:]
	}
	return image
}

func containerStatusStyle(status string) lipgloss.Style {
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
