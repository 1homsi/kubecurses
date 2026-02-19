package ui

import (
	"fmt"

	"github.com/1homsi/kubecurses/internal/model"
)

// DrawTabBar renders the tab bar across the top of the screen.
func DrawTabBar(s *Screen, screenW int, active model.Tab) {
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
}
