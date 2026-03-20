package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/1homsi/kubecurses/internal/model"
	"github.com/1homsi/kubecurses/internal/ui"
)

var styleEventWarning = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#D2A032")).
	Background(lipgloss.Color("#101010"))

const (
	evTypeW   = 7
	evNsW     = 12
	evObjW    = 25
	evReasonW = 18
	evCntW    = 4
	evAgeW    = 7
	evNsAt    = evTypeW + 1
	evObjAt   = evNsAt + evNsW + 1
	evReasonAt = evObjAt + evObjW + 1
	evCntAt   = evReasonAt + evReasonW + 1
	evAgeAt   = evCntAt + evCntW + 1
	evMsgAt   = evAgeAt + evAgeW + 1
)

type EventsView struct {
	rows         []model.Event
	scrollOffset int
}

func (v *EventsView) RowCount() int { return len(v.rows) }

func (v *EventsView) Render(width, height int, state *model.AppState) string {
	q := strings.ToLower(state.SearchQuery[model.TabEvents])
	v.rows = filterEvents(state.Events, q)
	sel := state.Selection[model.TabEvents]
	if sel >= len(v.rows) && len(v.rows) > 0 {
		sel = len(v.rows) - 1
		state.Selection[model.TabEvents] = sel
	}

	var lines []string
	lines = append(lines, v.renderHeader(width))
	contentH := height - 1

	if len(v.rows) > 0 {
		if sel < v.scrollOffset {
			v.scrollOffset = sel
		}
		if sel >= v.scrollOffset+contentH {
			v.scrollOffset = sel - contentH + 1
		}
		if v.scrollOffset < 0 {
			v.scrollOffset = 0
		}
	}

	for i := 0; i < contentH; i++ {
		rowIdx := v.scrollOffset + i
		if rowIdx >= len(v.rows) {
			lines = append(lines, ui.FillWidth(width, ui.StyleDefault))
			continue
		}
		lines = append(lines, v.renderEventRow(width, v.rows[rowIdx], rowIdx == sel))
	}

	return strings.Join(lines, "\n")
}

func filterEvents(events []model.Event, q string) []model.Event {
	if q == "" {
		return events
	}
	out := make([]model.Event, 0, len(events))
	for _, e := range events {
		if strings.Contains(strings.ToLower(e.Namespace), q) ||
			strings.Contains(strings.ToLower(e.Kind), q) ||
			strings.Contains(strings.ToLower(e.Name), q) ||
			strings.Contains(strings.ToLower(e.Reason), q) ||
			strings.Contains(strings.ToLower(e.Message), q) {
			out = append(out, e)
		}
	}
	return out
}

func (v *EventsView) renderHeader(w int) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%-*s", evTypeW, "TYPE"))
	b.WriteString(fmt.Sprintf(" %-*s", evNsW, "NAMESPACE"))
	b.WriteString(fmt.Sprintf(" %-*s", evObjW, "OBJECT"))
	b.WriteString(fmt.Sprintf(" %-*s", evReasonW, "REASON"))
	b.WriteString(fmt.Sprintf(" %-*s", evCntW, "CNT"))
	b.WriteString(fmt.Sprintf(" %-*s", evAgeW, "AGE"))
	if evMsgAt < w {
		b.WriteString(" MESSAGE")
	}
	return ui.StyleHeader.Render(ui.PadRight(b.String(), w))
}

func (v *EventsView) renderEventRow(w int, e model.Event, selected bool) string {
	style := ui.StyleDefault
	if e.Type == "Warning" {
		style = styleEventWarning
	}
	if selected {
		style = ui.StyleSelected
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("%-*s", evTypeW, truncate(e.Type, evTypeW)))
	b.WriteString(fmt.Sprintf(" %-*s", evNsW, truncate(e.Namespace, evNsW)))
	b.WriteString(fmt.Sprintf(" %-*s", evObjW, truncate(e.Kind+"/"+e.Name, evObjW)))
	b.WriteString(fmt.Sprintf(" %-*s", evReasonW, truncate(e.Reason, evReasonW)))
	b.WriteString(fmt.Sprintf(" %-*d", evCntW, e.Count))
	b.WriteString(fmt.Sprintf(" %-*s", evAgeW, truncate(formatDuration(e.Age), evAgeW)))
	if evMsgAt < w {
		msgW := w - evMsgAt
		if msgW > 0 {
			b.WriteString(fmt.Sprintf(" %-*s", msgW, truncate(e.Message, msgW)))
		}
	}

	return style.Render(ui.PadRight(b.String(), w))
}
