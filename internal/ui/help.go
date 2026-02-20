package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
)

var (
	styleHelpBg      = tcell.StyleDefault.Background(tcell.NewRGBColor(18, 22, 38)).Foreground(tcell.NewRGBColor(220, 222, 235))
	styleHelpTitle   = tcell.StyleDefault.Background(tcell.NewRGBColor(18, 22, 38)).Foreground(tcell.NewRGBColor(130, 190, 255)).Bold(true)
	styleHelpKey     = tcell.StyleDefault.Background(tcell.NewRGBColor(18, 22, 38)).Foreground(tcell.NewRGBColor(130, 190, 255))
	styleHelpSection = tcell.StyleDefault.Background(tcell.NewRGBColor(18, 22, 38)).Foreground(tcell.NewRGBColor(80, 200, 120)).Bold(true)
	styleHelpDim     = tcell.StyleDefault.Background(tcell.NewRGBColor(18, 22, 38)).Foreground(tcell.NewRGBColor(100, 105, 130))
)

type helpLine struct {
	key  string // empty = section header (uses desc) or blank line (both empty)
	desc string
}

var helpLines = []helpLine{
	{key: "", desc: "Navigation"},
	{key: "j / k  ↓ / ↑", desc: "Move down / up"},
	{key: "PgDn / PgUp", desc: "Page down / up"},
	{key: "", desc: ""},
	{key: "", desc: "Tabs"},
	{key: "Tab / Shift+Tab", desc: "Next / previous tab"},
	{key: "1  2  3  4  5", desc: "Jump to tab directly"},
	{key: "", desc: ""},
	{key: "", desc: "Actions"},
	{key: "l", desc: "Stream logs for selected pod"},
	{key: "c", desc: "Switch cluster / context"},
	{key: "/", desc: "Search / filter"},
	{key: "Esc", desc: "Clear search or close"},
	{key: "r", desc: "Manual refresh"},
	{key: "?", desc: "Toggle this help"},
	{key: "q  Ctrl+C", desc: "Quit"},
	{key: "", desc: ""},
	{key: "", desc: "Logs view"},
	{key: "s", desc: "Toggle autoscroll"},
	{key: "j / k  PgDn / PgUp", desc: "Scroll"},
	{key: "Esc", desc: "Close logs"},
}

// DrawHelp renders a centered help overlay on top of the current screen.
// Press any key to dismiss.
func DrawHelp(s *Screen, screenW, screenH int) {
	const boxW = 46
	w := boxW
	if w > screenW-4 {
		w = screenW - 4
	}
	// border(2) + blank after title(1) + content(N) + blank before hint(1) + hint(1)
	h := len(helpLines) + 5
	bx := (screenW - w) / 2
	by := (screenH - h) / 2

	for row := by; row < by+h; row++ {
		s.FillRect(Rect{X: bx, Y: row, W: w, H: 1}, ' ', styleHelpBg)
	}
	drawBox(s, bx, by, w, h)

	title := " Keyboard Shortcuts "
	titleX := bx + (w-len([]rune(title)))/2
	s.DrawText(titleX, by, styleHelpTitle, title)

	const keyColW = 18
	y := by + 2
	for _, l := range helpLines {
		if l.key == "" {
			if l.desc != "" {
				s.DrawText(bx+2, y, styleHelpSection, l.desc)
			}
			y++
			continue
		}
		s.DrawText(bx+2, y, styleHelpKey, fmt.Sprintf("%-*s", keyColW, l.key))
		s.DrawText(bx+2+keyColW, y, styleHelpBg, l.desc)
		y++
	}

	hintY := by + h - 2
	s.FillRect(Rect{X: bx + 1, Y: hintY, W: w - 2, H: 1}, ' ', styleHelpDim)
	s.DrawTextTrunc(bx+2, hintY, w-4, styleHelpDim, "  Press any key to close  ")
}
