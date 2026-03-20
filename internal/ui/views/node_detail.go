package views

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/1homsi/kubecurses/internal/model"
	"github.com/1homsi/kubecurses/internal/ui"
)

// NodeDetailView shows all pods running on a specific node.
// Activated when state.HeatmapNodeDetail is true.
type NodeDetailView struct{}

// NodeDetailPods returns the pods for the node named by state.HeatmapDetailNode,
// sorted by status severity then name.
func NodeDetailPods(state *model.AppState) []model.Pod {
	var pods []model.Pod
	for _, p := range state.Pods {
		if p.Node == state.HeatmapDetailNode {
			pods = append(pods, p)
		}
	}
	sort.Slice(pods, func(i, j int) bool {
		oi, oj := nodeDetailOrder(pods[i].Status), nodeDetailOrder(pods[j].Status)
		if oi != oj {
			return oi < oj
		}
		return pods[i].Name < pods[j].Name
	})
	return pods
}

func nodeDetailOrder(status string) int {
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

func (v *NodeDetailView) Render(width, height int, state *model.AppState) string {
	// Find the node.
	var selNode *model.Node
	for i := range state.Nodes {
		if state.Nodes[i].Name == state.HeatmapDetailNode {
			selNode = &state.Nodes[i]
			break
		}
	}

	pods := NodeDetailPods(state)
	sel := state.HeatmapDetailSel
	if sel >= len(pods) {
		sel = 0
	}

	var lines []string

	// ── Header ────────────────────────────────────────────────────────────────
	dot := "●"
	dotStyle := ui.StyleNodeReadyDot
	if selNode != nil && selNode.Status != "Ready" {
		dotStyle = ui.StyleNodeNotReadyDot
	}
	nodeTitle := fmt.Sprintf(" Node: %s", state.HeatmapDetailNode)

	var hdrB strings.Builder
	hdrB.WriteString(dotStyle.Render(dot))
	hdrB.WriteString(ui.StyleNodeName.Render(truncate(nodeTitle, width-2)))

	// Metrics on the same line (right-aligned) if available.
	if selNode != nil && selNode.MetricsOK {
		cpuPct := int64(0)
		if selNode.AllocCPUm > 0 {
			cpuPct = selNode.UsedCPUm * 100 / selNode.AllocCPUm
		}
		memPct := int64(0)
		if selNode.AllocMemMi > 0 {
			memPct = selNode.UsedMemMi * 100 / selNode.AllocMemMi
		}
		met := fmt.Sprintf("CPU %3d%%  MEM %3d%%  ", cpuPct, memPct)
		// Pad between title and metrics
		titleLen := 1 + len([]rune(truncate(nodeTitle, width-2)))
		metLen := len([]rune(met))
		gap := width - titleLen - metLen
		if gap > 0 {
			hdrB.WriteString(ui.StyleNodeName.Render(strings.Repeat(" ", gap)))
			hdrB.WriteString(ui.StyleNodeMeta.Render(met))
		}
	}

	hdrResult := hdrB.String()
	hdrRunes := len([]rune(hdrResult))
	// Approximate — pad with background
	if hdrRunes < width {
		hdrResult += ui.StyleDefault.Render(strings.Repeat(" ", width-hdrRunes))
	}
	lines = append(lines, hdrResult)

	// ── Taints ────────────────────────────────────────────────────────────────
	if selNode != nil && len(selNode.Taints) > 0 {
		taintsText := "Taints: " + strings.Join(selNode.Taints, ", ")
		taintsLine := ui.StyleDim.Render("  " + truncate(taintsText, width-4))
		lines = append(lines, ui.PadRight(taintsLine, width))
	} else {
		lines = append(lines, ui.FillWidth(width, ui.StyleDefault))
	}

	// ── Column headers ────────────────────────────────────────────────────────
	nameW := nameColW(width)
	hdr := fmt.Sprintf("  %-5s %-*s %-20s %8s %8s",
		"", nameW, "POD", "NAMESPACE", "RESTARTS", "AGE")
	lines = append(lines, ui.StyleHeader.Render(ui.PadRight(hdr, width)))

	// ── Pod rows ──────────────────────────────────────────────────────────────
	contentH := height - 5 // header + taints + col-hdr + hint + padding
	if contentH < 1 {
		contentH = 1
	}

	// Compute scroll offset to keep sel in view.
	scrollOffset := 0
	if sel >= contentH {
		scrollOffset = sel - contentH + 1
	}

	for i := scrollOffset; i < len(pods) && i-scrollOffset < contentH; i++ {
		p := pods[i]
		isSelected := i == sel

		style := ui.StyleDefault
		if isSelected {
			style = ui.StyleSelected
		}

		icon := podIcon(p.Status, state.NoIcons)
		iconStyle := podStatusStyle(p.Status)
		if isSelected {
			iconStyle = style
		}

		nsStyle := ui.StyleNamespace
		if isSelected {
			nsStyle = style
		}

		var rowB strings.Builder
		rowB.WriteString(iconStyle.Render("  " + icon + " "))
		rowB.WriteString(style.Render(fmt.Sprintf("%-*s ", nameW, truncate(p.Name, nameW))))
		rowB.WriteString(nsStyle.Render(fmt.Sprintf("%-20s ", truncate(p.Namespace, 20))))
		rowB.WriteString(style.Render(fmt.Sprintf("%8d ", p.Restarts)))
		rowB.WriteString(style.Render(fmt.Sprintf("%8s", p.Age)))

		result := rowB.String()
		runes := []rune(result)
		if len(runes) < width {
			result += style.Render(strings.Repeat(" ", width-len(runes)))
		}
		lines = append(lines, result)
	}

	if len(pods) == 0 {
		line := ui.StyleDim.Render("  No pods on this node.")
		lines = append(lines, ui.PadRight(line, width))
	}

	// Fill remaining rows.
	for len(lines) < height-1 {
		lines = append(lines, ui.FillWidth(width, ui.StyleDefault))
	}

	// ── Hint bar ──────────────────────────────────────────────────────────────
	hint := fmt.Sprintf("j/k: navigate  l: open logs  Esc: back to heatmap  %d pods", len(pods))
	lines = append(lines, ui.StyleDim.Render(ui.PadRight("  "+truncate(hint, width-4), width)))

	// Ensure exact height.
	for len(lines) < height {
		lines = append(lines, ui.FillWidth(width, ui.StyleDefault))
	}
	if len(lines) > height {
		lines = lines[:height]
	}

	return strings.Join(lines, "\n")
}

// nameColW returns the pod-name column width based on available screen width.
func nameColW(screenW int) int {
	const (
		colNs       = 21
		colRestarts = 9
		colAge      = 8
		minNameW    = 10
	)
	w := screenW - 2 - colAge - colNs - colRestarts
	if w < minNameW {
		w = minNameW
	}
	return w
}

// podStatusStyle returns the lipgloss.Style for a pod's status icon.
func podStatusStyle(status string) lipgloss.Style {
	switch status {
	case "Running":
		return ui.StylePodRunning
	case "Pending":
		return ui.StylePodPending
	case "Terminating":
		return ui.StylePodDefault
	case "Failed", "CrashLoopBackOff", "OOMKilled", "ImagePullBackOff",
		"ErrImagePull", "CreateContainerConfigError", "InvalidImageName":
		return ui.StylePodFailed
	}
	return ui.StylePodDefault
}
