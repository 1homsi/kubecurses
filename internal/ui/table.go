package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// TableModel is implemented by each resource view to supply table data.
type TableModel interface {
	// Headers returns the column header strings.
	Headers() []string
	// Rows returns all data rows. Each inner slice must match len(Headers()).
	Rows() [][]string
	// StyleForRow returns the lipgloss style to use for the given data row.
	// row is the 0-based index into Rows().
	StyleForRow(row int) lipgloss.Style
}

// RenderTable renders a scrollable table and returns it as a string.
// selected is the 0-based index of the highlighted row.
// scrollOffset is the first visible data row (for vertical scrolling).
// Returns (rendered string, new scrollOffset).
func RenderTable(width, height int, model TableModel, selected, scrollOffset int) (string, int) {
	if height <= 0 || width <= 0 {
		return "", scrollOffset
	}

	headers := model.Headers()
	rows := model.Rows()

	// Reserve first row for headers.
	visibleRows := height - 1
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

	// Calculate column widths (equal distribution).
	colCount := len(headers)
	if colCount == 0 {
		return "", scrollOffset
	}
	colW := width / colCount

	var lines []string

	// Header row.
	var hdrParts []string
	for _, h := range headers {
		hdrParts = append(hdrParts, fmt.Sprintf("%-*s", colW, Truncate(h, colW)))
	}
	hdrText := strings.Join(hdrParts, "")
	lines = append(lines, StyleHeader.Render(PadRight(hdrText, width)))

	// Data rows.
	for i := 0; i < visibleRows; i++ {
		rowIdx := scrollOffset + i
		if rowIdx >= len(rows) {
			lines = append(lines, FillWidth(width, StyleDefault))
			continue
		}

		style := model.StyleForRow(rowIdx)
		if rowIdx == selected {
			style = StyleSelected
		}

		var parts []string
		for col, cell := range rows[rowIdx] {
			if col >= colCount {
				break
			}
			parts = append(parts, fmt.Sprintf("%-*s", colW, Truncate(cell, colW)))
		}
		rowText := strings.Join(parts, "")
		lines = append(lines, style.Render(PadRight(rowText, width)))
	}

	return strings.Join(lines, "\n"), scrollOffset
}
