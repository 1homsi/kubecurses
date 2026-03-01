package views

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"

	"github.com/1homsi/kubecurses/internal/model"
	"github.com/1homsi/kubecurses/internal/ui"
)

// DeploymentsView renders the deployments table.
type DeploymentsView struct {
	scrollOffset  int
	filteredDeps  []model.Deployment
}

func (v *DeploymentsView) RowCount() int { return len(v.filteredDeps) }

func (v *DeploymentsView) SelectedRef(sel int) (ns, name string) {
	if sel < 0 || sel >= len(v.filteredDeps) {
		return "", ""
	}
	return v.filteredDeps[sel].Namespace, v.filteredDeps[sel].Name
}

func (v *DeploymentsView) Draw(s *ui.Screen, r ui.Rect, state *model.AppState) {
	v.filteredDeps = filterDeployments(state)
	m := &deploymentsModel{deps: v.filteredDeps}
	sel := state.Selection[model.TabDeployments]
	v.scrollOffset = ui.DrawTable(s, r, m, sel, v.scrollOffset)
}

func filterDeployments(state *model.AppState) []model.Deployment {
	q := strings.ToLower(state.SearchQuery[model.TabDeployments])
	out := make([]model.Deployment, 0, len(state.Deployments))
	for _, d := range state.Deployments {
		if state.Namespace != "" && d.Namespace != state.Namespace {
			continue
		}
		if state.NamespaceFilter != "" && d.Namespace != state.NamespaceFilter {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(d.Name), q) &&
			!strings.Contains(strings.ToLower(d.Namespace), q) {
			continue
		}
		out = append(out, d)
	}
	return out
}

type deploymentsModel struct {
	deps []model.Deployment
}

func (m *deploymentsModel) Headers() []string {
	return []string{"NAMESPACE", "NAME", "READY", "UP-TO-DATE", "AVAILABLE", "AGE"}
}

func (m *deploymentsModel) Rows() [][]string {
	rows := make([][]string, len(m.deps))
	for i, d := range m.deps {
		rows[i] = []string{
			d.Namespace,
			d.Name,
			d.Ready,
			fmt.Sprintf("%d", d.UpToDate),
			fmt.Sprintf("%d", d.Available),
			formatDuration(d.Age),
		}
	}
	return rows
}

func (m *deploymentsModel) StyleForRow(_ int) tcell.Style {
	return ui.StyleDefault
}
