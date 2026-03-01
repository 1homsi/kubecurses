package views

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"

	"github.com/1homsi/kubecurses/internal/model"
	"github.com/1homsi/kubecurses/internal/ui"
)

var styleEventWarning = tcell.StyleDefault.
	Foreground(tcell.NewRGBColor(210, 160, 50)).
	Background(tcell.NewRGBColor(13, 14, 20))

const (
	evTypeW   = 7
	evNsW     = 12
	evObjW    = 25
	evReasonW = 18
	evCntW    = 4
	evAgeW    = 7
	evNsAt     = evTypeW + 1
	evObjAt    = evNsAt + evNsW + 1
	evReasonAt = evObjAt + evObjW + 1
	evCntAt    = evReasonAt + evReasonW + 1
	evAgeAt    = evCntAt + evCntW + 1
	evMsgAt    = evAgeAt + evAgeW + 1
)

type EventsView struct {
	rows         []model.Event
	scrollOffset int
}

func (v *EventsView) RowCount() int { return len(v.rows) }

func (v *EventsView) Draw(s *ui.Screen, r ui.Rect, state *model.AppState) {
	q := strings.ToLower(state.SearchQuery[model.TabEvents])
	v.rows = filterEvents(state.Events, q)
	sel := state.Selection[model.TabEvents]
	if sel >= len(v.rows) && len(v.rows) > 0 {
		sel = len(v.rows) - 1
		state.Selection[model.TabEvents] = sel
	}

	v.drawHeader(s, r.X, r.Y, r.W)
	content := ui.Rect{X: r.X, Y: r.Y + 1, W: r.W, H: r.H - 1}

	if len(v.rows) > 0 {
		if sel < v.scrollOffset {
			v.scrollOffset = sel
		}
		if sel >= v.scrollOffset+content.H {
			v.scrollOffset = sel - content.H + 1
		}
		if v.scrollOffset < 0 {
			v.scrollOffset = 0
		}
	}

	for i := 0; i < content.H; i++ {
		rowIdx := v.scrollOffset + i
		y := content.Y + i
		if rowIdx >= len(v.rows) {
			s.FillRect(ui.Rect{X: content.X, Y: y, W: content.W, H: 1}, ' ', ui.StyleDefault)
			continue
		}
		v.drawEventRow(s, content.X, y, content.W, v.rows[rowIdx], rowIdx == sel)
	}
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

func (v *EventsView) drawHeader(s *ui.Screen, x, y, w int) {
	s.FillRect(ui.Rect{X: x, Y: y, W: w, H: 1}, ' ', ui.StyleHeader)
	s.DrawText(x, y, ui.StyleHeader, fmt.Sprintf("%-*s", evTypeW, "TYPE"))
	s.DrawText(x+evNsAt, y, ui.StyleHeader, fmt.Sprintf("%-*s", evNsW, "NAMESPACE"))
	s.DrawText(x+evObjAt, y, ui.StyleHeader, fmt.Sprintf("%-*s", evObjW, "OBJECT"))
	s.DrawText(x+evReasonAt, y, ui.StyleHeader, fmt.Sprintf("%-*s", evReasonW, "REASON"))
	s.DrawText(x+evCntAt, y, ui.StyleHeader, fmt.Sprintf("%-*s", evCntW, "CNT"))
	s.DrawText(x+evAgeAt, y, ui.StyleHeader, fmt.Sprintf("%-*s", evAgeW, "AGE"))
	if evMsgAt < w {
		s.DrawText(x+evMsgAt, y, ui.StyleHeader, "MESSAGE")
	}
}

func (v *EventsView) drawEventRow(s *ui.Screen, x, y, w int, e model.Event, selected bool) {
	style := ui.StyleDefault
	if e.Type == "Warning" {
		style = styleEventWarning
	}
	if selected {
		style = ui.StyleSelected
	}
	s.FillRect(ui.Rect{X: x, Y: y, W: w, H: 1}, ' ', style)
	s.DrawTextTrunc(x, y, evTypeW, style, e.Type)
	s.DrawTextTrunc(x+evNsAt, y, evNsW, style, e.Namespace)
	s.DrawTextTrunc(x+evObjAt, y, evObjW, style, e.Kind+"/"+e.Name)
	s.DrawTextTrunc(x+evReasonAt, y, evReasonW, style, e.Reason)
	s.DrawText(x+evCntAt, y, style, fmt.Sprintf("%-*d", evCntW, e.Count))
	s.DrawTextTrunc(x+evAgeAt, y, evAgeW, style, formatDuration(e.Age))
	if evMsgAt < w {
		s.DrawTextTrunc(x+evMsgAt, y, w-evMsgAt, style, e.Message)
	}
}
