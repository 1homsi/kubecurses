package ui

import (
	"github.com/gdamore/tcell/v2"

	"github.com/1homsi/kubecurses/internal/model"
)

// DrawClusterPicker renders the cluster picker overlay using state from AppState.
// Called from draw() when state.ClusterPickerMode is true.
func DrawClusterPicker(s *Screen, state *model.AppState) {
	drawPicker(s, state.ClusterPickerList, state.ClusterPickerCurr, state.ClusterPickerSel)
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
	w, h := s.Size()
	s.Clear()

	// Fill entire screen with dark background.
	for row := 0; row < h; row++ {
		s.FillRect(Rect{X: 0, Y: row, W: w, H: 1}, ' ', stylePickerBg)
	}

	// ── box sizing ────────────────────────────────────────────────────────
	// Width: fit the longest context name + chrome, capped at screen width-4.
	minW := 40
	boxW := minW
	for _, c := range contexts {
		if l := len([]rune(c)) + 10; l > boxW { // 10 = cursor + marker + padding
			boxW = l
		}
	}
	if boxW > w-4 {
		boxW = w - 4
	}

	// Height: border(2) + blank(1) + items + blank(1) + hint(1) + blank(1)
	maxVisible := h - 8
	if maxVisible < 1 {
		maxVisible = 1
	}
	visibleItems := len(contexts)
	if visibleItems > maxVisible {
		visibleItems = maxVisible
	}
	boxH := visibleItems + 6

	boxX := (w - boxW) / 2
	boxY := (h - boxH) / 2

	// ── border ────────────────────────────────────────────────────────────
	drawBox(s, boxX, boxY, boxW, boxH)

	// Title centred on top border.
	title := " Select cluster "
	titleX := boxX + (boxW-len([]rune(title)))/2
	if titleX < boxX+1 {
		titleX = boxX + 1
	}
	s.DrawText(titleX, boxY, stylePickerTitle, title)

	// ── scroll offset ─────────────────────────────────────────────────────
	scrollOffset := 0
	if sel >= scrollOffset+visibleItems {
		scrollOffset = sel - visibleItems + 1
	}

	// ── items ─────────────────────────────────────────────────────────────
	// Layout per row (all inside the border, i.e. x in [boxX+1, boxX+boxW-2]):
	//   col boxX+1        : 1 space padding
	//   col boxX+2        : cursor "▶" or " "
	//   col boxX+3        : 1 space
	//   col boxX+4 …      : context name (truncated)
	//   col boxX+boxW-4   : " ✦ " marker for current context (3 chars)
	//   col boxX+boxW-1   : border "│"

	const (
		cursorCol  = 2 // offset from boxX
		nameCol    = 4 // offset from boxX
		markerW    = 3 // " ✦ "
		innerPad   = 1 // left padding inside border
	)
	nameMaxW := boxW - nameCol - 1 - markerW // leave room for marker + right border

	for i := 0; i < visibleItems; i++ {
		idx := scrollOffset + i
		if idx >= len(contexts) {
			break
		}
		name := contexts[idx]
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

		// Fill the full inner row width (border-to-border minus the │ chars).
		s.FillRect(Rect{X: boxX + innerPad, Y: itemY, W: boxW - 2, H: 1}, ' ', style)

		// Cursor indicator.
		if idx == sel {
			s.DrawText(boxX+cursorCol, itemY, style, "▶")
		}

		// Context name (truncated so it never touches the marker column).
		s.DrawTextTrunc(boxX+nameCol, itemY, nameMaxW, style, name)

		// Current-context marker, pinned to the right inside the border.
		if name == current {
			s.DrawText(boxX+boxW-1-markerW, itemY, style, " ✦ ")
		}
	}

	// ── hint bar ──────────────────────────────────────────────────────────
	hintY := boxY + boxH - 2
	hint := "  j/k: move  Enter: select  Esc: keep current  q: quit  "
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

// DrawHexBorder draws a box border with diagonal corners (╱ ╲) to give a
// hexagonal feel. Interior is not touched.
//
//	╱──────╲
//	│      │
//	╲──────╱
func DrawHexBorder(s *Screen, x, y, w, h int, style tcell.Style) {
	s.DrawText(x, y, style, "╱")
	s.DrawText(x+w-1, y, style, "╲")
	s.DrawText(x, y+h-1, style, "╲")
	s.DrawText(x+w-1, y+h-1, style, "╱")
	for i := 1; i < w-1; i++ {
		s.DrawText(x+i, y, style, "─")
		s.DrawText(x+i, y+h-1, style, "─")
	}
	for i := 1; i < h-1; i++ {
		s.DrawText(x, y+i, style, "│")
		s.DrawText(x+w-1, y+i, style, "│")
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
