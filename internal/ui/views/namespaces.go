package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

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

func (v *NamespacesView) Render(width, height int, state *model.AppState) string {
	v.filteredNss = filterNamespaces(state)
	m := &namespacesModel{nss: v.filteredNss}
	sel := state.Selection[model.TabNamespaces]
	var rendered string
	rendered, v.scrollOffset = ui.RenderTable(width, height, m, sel, v.scrollOffset)
	return rendered
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

func (m *namespacesModel) StyleForRow(_ int) lipgloss.Style {
	return ui.StyleDefault
}
