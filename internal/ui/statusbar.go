package ui

import (
	"fmt"

	"github.com/1homsi/kubecurses/internal/model"
)

const keybindingHint = " q:Quit  Tab:Next  /:Search  r:Refresh  j/k:↑↓  PgDn/PgUp:Scroll"

// DrawStatusBar renders the status bar at the bottom of the screen.
// When search mode is active it renders a search input bar instead.
func DrawStatusBar(s *Screen, screenW, screenH int, state *model.AppState) {
	r := StatusBarRect(screenW, screenH)

	if state.SearchMode {
		s.FillRect(r, ' ', StyleSelected)
		prompt := fmt.Sprintf(" / %s", state.SearchQuery)
		s.DrawTextTrunc(r.X, r.Y, r.W-1, StyleSelected, prompt)
		// Blinking cursor simulation: draw a block at the end of the query.
		cursorX := r.X + len([]rune(prompt))
		if cursorX < r.X+r.W {
			s.DrawText(cursorX, r.Y, StyleSelected.Reverse(true), " ")
		}
		return
	}

	s.FillRect(r, ' ', StyleStatusBar)

	if state.LastErr != nil {
		msg := fmt.Sprintf(" ERROR: %v", state.LastErr)
		s.DrawTextTrunc(r.X, r.Y, r.W, StyleError, msg)
		return
	}

	nsDisplay := state.Namespace
	if nsDisplay == "" {
		nsDisplay = "all"
	}
	filterSuffix := ""
	if state.SearchQuery != "" {
		filterSuffix = fmt.Sprintf("  filter:%q", state.SearchQuery)
	}
	left := fmt.Sprintf(" ns:%s%s%s", nsDisplay, filterSuffix, keybindingHint)
	s.DrawTextTrunc(r.X, r.Y, r.W, StyleStatusBar, left)
}
