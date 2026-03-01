package ui

import (
	"fmt"

	"github.com/1homsi/kubecurses/internal/model"
)

func contextHint(state *model.AppState) string {
	switch {
	case state.HeatmapNodeDetail:
		return " Esc:back  l:Logs  e:Exec  d:Describe  hjkl:navigate"
	case state.LogsMode:
		return " Esc:close  s:autoscroll  j/k:scroll  PgUp/Dn:page"
	case state.DescribeMode:
		return " Esc:close  j/k:scroll  PgUp/Dn:page"
	case state.ClusterPickerMode, state.NamespacePickerMode:
		return " Esc:cancel  Enter:select  j/k:navigate"
	case state.ActiveTab == model.TabHeatmap:
		return " q:Quit  Tab:Next  hjkl:navigate  Enter:drill-in"
	default:
		return " q:Quit  Tab:Next  /:Search  l:Logs  e:Exec  d:Describe  n:Namespace  c:Cluster  ?:Help"
	}
}

// DrawStatusBar renders the status bar at the bottom of the screen.
// When search mode is active it renders a search input bar instead.
func DrawStatusBar(s *Screen, screenW, screenH int, state *model.AppState) {
	r := StatusBarRect(screenW, screenH)

	if state.SearchMode {
		s.FillRect(r, ' ', StyleSelected)
		prompt := fmt.Sprintf(" / %s", state.ActiveSearchQuery())
		s.DrawTextTrunc(r.X, r.Y, r.W-1, StyleSelected, prompt)
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
	if state.NamespaceFilter != "" {
		nsDisplay = state.NamespaceFilter
	}

	filterSuffix := ""
	if q := state.ActiveSearchQuery(); q != "" {
		filterSuffix = fmt.Sprintf("  filter:%q", q)
	}

	truncSuffix := ""
	if state.PodsTruncated {
		truncSuffix = fmt.Sprintf("  ⚠ pods capped: %d shown of %d", len(state.Pods), state.TotalPods)
	}

	left := fmt.Sprintf(" ns:%s%s%s%s", nsDisplay, filterSuffix, truncSuffix, contextHint(state))
	s.DrawTextTrunc(r.X, r.Y, r.W, StyleStatusBar, left)
}
