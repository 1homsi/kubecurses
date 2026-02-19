package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
)

var (
	stylePickerBg       = tcell.StyleDefault.Background(tcell.NewRGBColor(13, 14, 20)).Foreground(tcell.NewRGBColor(220, 222, 235))
	stylePickerTitle    = tcell.StyleDefault.Background(tcell.NewRGBColor(13, 14, 20)).Foreground(tcell.NewRGBColor(130, 190, 255)).Bold(true)
	stylePickerItem     = tcell.StyleDefault.Background(tcell.NewRGBColor(18, 22, 38)).Foreground(tcell.NewRGBColor(220, 222, 235))
	stylePickerSelected = tcell.StyleDefault.Background(tcell.NewRGBColor(0, 68, 148)).Foreground(tcell.ColorWhite).Bold(true)
	stylePickerCurrent  = tcell.StyleDefault.Background(tcell.NewRGBColor(18, 22, 38)).Foreground(tcell.NewRGBColor(80, 200, 120))
	stylePickerHint     = tcell.StyleDefault.Background(tcell.NewRGBColor(13, 14, 20)).Foreground(tcell.NewRGBColor(100, 105, 130))
)

// PickContext presents a full-screen context picker and returns the chosen
// context name. Returns ("", true) if the user quits without selecting.
func PickContext(s *Screen, contexts []string, current string) (string, bool) {
	if len(contexts) == 0 {
		return current, false
	}

	// Start selection on the current context.
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
				// Escape with no selection → use current context.
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

	// ── box dimensions ────────────────────────────────────────────────────
	boxW := 52
	if boxW > w-4 {
		boxW = w - 4
	}
	itemW := boxW - 4

	// Height: title(1) + blank(1) + items + blank(1) + hint(1), +2 for border
	maxItems := len(contexts)
	visibleItems := maxItems
	maxVisible := h - 8
	if visibleItems > maxVisible {
		visibleItems = maxVisible
	}
	boxH := visibleItems + 6

	boxX := (w - boxW) / 2
	boxY := (h - boxH) / 2

	// ── background fill ───────────────────────────────────────────────────
	for row := 0; row < h; row++ {
		s.FillRect(Rect{X: 0, Y: row, W: w, H: 1}, ' ', stylePickerBg)
	}

	// ── box border ────────────────────────────────────────────────────────
	drawBox(s, boxX, boxY, boxW, boxH)

	// ── title ─────────────────────────────────────────────────────────────
	title := " Select cluster "
	titleX := boxX + (boxW-len([]rune(title)))/2
	s.DrawText(titleX, boxY, stylePickerTitle, title)

	// ── item list ─────────────────────────────────────────────────────────
	// Scroll offset to keep sel visible.
	scrollOffset := 0
	if sel >= scrollOffset+visibleItems {
		scrollOffset = sel - visibleItems + 1
	}

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

		// Fill item row.
		s.FillRect(Rect{X: boxX + 2, Y: itemY, W: boxW - 4, H: 1}, ' ', style)

		// Cursor indicator.
		cursor := "  "
		if idx == sel {
			cursor = "▶ "
		}

		// Current context marker.
		suffix := ""
		if name == current {
			suffix = " ✦"
		}

		label := fmt.Sprintf("%s%-*s%s", cursor, itemW-len([]rune(cursor))-len([]rune(suffix)), truncatePicker(name, itemW-4), suffix)
		s.DrawText(boxX+2, itemY, style, label)
	}

	// ── hint ──────────────────────────────────────────────────────────────
	hint := " j/k: move   Enter: select   Esc: keep current   q: quit "
	hintX := boxX + (boxW-len([]rune(hint)))/2
	if hintX < boxX+1 {
		hintX = boxX + 1
	}
	s.DrawTextTrunc(hintX, boxY+boxH-2, boxW-2, stylePickerHint, hint)
}

// drawBox draws a rounded box border.
func drawBox(s *Screen, x, y, w, h int) {
	style := stylePickerItem

	// Corners
	s.DrawText(x, y, style, "╭")
	s.DrawText(x+w-1, y, style, "╮")
	s.DrawText(x, y+h-1, style, "╰")
	s.DrawText(x+w-1, y+h-1, style, "╯")

	// Top and bottom edges
	for i := 1; i < w-1; i++ {
		s.DrawText(x+i, y, style, "─")
		s.DrawText(x+i, y+h-1, style, "─")
	}

	// Side edges + interior fill
	for i := 1; i < h-1; i++ {
		s.DrawText(x, y+i, style, "│")
		s.DrawText(x+w-1, y+i, style, "│")
		s.FillRect(Rect{X: x + 1, Y: y + i, W: w - 2, H: 1}, ' ', stylePickerItem)
	}
}

func truncatePicker(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 1 {
		return "…"
	}
	return string(r[:max-1]) + "…"
}
