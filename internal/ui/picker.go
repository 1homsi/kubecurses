package ui

import (
	"github.com/gdamore/tcell/v2"

	"github.com/1homsi/kubecurses/internal/model"
)

// DrawClusterPicker renders the cluster picker overlay using state from AppState.
// Called from draw() when state.ClusterPickerMode is true.
func DrawClusterPicker(s *Screen, state *model.AppState) {
	drawPickerGeneric(s, state.ClusterPickerList, state.ClusterPickerCurr, state.ClusterPickerSel, " Select cluster ")
}

// DrawNamespacePicker renders the namespace picker overlay.
// Called from draw() when state.NamespacePickerMode is true.
func DrawNamespacePicker(s *Screen, state *model.AppState) {
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
	drawPickerGeneric(s, display, current, state.NamespacePickerSel, " Select namespace ")
}

var (
	stylePickerBg       = tcell.StyleDefault.Background(tcell.NewRGBColor(13, 14, 20)).Foreground(tcell.NewRGBColor(220, 222, 235))
	stylePickerTitle    = tcell.StyleDefault.Background(tcell.NewRGBColor(13, 14, 20)).Foreground(tcell.NewRGBColor(130, 190, 255)).Bold(true)
	stylePickerItem     = tcell.StyleDefault.Background(tcell.NewRGBColor(18, 22, 38)).Foreground(tcell.NewRGBColor(220, 222, 235))
	stylePickerSelected = tcell.StyleDefault.Background(tcell.NewRGBColor(0, 68, 148)).Foreground(tcell.ColorWhite).Bold(true)
	stylePickerCurrent  = tcell.StyleDefault.Background(tcell.NewRGBColor(18, 22, 38)).Foreground(tcell.NewRGBColor(80, 200, 120))
	stylePickerHint     = tcell.StyleDefault.Background(tcell.NewRGBColor(18, 22, 38)).Foreground(tcell.NewRGBColor(100, 105, 130))
)

// PickContext presents a full-screen context picker and returns the chosen
// context name. Returns ("", true) if the user quits without selecting.
func PickContext(s *Screen, contexts []string, current string) (string, bool) {
	if len(contexts) == 0 {
		return current, false
	}

	sel := 0
	for i, c := range contexts {
		if c == current {
			sel = i
			break
		}
	}

	for {
		drawPicker(s, contexts, current, sel)
		s.Show()

		ev := s.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventResize:
			s.Sync()
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyCtrlC:
				return "", true
			case tcell.KeyEnter:
				return contexts[sel], false
			case tcell.KeyUp:
				if sel > 0 {
					sel--
				}
			case tcell.KeyDown:
				if sel < len(contexts)-1 {
					sel++
				}
			case tcell.KeyEsc:
				return current, false
			}
			switch ev.Rune() {
			case 'q':
				return "", true
			case 'k':
				if sel > 0 {
					sel--
				}
			case 'j':
				if sel < len(contexts)-1 {
					sel++
				}
			case 'g':
				sel = 0
			case 'G':
				sel = len(contexts) - 1
			}
		}
	}
}

func drawPicker(s *Screen, contexts []string, current string, sel int) {
	drawPickerGeneric(s, contexts, current, sel, " Select cluster ")
}

func drawPickerGeneric(s *Screen, items []string, current string, sel int, title string) {
	w, h := s.Size()
	s.Clear()

	for row := 0; row < h; row++ {
		s.FillRect(Rect{X: 0, Y: row, W: w, H: 1}, ' ', stylePickerBg)
	}

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
	if boxW > w-4 {
		boxW = w - 4
	}

	maxVisible := h - 8
	if maxVisible < 1 {
		maxVisible = 1
	}
	visibleItems := len(items)
	if visibleItems > maxVisible {
		visibleItems = maxVisible
	}
	boxH := visibleItems + 6

	boxX := (w - boxW) / 2
	boxY := (h - boxH) / 2

	drawBox(s, boxX, boxY, boxW, boxH)

	titleX := boxX + (boxW-len([]rune(title)))/2
	if titleX < boxX+1 {
		titleX = boxX + 1
	}
	s.DrawText(titleX, boxY, stylePickerTitle, title)

	scrollOffset := 0
	if sel >= scrollOffset+visibleItems {
		scrollOffset = sel - visibleItems + 1
	}

	const (
		cursorCol = 2
		nameCol   = 4
		markerW   = 3
		innerPad  = 1
	)
	nameMaxW := boxW - nameCol - 1 - markerW

	for i := 0; i < visibleItems; i++ {
		idx := scrollOffset + i
		if idx >= len(items) {
			break
		}
		name := items[idx]
		itemY := boxY + 2 + i

		var style tcell.Style
		switch {
		case idx == sel:
			style = stylePickerSelected
		case name == current:
			style = stylePickerCurrent
		default:
			style = stylePickerItem
		}

		s.FillRect(Rect{X: boxX + innerPad, Y: itemY, W: boxW - 2, H: 1}, ' ', style)

		if idx == sel {
			s.DrawText(boxX+cursorCol, itemY, style, "▶")
		}

		s.DrawTextTrunc(boxX+nameCol, itemY, nameMaxW, style, name)

		if name == current {
			s.DrawText(boxX+boxW-1-markerW, itemY, style, " ✦ ")
		}
	}

	hintY := boxY + boxH - 2
	hint := "  j/k: move  Enter: select  Esc: cancel  q: quit  "
	s.FillRect(Rect{X: boxX + 1, Y: hintY, W: boxW - 2, H: 1}, ' ', stylePickerHint)
	s.DrawTextTrunc(boxX+2, hintY, boxW-4, stylePickerHint, hint)
}

