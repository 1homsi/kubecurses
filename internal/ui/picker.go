package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/1homsi/kubecurses/internal/model"
)

// RenderClusterPicker renders the cluster picker overlay and returns it as a multi-line string.
func RenderClusterPicker(state *model.AppState, width, height int) string {
	return renderPickerGeneric(state.ClusterPickerList, state.ClusterPickerCurr, state.ClusterPickerSel, " Select cluster ", width, height)
}

// RenderNamespacePicker renders the namespace picker overlay and returns it as a multi-line string.
func RenderNamespacePicker(state *model.AppState, width, height int) string {
	list := state.NamespacePickerList
	display := make([]string, len(list))
	for i, ns := range list {
		if ns == "" {
			display[i] = "(all namespaces)"
		} else {
			display[i] = ns
		}
	}
	current := state.NamespaceFilter
	if current == "" {
		current = "(all namespaces)"
	}
	return renderPickerGeneric(display, current, state.NamespacePickerSel, " Select namespace ", width, height)
}

var (
	stylePickerBg       = lipgloss.NewStyle().Background(lipgloss.Color("#101010")).Foreground(lipgloss.Color("#DCDEEB"))
	stylePickerTitle    = lipgloss.NewStyle().Background(lipgloss.Color("#101010")).Foreground(lipgloss.Color("#82BEFF")).Bold(true)
	stylePickerItem     = lipgloss.NewStyle().Background(lipgloss.Color("#121626")).Foreground(lipgloss.Color("#DCDEEB"))
	stylePickerSelected = lipgloss.NewStyle().Background(lipgloss.Color("#004494")).Foreground(lipgloss.Color("#FFFFFF")).Bold(true)
	stylePickerCurrent  = lipgloss.NewStyle().Background(lipgloss.Color("#121626")).Foreground(lipgloss.Color("#50C878"))
	stylePickerHint     = lipgloss.NewStyle().Background(lipgloss.Color("#121626")).Foreground(lipgloss.Color("#646982"))
)

func renderPickerGeneric(items []string, current string, sel int, title string, screenW, screenH int) string {
	minW := 40
	boxW := minW
	for _, c := range items {
		if l := len([]rune(c)) + 10; l > boxW {
			boxW = l
		}
	}
	if len([]rune(title))+4 > boxW {
		boxW = len([]rune(title)) + 4
	}
	if boxW > screenW-4 {
		boxW = screenW - 4
	}

	maxVisible := screenH - 8
	if maxVisible < 1 {
		maxVisible = 1
	}
	visibleItems := len(items)
	if visibleItems > maxVisible {
		visibleItems = maxVisible
	}

	innerW := boxW - 2 // usable width between │ and │

	var lines []string

	// Top border with title.
	borderStyle := stylePickerItem
	topBorder := "╭" + strings.Repeat("─", innerW) + "╮"
	titleRunes := []rune(title)
	titleStart := (boxW - len(titleRunes)) / 2
	if titleStart < 1 {
		titleStart = 1
	}
	// Overlay title onto top border.
	topRunes := []rune(topBorder)
	for i, ch := range titleRunes {
		pos := titleStart + i
		if pos < len(topRunes)-1 {
			topRunes[pos] = ch
		}
	}
	lines = append(lines, borderStyle.Render(string(topRunes[:1]))+
		stylePickerTitle.Render(string(topRunes[1:titleStart]))+
		stylePickerTitle.Render(title)+
		stylePickerTitle.Render(string(topRunes[titleStart+len(titleRunes):len(topRunes)-1]))+
		borderStyle.Render(string(topRunes[len(topRunes)-1:])))

	// Blank row after title.
	lines = append(lines, borderStyle.Render("│")+stylePickerItem.Render(strings.Repeat(" ", innerW))+borderStyle.Render("│"))

	// Item rows.
	scrollOffset := 0
	if sel >= scrollOffset+visibleItems {
		scrollOffset = sel - visibleItems + 1
	}

	// Layout: [pad 1][cursor 3][name nameMaxW][marker 3] = innerW
	const (
		padW    = 1
		cursorW = 3
		markerW = 3
	)
	nameMaxW := innerW - padW - cursorW - markerW
	if nameMaxW < 4 {
		nameMaxW = 4
	}

	for i := 0; i < visibleItems; i++ {
		idx := scrollOffset + i
		if idx >= len(items) {
			break
		}
		name := items[idx]

		var style lipgloss.Style
		switch {
		case idx == sel:
			style = stylePickerSelected
		case name == current:
			style = stylePickerCurrent
		default:
			style = stylePickerItem
		}

		cursor := "   "
		if idx == sel {
			cursor = " ▶ "
		}

		nameStr := fmt.Sprintf("%-*s", nameMaxW, Truncate(name, nameMaxW))

		marker := "   "
		if name == current {
			marker = " ✦ "
		}

		rowContent := " " + cursor + nameStr + marker
		// Ensure exact innerW width.
		rowRunes := []rune(rowContent)
		if len(rowRunes) < innerW {
			rowContent += strings.Repeat(" ", innerW-len(rowRunes))
		} else if len(rowRunes) > innerW {
			rowContent = string(rowRunes[:innerW])
		}
		lines = append(lines, borderStyle.Render("│")+style.Render(rowContent)+borderStyle.Render("│"))
	}

	// Blank row before hint.
	lines = append(lines, borderStyle.Render("│")+stylePickerItem.Render(strings.Repeat(" ", innerW))+borderStyle.Render("│"))

	// Hint row.
	hint := "  j/k: move  Enter: select  Esc: cancel  q: quit  "
	hintContent := fmt.Sprintf("%-*s", innerW, Truncate(hint, innerW))
	lines = append(lines, borderStyle.Render("│")+stylePickerHint.Render(hintContent)+borderStyle.Render("│"))

	// Bottom border.
	lines = append(lines, borderStyle.Render("╰"+strings.Repeat("─", innerW)+"╯"))

	return strings.Join(lines, "\n")
}

