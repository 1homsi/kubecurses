package ui

import (
	"fmt"

	"github.com/1homsi/kubecurses/internal/model"
)

// RenderTabBar renders the tab bar and returns it as a string.
// Right-aligns cluster/context info: ctx:NAME | N nodes | N pods
func RenderTabBar(screenW int, active model.Tab, state *model.AppState) string {
	var tabs string
	x := 0
	for i := model.Tab(0); i < model.Tab(len(model.TabNames)); i++ {
		label := fmt.Sprintf(" %d:%s ", int(i)+1, model.TabNames[i])
		style := StyleTabInactive
		if i == active {
			style = StyleTabActive
		}
		tabs += style.Render(label)
		x += len([]rune(label))
	}

	// Right-align cluster/context summary.
	ctx := state.Context
	if ctx == "" {
		ctx = "—"
	}
	info := fmt.Sprintf("ctx:%s | %d nodes | %d pods ", ctx, len(state.Nodes), len(state.Pods))
	gap := screenW - x - len([]rune(info))
	if gap > 2 {
		tabs += StyleTabInactive.Render(fmt.Sprintf("%*s", gap, ""))
		tabs += StyleTabInactive.Render(info)
	} else {
		// Fill remaining width with inactive style.
		remaining := screenW - x
		if remaining > 0 {
			tabs += StyleTabInactive.Render(fmt.Sprintf("%*s", remaining, ""))
		}
	}
	return tabs
}
