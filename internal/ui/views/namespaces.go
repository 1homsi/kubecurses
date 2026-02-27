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
}

func (v *NamespacesView) Draw(s *ui.Screen, r ui.Rect, state *model.AppState) {
	m := &namespacesModel{nss: filterNamespaces(state)}
	sel := state.Selection[model.TabNamespaces]
	v.scrollOffset = ui.DrawTable(s, r, m, sel, v.scrollOffset)
}

// SelectedNamespace returns the name of the filtered namespace at index sel.
// Returns "" when sel is out of range.
func SelectedNamespace(sel int, state *model.AppState) string {
	nss := filterNamespaces(state)
	if sel < 0 || sel >= len(nss) {
		return ""
	}
	return nss[sel].Name
}

func filterNamespaces(state *model.AppState) []model.Namespace {
	q := strings.ToLower(state.SearchQuery)
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