// drawBox draws a rounded-corner box border and fills the interior.
func drawBox(s *Screen, x, y, w, h int) {
	drawBorderOnly(s, x, y, w, h, stylePickerItem)
	for i := 1; i < h-1; i++ {
		s.FillRect(Rect{X: x + 1, Y: y + i, W: w - 2, H: 1}, ' ', stylePickerItem)
	}
}

// DrawBorderOnly draws only the rounded-corner box border using the given style.
// The interior is not touched — callers fill it themselves.
func DrawBorderOnly(s *Screen, x, y, w, h int, style tcell.Style) {
	drawBorderOnly(s, x, y, w, h, style)
}

// DrawHexFill paints a filled hexagonal polygon on the screen using spaces.
// Every cell within the hex shape is filled: interior cells receive fillStyle
// and perimeter cells receive perimStyle. Only the background components of the
// styles are visible (no special characters are written). The hex geometry
// matches DrawHexBorder (numTaper=2, step=2).
//
// When fillStyle == perimStyle the result is a seamless solid hex. Set a
// contrasting perimStyle to create a glowing border effect for selection.
func DrawHexFill(s *Screen, x, y, w, h int, fillStyle, perimStyle tcell.Style) {
	const step = 2
	const numTaper = 2
	minW := 2*(numTaper-1)*step + 4
	if w < minW {
		w = minW
	}
	if h < 2*numTaper {
		h = 2 * numTaper
	}

	for ry := 0; ry < h; ry++ {
		// Mirror of DrawHexBorder's indent arithmetic.
		indent := 0
		if ry < numTaper {
			indent = (numTaper - 1 - ry) * step
		} else if fromBot := h - 1 - ry; fromBot < numTaper {
			indent = (numTaper - 1 - fromBot) * step
		}

		left := x + indent
		right := x + w - 1 - indent
		if left > right {
			continue
		}
		span := right - left + 1

		if indent > 0 || span <= 2 {
			// Cap rows (indented) are entirely perimeter — the rounded "points"
			// of the hex shape need a solid highlight so the ring closes cleanly.
			s.FillRect(Rect{X: left, Y: y + ry, W: span, H: 1}, ' ', perimStyle)
		} else {
			// Body row: interior between the perimeter edges.
			if span > 2 {
				s.FillRect(Rect{X: left + 1, Y: y + ry, W: span - 2, H: 1}, ' ', fillStyle)
			}
			s.FillRect(Rect{X: left, Y: y + ry, W: 1, H: 1}, ' ', perimStyle)
			s.FillRect(Rect{X: right, Y: y + ry, W: 1, H: 1}, ' ', perimStyle)
		}
	}
}

// DrawHexBorder draws a hexagonal box border with a multi-row taper:
//
//	   ╱──────────╲     ← top cap     (indent = numTaper-1)
//	  ╱            ╲    ← taper row   (indent = numTaper-2)
//	 ╱              ╲   ← taper row   (indent = 1)
//	╱                ╲  ← shoulder    (indent = 0, full width)
//	│  content        │ ← body rows
//	╲                ╱
//	 ╲              ╱
//	  ╲            ╱
//	   ╲──────────╱     ← bottom cap
//
// numTaper = w/8 (min 2) — the number of diagonal rows on each side.
// This creates a genuine hexagonal silhouette at any box width.
// Minimum h = 2*numTaper (no body rows).
func DrawHexBorder(s *Screen, x, y, w, h int, style tcell.Style) {
	const step = 2 // chars narrowed per taper row on each side
	numTaper := 2
	// Minimum width: cap must be at least 4 chars wide (2 corners + 2 dashes).
	minW := 2*(numTaper-1)*step + 4
	if w < minW {
		w = minW
	}
	if h < 2*numTaper {
		h = 2 * numTaper
	}

	// Top taper: only the topmost row (k=0, cap/edge) gets dashes.
	for k := 0; k < numTaper; k++ {
		indent := (numTaper - 1 - k) * step
		row := y + k
		s.DrawText(x+indent, row, style, "╱")
		if k == 0 {
			for i := indent + 1; i < w-indent-1; i++ {
				s.DrawText(x+i, row, style, "─")
			}
		}
		s.DrawText(x+w-indent-1, row, style, "╲")
	}

	// Vertical body sides.
	for i := numTaper; i < h-numTaper; i++ {
		s.DrawText(x, y+i, style, "│")
		s.DrawText(x+w-1, y+i, style, "│")
	}

	// Bottom taper: only the bottommost row (k=numTaper-1, cap/edge) gets dashes.
	for k := 0; k < numTaper; k++ {
		indent := k * step
		row := y + h - numTaper + k
		s.DrawText(x+indent, row, style, "╲")
		if k == numTaper-1 {
			for i := indent + 1; i < w-indent-1; i++ {
				s.DrawText(x+i, row, style, "─")
			}
		}
		s.DrawText(x+w-indent-1, row, style, "╱")
	}
}

// drawBorderOnly draws only the rounded-corner box border using the given style.
// The interior is not touched — callers fill it themselves.
func drawBorderOnly(s *Screen, x, y, w, h int, style tcell.Style) {
	s.DrawText(x, y, style, "╭")
	s.DrawText(x+w-1, y, style, "╮")
	s.DrawText(x, y+h-1, style, "╰")
	s.DrawText(x+w-1, y+h-1, style, "╯")
	for i := 1; i < w-1; i++ {
		s.DrawText(x+i, y, style, "─")
		s.DrawText(x+i, y+h-1, style, "─")
	}
	for i := 1; i < h-1; i++ {
		s.DrawText(x, y+i, style, "│")
		s.DrawText(x+w-1, y+i, style, "│")
	}
}
