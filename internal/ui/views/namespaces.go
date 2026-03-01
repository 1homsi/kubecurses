package views

import (
	"strings"

	"github.com/gdamore/tcell/v2"

	"github.com/1homsi/kubecurses/internal/model"
	"github.com/1homsi/kubecurses/internal/ui"
)

// NamespacesView renders the namespaces table.
type NamespacesView struct {
	scrollOffset int
	filteredNss  []model.Namespace
}

func (v *NamespacesView) RowCount() int { return len(v.filteredNss) }

func (v *NamespacesView) SelectedRef(sel int) string {
	if sel < 0 || sel >= len(v.filteredNss) {
		return ""
	}
	return v.filteredNss[sel].Name
}

func (v *NamespacesView) Draw(s *ui.Screen, r ui.Rect, state *model.AppState) {
	v.filteredNss = filterNamespaces(state)
	m := &namespacesModel{nss: v.filteredNss}
	sel := state.Selection[model.TabNamespaces]
	v.scrollOffset = ui.DrawTable(s, r, m, sel, v.scrollOffset)
}

func filterNamespaces(state *model.AppState) []model.Namespace {
	q := strings.ToLower(state.SearchQuery[model.TabNamespaces])
	out := make([]model.Namespace, 0, len(state.Namespaces))
	for _, ns := range state.Namespaces {
		if q != "" && !strings.Contains(strings.ToLower(ns.Name), q) {
			continue
		}
		out = append(out, ns)
	}
	return out
}

type namespacesModel struct {
	nss []model.Namespace
}

func (m *namespacesModel) Headers() []string {
	return []string{"NAME", "STATUS", "AGE"}
}

func (m *namespacesModel) Rows() [][]string {
	rows := make([][]string, len(m.nss))
	for i, ns := range m.nss {
		rows[i] = []string{
			ns.Name,
			ns.Status,
			formatDuration(ns.Age),
		}
	}
	return rows
}

func (m *namespacesModel) StyleForRow(_ int) tcell.Style {
	return ui.StyleDefault
}
