package ui

import (
	"fmt"
	"strings"

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

// RenderStatusBar renders the status bar and returns it as a string.
// When search mode is active it renders a search input bar instead.
func RenderStatusBar(screenW int, state *model.AppState) string {
	if state.SearchMode {
		prompt := fmt.Sprintf(" / %s", state.ActiveSearchQuery())
		r := []rune(prompt)
		if len(r) > screenW-1 {
			r = r[:screenW-1]
		}
		cursor := " "
		if len(r) < screenW {
			cursor = StyleSelected.Reverse(true).Render(" ")
		}
		return StyleSelected.Render(PadRight(string(r), screenW-1)) + cursor
	}

	if state.LastErr != nil {
		msg := fmt.Sprintf(" ERROR: %v", state.LastErr)
		return StyleError.Render(PadRight(Truncate(msg, screenW), screenW))
	}

	nsDisplay := state.Namespace
	if nsDisplay == "" {
		nsDisplay = "all"
	}
	if state.NamespaceFilter != "" {
		nsDisplay = state.NamespaceFilter
	}

	var parts []string
	parts = append(parts, fmt.Sprintf(" ns:%s", nsDisplay))

	if q := state.ActiveSearchQuery(); q != "" {
		parts = append(parts, fmt.Sprintf("filter:%q", q))
	}

	if state.PodsTruncated {
		parts = append(parts, fmt.Sprintf("⚠ pods capped: %d shown of %d", len(state.Pods), state.TotalPods))
	}

	left := strings.Join(parts, "  ") + contextHint(state)
	return StyleStatusBar.Render(PadRight(Truncate(left, screenW), screenW))
}
