package ui

import "github.com/gdamore/tcell/v2"

// TableModel is implemented by each resource view to supply table data.
type TableModel interface {
	// Headers returns the column header strings.
	Headers() []string
	// Rows returns all data rows. Each inner slice must match len(Headers()).
	Rows() [][]string
	// StyleForRow returns the tcell style to use for the given data row.
	// row is the 0-based index into Rows().
	StyleForRow(row int) tcell.Style
}

// DrawTable renders a scrollable table inside rect r.
// selected is the 0-based index of the highlighted row.
// scrollOffset is the first visible data row (for vertical scrolling).
// Returns the new scrollOffset adjusted so the selected row is visible.
func DrawTable(s *Screen, r Rect, model TableModel, selected, scrollOffset int) int {
	if r.H <= 0 || r.W <= 0 {
		return scrollOffset
	}

	headers := model.Headers()
	rows := model.Rows()

	// Reserve first row for headers.
	visibleRows := r.H - 1
	if visibleRows < 1 {
		visibleRows = 1
	}

	// Adjust scrollOffset to keep selected in view.
	if selected < scrollOffset {
		scrollOffset = selected
	}
	if selected >= scrollOffset+visibleRows {
		scrollOffset = selected - visibleRows + 1
	}
	if scrollOffset < 0 {
		scrollOffset = 0
	}

	// Calculate column widths (equal distribution for simplicity).
	colCount := len(headers)
	if colCount == 0 {
		return scrollOffset
	}
	colW := r.W / colCount

	// Draw header row.
	s.FillRect(Rect{X: r.X, Y: r.Y, W: r.W, H: 1}, ' ', StyleHeader)
	for col, h := range headers {
		s.DrawTextTrunc(r.X+col*colW, r.Y, colW, StyleHeader, h)
	}

	// Draw data rows.
	for i := 0; i < visibleRows; i++ {
		rowIdx := scrollOffset + i
		screenY := r.Y + 1 + i
		if screenY >= r.Y+r.H {
			break
		}

		if rowIdx >= len(rows) {
			// Clear trailing empty rows.
			s.FillRect(Rect{X: r.X, Y: screenY, W: r.W, H: 1}, ' ', StyleDefault)
			continue
		}

		style := model.StyleForRow(rowIdx)
		if rowIdx == selected {
			style = StyleSelected
		}
		s.FillRect(Rect{X: r.X, Y: screenY, W: r.W, H: 1}, ' ', style)
		for col, cell := range rows[rowIdx] {
			if col >= colCount {
				break
			}
			s.DrawTextTrunc(r.X+col*colW, screenY, colW, style, cell)
		}
	}

	return scrollOffset
}
