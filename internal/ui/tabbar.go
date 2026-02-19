package ui

import (
	"fmt"

	"github.com/1homsi/kubecurses/internal/model"
)

// DrawTabBar renders the tab bar across the top of the screen.
// Right-aligns cluster/context info: ctx:NAME | N nodes | N pods
func DrawTabBar(s *Screen, screenW int, active model.Tab, state *model.AppState) {
	r := TabBarRect(screenW)
	s.FillRect(r, ' ', StyleTabInactive)

	x := 0
	for i := model.Tab(0); i < model.Tab(len(model.TabNames)); i++ {
		label := fmt.Sprintf(" %d:%s ", int(i)+1, model.TabNames[i])
		style := StyleTabInactive
		if i == active {
			style = StyleTabActive
		}
		s.DrawText(x, r.Y, style, label)
		x += len([]rune(label))
	}

	// Right-align cluster/context summary.
	ctx := state.Context
	if ctx == "" {
		ctx = "—"
	}
	info := fmt.Sprintf("ctx:%s | %d nodes | %d pods ", ctx, len(state.Nodes), len(state.Pods))
	infoX := screenW - len([]rune(info))
	if infoX > x+2 {
		s.DrawText(infoX, r.Y, StyleTabInactive, info)
	}
}
