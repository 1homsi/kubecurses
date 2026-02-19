package views

import (
	"strings"

	"github.com/gdamore/tcell/v2"

	"github.com/1homsi/kubecurses/internal/model"
	"github.com/1homsi/kubecurses/internal/ui"
)

// NodesView renders a flat nodes table.
// Nodes are primarily shown inside NodeOverviewView; this view is kept as
// a fallback / future standalone tab.
type NodesView struct {
	scrollOffset int
}

func (v *NodesView) Draw(s *ui.Screen, r ui.Rect, state *model.AppState) {
	m := &nodesModel{nodes: filterNodes(state)}
	// NodesView has no dedicated tab index currently; use selection 0.
	v.scrollOffset = ui.DrawTable(s, r, m, 0, v.scrollOffset)
}

func filterNodes(state *model.AppState) []model.Node {
	q := strings.ToLower(state.SearchQuery)
	if q == "" {
		return state.Nodes
	}
	out := make([]model.Node, 0, len(state.Nodes))
	for _, n := range state.Nodes {
		if strings.Contains(strings.ToLower(n.Name), q) {
			out = append(out, n)
		}
	}
	return out
}

type nodesModel struct {
	nodes []model.Node
}

func (m *nodesModel) Headers() []string {
	return []string{"NAME", "STATUS", "ROLES", "AGE", "VERSION"}
}

func (m *nodesModel) Rows() [][]string {
	rows := make([][]string, len(m.nodes))
	for i, n := range m.nodes {
		rows[i] = []string{
			n.Name,
			n.Status,
			n.Roles,
			formatDuration(n.Age),
			n.Version,
		}
	}
	return rows
}

func (m *nodesModel) StyleForRow(row int) tcell.Style {
	if row >= len(m.nodes) {
		return ui.StyleDefault
	}
	if m.nodes[row].Status == "Ready" {
		return ui.StyleNodeReady
	}
	return ui.StyleNodeNotReady
}
