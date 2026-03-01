package views

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"

	"github.com/1homsi/kubecurses/internal/model"
	"github.com/1homsi/kubecurses/internal/ui"
)

// PodsView renders a flat, scrollable list of all pods.
type PodsView struct {
	scrollOffset int
}

func (v *PodsView) Draw(s *ui.Screen, r ui.Rect, state *model.AppState) {
	m := &podsModel{pods: filterPods(state)}
	sel := state.Selection[model.TabPods]
	v.scrollOffset = ui.DrawTable(s, r, m, sel, v.scrollOffset)
}

func filterPods(state *model.AppState) []model.Pod {
	q := strings.ToLower(state.SearchQuery[model.TabPods])
	out := make([]model.Pod, 0, len(state.Pods))
	for _, p := range state.Pods {
		if state.Namespace != "" && p.Namespace != state.Namespace {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(p.Name), q) &&
			!strings.Contains(strings.ToLower(p.Namespace), q) &&
			!strings.Contains(strings.ToLower(p.Status), q) {
			continue
		}
		out = append(out, p)
	}
	return out
}

type podsModel struct {
	pods []model.Pod
}

func (m *podsModel) Headers() []string {
	return []string{"NAMESPACE", "NAME", "READY", "STATUS", "RESTARTS", "AGE", "NODE"}
}

func (m *podsModel) Rows() [][]string {
	rows := make([][]string, len(m.pods))
	for i, p := range m.pods {
		rows[i] = []string{
			p.Namespace,
			p.Name,
			p.Ready,
			p.Status,
			fmt.Sprintf("%d", p.Restarts),
			formatDuration(p.Age),
			p.Node,
		}
	}
	return rows
}

func (m *podsModel) StyleForRow(row int) tcell.Style {
	if row >= len(m.pods) {
		return ui.StyleDefault
	}
	return podBaseStyle(m.pods[row].Status)
}