// RenderBorder renders only the rounded-corner box border as []string rows.
func RenderBorder(w, h int, style lipgloss.Style) []string {
	if w < 2 || h < 2 {
		return nil
	}
	lines := make([]string, h)
	lines[0] = style.Render("╭" + strings.Repeat("─", w-2) + "╮")
	for i := 1; i < h-1; i++ {
		lines[i] = style.Render("│") + strings.Repeat(" ", w-2) + style.Render("│")
	}
	lines[h-1] = style.Render("╰" + strings.Repeat("─", w-2) + "╯")
	return lines
}

// RenderHexFill returns []string rows representing a filled hexagonal polygon.
// Every cell within the hex shape is filled: interior cells receive fillStyle
// and perimeter cells receive perimStyle. The hex geometry matches RenderHexBorder
// (numTaper=2, step=2).
func RenderHexFill(w, h int, fillStyle, perimStyle lipgloss.Style) []string {
	const step = 2
	const numTaper = 2
	minW := 2*(numTaper-1)*step + 4
	if w < minW {
		w = minW
	}
	if h < 2*numTaper {
		h = 2 * numTaper
	}

	rows := make([]string, h)
	for ry := 0; ry < h; ry++ {
		indent := 0
		if ry < numTaper {
			indent = (numTaper - 1 - ry) * step
		} else if fromBot := h - 1 - ry; fromBot < numTaper {
			indent = (numTaper - 1 - fromBot) * step
		}

		left := indent
		right := w - 1 - indent
		if left > right {
			rows[ry] = strings.Repeat(" ", w)
			continue
		}
		span := right - left + 1

		if indent > 0 || span <= 2 {
			// Cap rows (indented) are entirely perimeter.
			rows[ry] = strings.Repeat(" ", left) +
				perimStyle.Render(strings.Repeat(" ", span)) +
				strings.Repeat(" ", w-left-span)
		} else {
			// Body row: perimeter edges + interior fill.
			interior := ""
			if span > 2 {
				interior = fillStyle.Render(strings.Repeat(" ", span-2))
			}
			rows[ry] = perimStyle.Render(" ") +
				interior +
				perimStyle.Render(" ")
		}
	}
	return rows
}

// RenderHexBorder returns []string rows representing a hexagonal box border.
func RenderHexBorder(w, h int, style lipgloss.Style) []string {
	const step = 2
	numTaper := 2
	minW := 2*(numTaper-1)*step + 4
	if w < minW {
		w = minW
	}
	if h < 2*numTaper {
		h = 2 * numTaper
	}

	rows := make([]string, h)
	for k := 0; k < numTaper; k++ {
		indent := (numTaper - 1 - k) * step
		row := strings.Repeat(" ", indent) + style.Render("╱")
		if k == 0 {
			row += style.Render(strings.Repeat("─", w-2*indent-2))
		} else {
			row += strings.Repeat(" ", w-2*indent-2)
		}
		row += style.Render("╲")
		if indent > 0 {
			row += strings.Repeat(" ", indent)
		}
		rows[k] = row
	}

	for i := numTaper; i < h-numTaper; i++ {
		rows[i] = style.Render("│") + strings.Repeat(" ", w-2) + style.Render("│")
	}

	for k := 0; k < numTaper; k++ {
		indent := k * step
		row := strings.Repeat(" ", indent) + style.Render("╲")
		if k == numTaper-1 {
			row += style.Render(strings.Repeat("─", w-2*indent-2))
		} else {
			row += strings.Repeat(" ", w-2*indent-2)
		}
		row += style.Render("╱")
		if indent > 0 {
			row += strings.Repeat(" ", indent)
		}
		rows[h-numTaper+k] = row
	}
	return rows
}
