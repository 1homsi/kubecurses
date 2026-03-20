package views

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/1homsi/kubecurses/internal/model"
	"github.com/1homsi/kubecurses/internal/ui"
)

// rowKind distinguishes display row types.
type rowKind int

const (
	rkPod    rowKind = iota
	rkNode
	rkReason
	rkWarning
)

// ovRow is a flat display row.
type ovRow struct {
	kind     rowKind
	node     model.Node
	pod      model.Pod
	podCount int    // for rkNode: pods scheduled on this node
	text     string // for rkReason / rkWarning
}

// ovCacheKey groups all inputs that affect the row model.
type ovCacheKey struct {
	podGen   uint64
	nodeGen  uint64
	query    string
	ns       string
	nsFilter string
}

// NodeOverviewView renders nodes as section headers with their pods nested below.
type NodeOverviewView struct {
	rows         []ovRow
	scrollOffset int
	cacheValid   bool
	cacheKey     ovCacheKey
	cachedRows   []ovRow
}

// dynCols computes dynamic column widths from row width w.
func dynCols(w int) (nameW, nsW, statusAt int) {
	statusAt = w - 32
	if statusAt < 40 {
		statusAt = 40
	}
	avail := statusAt - 4
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

func (v *NodeOverviewView) buildRows(state *model.AppState, query string) []ovRow {
	byNode := make(map[string][]model.Pod, len(state.Nodes))
	var unscheduled []model.Pod

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

	var warningRows []ovRow
	if len(nodes) > 1 {
		totalPods := 0
		maxPods := 0
		heaviest := ""
		for _, n := range nodes {
			c := len(byNode[n.Name])
			totalPods += c
			if c > maxPods {
				maxPods = c
				heaviest = n.Name
			}
		}
		if totalPods > 0 && maxPods > 0 {
			avg := totalPods / len(nodes)
			if avg > 0 && maxPods >= avg*2 {
				pct := maxPods * 100 / totalPods
				msg := fmt.Sprintf(
					"⚠  Scheduling imbalance: %s has %d%% of pods (%d/%d total)",
					truncate(heaviest, 30), pct, maxPods, totalPods,
				)
				warningRows = append(warningRows, ovRow{kind: rkWarning, text: msg})
			}
		}
	}

	rows := make([]ovRow, 0, len(nodes)+len(state.Pods))
	for _, n := range nodes {
		pods := byNode[n.Name]
		if query != "" && len(pods) == 0 && !nodeMatchesQuery(n, query) {
			continue
		}
		rows = append(rows, ovRow{kind: rkNode, node: n, podCount: len(pods)})
		for _, p := range pods {
			rows = append(rows, ovRow{kind: rkPod, pod: p})
			if p.Status == "Pending" && p.PendingReason != "" {
				rows = append(rows, ovRow{kind: rkReason, text: p.PendingReason})
			}
		}
	}

	if len(unscheduled) > 0 {
		rows = append(rows, ovRow{
			kind: rkNode,
			node: model.Node{Name: "<unscheduled>", Status: "Unknown"},
		})
		for _, p := range unscheduled {
			rows = append(rows, ovRow{kind: rkPod, pod: p})
			if p.Status == "Pending" && p.PendingReason != "" {
				rows = append(rows, ovRow{kind: rkReason, text: p.PendingReason})
			}
		}
	}

	return append(warningRows, rows...)
}

func (v *NodeOverviewView) Render(width, height int, state *model.AppState) string {
	key := ovCacheKey{state.PodGeneration, state.NodeGeneration, state.SearchQuery[model.TabNodeOverview], state.Namespace, state.NamespaceFilter}
	if !v.cacheValid || key != v.cacheKey {
		v.cachedRows = v.buildRows(state, state.SearchQuery[model.TabNodeOverview])
		v.cacheKey = key
		v.cacheValid = true
	}
	v.rows = v.cachedRows
	sel := state.Selection[model.TabNodeOverview]

	if sel >= len(v.rows) && len(v.rows) > 0 {
		sel = len(v.rows) - 1
		state.Selection[model.TabNodeOverview] = sel
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

func (v *NodeOverviewView) renderHeader(w int) string {
	nameW, nsW, _ := dynCols(w)
	hdr := fmt.Sprintf("    %-*s%-*s%-10s%-6s%-5sAGE",
		nameW, "NAME", nsW, "NAMESPACE", "STATUS", "READY", "REST")
	return ui.StyleHeader.Render(ui.PadRight(hdr, w))
}

// RowCount returns the current number of display rows.
func (v *NodeOverviewView) RowCount() int { return len(v.rows) }

// SelectedPodRef returns the namespace and pod name for the row at idx.
func (v *NodeOverviewView) SelectedPodRef(idx int) (ns, pod string) {
	if idx < 0 || idx >= len(v.rows) {
		return "", ""
	}
	row := v.rows[idx]
	if row.kind != rkPod {
		return "", ""
	}
	return row.pod.Namespace, row.pod.Name
}

// SelectedRef returns the kind, namespace, and name for the row at idx.
func (v *NodeOverviewView) SelectedRef(idx int) (kind, ns, name string) {
	if idx < 0 || idx >= len(v.rows) {
		return "", "", ""
	}
	row := v.rows[idx]
	switch row.kind {
	case rkPod:
		return "pod", row.pod.Namespace, row.pod.Name
	case rkNode:
		return "node", "", row.node.Name
	}
	return "", "", ""
}

func (v *NodeOverviewView) renderRow(w int, row ovRow, selected bool) string {
	switch row.kind {
	case rkNode:
		return v.renderNodeRow(w, row.node, row.podCount, selected)
	case rkReason:
		return v.renderReasonRow(w, row.text)
	case rkWarning:
		return v.renderWarningRow(w, row.text)
	default:
		return v.renderPodRow(w, row.pod, selected)
	}
}

var styleTaintBadge = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#D2A032")).
	Background(lipgloss.Color("#161A2E"))

func (v *NodeOverviewView) renderNodeRow(w int, n model.Node, podCount int, selected bool) string {
	nameW, nsW, _ := dynCols(w)

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

	var b strings.Builder
	b.WriteString(dotStyle.Render("●"))
	b.WriteString(base.Render("   "))

	nameText := truncate(n.Name, nameW)
	if len(n.Taints) > 0 && !selected {
		badge := fmt.Sprintf("⚑%d", len(n.Taints))
		badgeLen := len([]rune(badge))
		nameLen := len([]rune(nameText))
		gap := nameW - nameLen - badgeLen
		if gap > 1 {
			b.WriteString(nameStyle.Render(fmt.Sprintf("%-*s", nameLen, nameText)))
			b.WriteString(base.Render(strings.Repeat(" ", gap)))
			b.WriteString(styleTaintBadge.Render(badge))
		} else {
			b.WriteString(nameStyle.Render(fmt.Sprintf("%-*s", nameW, nameText)))
		}
	} else {
		b.WriteString(nameStyle.Render(fmt.Sprintf("%-*s", nameW, nameText)))
	}

	// NAMESPACE column: metrics when available, version otherwise.
	if n.MetricsOK && n.AllocCPUm > 0 {
		cpuPct := int(n.UsedCPUm * 100 / n.AllocCPUm)
		memPct := int(n.UsedMemMi * 100 / n.AllocMemMi)
		cpuStyle, memStyle := metaStyle, metaStyle
		if !selected {
			cpuStyle = metricStyle(cpuPct)
			memStyle = metricStyle(memPct)
		}
		cpu := fmt.Sprintf("cpu:%d%% ", cpuPct)
		mem := fmt.Sprintf("mem:%d%% ", memPct)
		pods := ""
		if n.AllocPods > 0 {
			pods = fmt.Sprintf("%d/%d pods", podCount, n.AllocPods)
		}
		nsContent := cpu + mem + pods
		if len([]rune(nsContent)) > nsW {
			nsContent = string([]rune(nsContent)[:nsW])
		}
		// Render each part with its own style.
		b.WriteString(cpuStyle.Render(truncate(cpu, nsW/2)))
		remaining := nsW - len([]rune(cpu))
		if remaining > 0 {
			b.WriteString(memStyle.Render(truncate(mem, remaining)))
			remaining -= len([]rune(mem))
		}
		if remaining > 0 && pods != "" {
			b.WriteString(metaStyle.Render(truncate(pods, remaining)))
		}
	} else {
		b.WriteString(metaStyle.Render(fmt.Sprintf("%-*s", nsW, truncate(n.Version, nsW))))
	}

	b.WriteString(statusStyle.Render(fmt.Sprintf("%-10s", n.Status)))
	// Skip READY column for nodes.
	b.WriteString(base.Render(strings.Repeat(" ", 11)))
	b.WriteString(metaStyle.Render(formatDuration(n.Age)))

	result := b.String()
	// Pad to width.
	runes := []rune(result)
	if len(runes) < w {
		result += base.Render(strings.Repeat(" ", w-len(runes)))
	}
	return result
}

func metricStyle(pct int) lipgloss.Style {
	switch {
	case pct >= 85:
		return ui.StyleMetricsCrit
	case pct >= 70:
		return ui.StyleMetricsWarn
	}
	return ui.StyleMetricsOK
}

func (v *NodeOverviewView) renderReasonRow(w int, reason string) string {
	text := "    " + truncate("→ "+reason, w-6)
	return ui.StyleDefault.Render(ui.PadRight(text, w))
}

func (v *NodeOverviewView) renderWarningRow(w int, msg string) string {
	text := "  " + truncate(msg, w-4)
	return ui.StyleWarning.Render(ui.PadRight(text, w))
}

func (v *NodeOverviewView) renderPodRow(w int, p model.Pod, selected bool) string {
	nameW, nsW, statusAt := dynCols(w)
	readyAt := statusAt + 10
	restAt := readyAt + 6
	ageAt := restAt + 5
	_ = ageAt

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

	iconStyle := podBaseStyle(p.Status)
	if selected {
		iconStyle = ui.StyleSelected
	}

	var b strings.Builder
	b.WriteString(bg.Render(" "))
	b.WriteString(iconStyle.Render(podStatusIcon(p.Status)))
	b.WriteString(bg.Render("  "))
	b.WriteString(nameStyle.Render(fmt.Sprintf("%-*s", nameW, truncate(p.Name, nameW))))
	b.WriteString(nsStyle.Render(fmt.Sprintf("%-*s", nsW, truncate(p.Namespace, nsW))))
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

func podBaseStyle(status string) lipgloss.Style {
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

func restartCountStyle(restarts int32) lipgloss.Style {
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
