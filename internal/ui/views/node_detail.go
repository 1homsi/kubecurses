package views

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"

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

func (v *NodeDetailView) Draw(s *ui.Screen, r ui.Rect, state *model.AppState) {
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

	// ── Header ────────────────────────────────────────────────────────────────
	titleY := r.Y
	dot := "●"
	dotStyle := ui.StyleNodeReadyDot
	if selNode != nil && selNode.Status != "Ready" {
		dotStyle = ui.StyleNodeNotReadyDot
	}
	s.DrawText(r.X, titleY, dotStyle, dot)
	nodeTitle := fmt.Sprintf(" Node: %s", state.HeatmapDetailNode)
	s.DrawTextTrunc(r.X+1, titleY, r.W-2, ui.StyleNodeName, nodeTitle)

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
		s.DrawTextTrunc(r.X+r.W-len([]rune(met))-1, titleY, len([]rune(met)), ui.StyleNodeMeta, met)
	}

	// ── Taints ────────────────────────────────────────────────────────────────
	taintsY := r.Y + 1
	if selNode != nil && len(selNode.Taints) > 0 {
		s.FillRect(ui.Rect{X: r.X, Y: taintsY, W: r.W, H: 1}, ' ', ui.StyleDefault)
		s.DrawTextTrunc(r.X+2, taintsY, r.W-4, ui.StyleDim, "Taints: "+strings.Join(selNode.Taints, ", "))
	} else {
		s.FillRect(ui.Rect{X: r.X, Y: taintsY, W: r.W, H: 1}, ' ', ui.StyleDefault)
	}

	// ── Column headers ────────────────────────────────────────────────────────
	hdrY := r.Y + 2
	s.DrawTextTrunc(r.X, hdrY, r.W, ui.StyleHeader,
		fmt.Sprintf("  %-5s %-*s %-20s %8s %8s",
			"", nameColW(r.W), "POD", "NAMESPACE", "RESTARTS", "AGE"))

	// ── Pod rows ──────────────────────────────────────────────────────────────
	contentTop := r.Y + 3
	contentH := r.H - 5 // header + taints + col-hdr + hint + padding
	if contentH < 1 {
		contentH = 1
	}

	// Compute scroll offset to keep sel in view.
	scrollOffset := 0
	if sel >= contentH {
		scrollOffset = sel - contentH + 1
	}

	nameW := nameColW(r.W)
	for i := scrollOffset; i < len(pods) && i-scrollOffset < contentH; i++ {
		p := pods[i]
		rowY := contentTop + i - scrollOffset
		isSelected := i == sel

		var style = ui.StyleDefault
		if isSelected {
			style = ui.StyleSelected
		}

		icon := podIcon(p.Status, state.NoIcons)
		iconStyle := podStatusStyle(p.Status)
		if isSelected {
			iconStyle = style
		}

		// Fill row background.
		s.FillRect(ui.Rect{X: r.X, Y: rowY, W: r.W, H: 1}, ' ', style)

		// Icon.
		s.DrawText(r.X+2, rowY, iconStyle, icon)

		// Pod name.
		nameStart := r.X + 2 + len([]rune(icon)) + 1
		s.DrawTextTrunc(nameStart, rowY, nameW, style, p.Name)

		// Namespace.
		nsStart := nameStart + nameW + 1
		nsStyle := ui.StyleNamespace
		if isSelected {
			nsStyle = style
		}
		s.DrawTextTrunc(nsStart, rowY, 20, nsStyle, p.Namespace)

		// Restarts.
		rstStart := nsStart + 21
		s.DrawTextTrunc(rstStart, rowY, 8, style, fmt.Sprintf("%8d", p.Restarts))

		// Age.
		ageStart := rstStart + 9
		s.DrawTextTrunc(ageStart, rowY, 8, style, fmt.Sprintf("%8s", p.Age))
	}

	if len(pods) == 0 {
		s.DrawText(r.X+2, contentTop, ui.StyleDim, "No pods on this node.")
	}

	// ── Hint bar ──────────────────────────────────────────────────────────────
	hintY := r.Y + r.H - 1
	s.DrawTextTrunc(r.X+2, hintY, r.W-4, ui.StyleDim,
		fmt.Sprintf("j/k: navigate  l: open logs  Esc: back to heatmap  %d pods", len(pods)))
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

// podStatusStyle returns the tcell.Style for a pod's status icon.
func podStatusStyle(status string) tcell.Style {
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
