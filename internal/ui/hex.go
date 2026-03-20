package ui

// Hex tile geometry for terminal honeycomb rendering.
//
// Terminal hex tiles approximate flat-top hexagons: a broad horizontal body
// flanked by tapering shoulder rows that narrow by one column per row on each
// side. The three constants below set the taper depth and derive minimum
// supported tile dimensions.
//
// Visual cross-section (w=16, h=9, shoulder rows marked with ‹s›):
//
//	   ╱──────────╲     ← ry=0 (s) indent 3
//	  ╱────────────╲    ← ry=1 (s) indent 2
//	 ╱──────────────╲   ← ry=2 (s) indent 1
//	│────────────────│  ← ry=3 body
//	│────────────────│  ← ry=4 body
//	│────────────────│  ← ry=5 body
//	 ╲──────────────╱   ← ry=6 (s) indent 1
//	  ╲────────────╱    ← ry=7 (s) indent 2
//	   ╲──────────╱     ← ry=8 (s) indent 3

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	// HexShoulderRows is the number of tapering rows at the top and bottom of
	// every hex tile. Each shoulder row narrows the tile by one column per side,
	// so the tip indent equals HexShoulderRows.
	HexShoulderRows = 3

	// HexMinWidth ensures the narrowest row (indent = HexShoulderRows each side)
	// still has at least two cells — one for each perimeter edge character.
	HexMinWidth = 2*HexShoulderRows + 2 // = 8

	// HexMinHeight fits both shoulder sets plus at least one body row.
	HexMinHeight = 2*HexShoulderRows + 1 // = 7
)

// HexShoulderIndent returns the horizontal indent (in columns, symmetric on
// both sides) for local row ry within a tile of total height h. Body rows
// return 0; shoulder rows taper linearly from 1 (innermost) to HexShoulderRows
// (tip). Callers must ensure 0 ≤ ry < h and h ≥ HexMinHeight.
func HexShoulderIndent(ry, h int) int {
	if ry < HexShoulderRows {
		return HexShoulderRows - ry
	}
	if fromBot := h - 1 - ry; fromBot < HexShoulderRows {
		return HexShoulderRows - fromBot
	}
	return 0
}

// HexVertStep returns the vertical distance (in rows) between the top edges of
// successive tile rows in a honeycomb layout. The step is sized so that the
// bottom HexShoulderRows of row N and the top HexShoulderRows of row N+1
// share the same canvas rows, producing the interlocking hex-mesh silhouette.
func HexVertStep(tileH int) int {
	return max(tileH-HexShoulderRows, 1)
}

// HexFootprint returns an h×w boolean grid where true marks every cell that
// falls inside the hex tile silhouette (perimeter included). Dimensions are
// clamped to HexMinWidth / HexMinHeight before use.
func HexFootprint(w, h int) [][]bool {
	if w < HexMinWidth {
		w = HexMinWidth
	}
	if h < HexMinHeight {
		h = HexMinHeight
	}
	grid := make([][]bool, h)
	for ry := range grid {
		row := make([]bool, w)
		indent := HexShoulderIndent(ry, h)
		for cx := indent; cx <= w-1-indent; cx++ {
			row[cx] = true
		}
		grid[ry] = row
	}
	return grid
}

// HexPerimeterMask returns an h×w boolean grid where true marks only the
// one-cell-wide perimeter ring of the hex tile. Shoulder rows are entirely
// perimeter; body rows have only the two outermost filled cells marked.
func HexPerimeterMask(w, h int) [][]bool {
	if w < HexMinWidth {
		w = HexMinWidth
	}
	if h < HexMinHeight {
		h = HexMinHeight
	}
	perim := make([][]bool, h)
	for ry := range perim {
		row := make([]bool, w)
		indent := HexShoulderIndent(ry, h)
		left := indent
		right := w - 1 - indent
		if ry < HexShoulderRows || (h-1-ry) < HexShoulderRows {
			// Shoulder row — full span is perimeter.
			for cx := left; cx <= right; cx++ {
				row[cx] = true
			}
		} else {
			// Body row — only the two edge cells.
			if left <= right {
				row[left] = true
				row[right] = true
			}
		}
		perim[ry] = row
	}
	return perim
}

// RenderHexTile returns h styled strings each w terminal columns wide, forming
// a hex tile. bgStyle is applied to the empty cells outside the silhouette;
// fillStyle to body-interior cells; borderStyle to shoulder rows and body edge
// cells. Dimensions are clamped to HexMinWidth / HexMinHeight before use.
func RenderHexTile(w, h int, bgStyle, fillStyle, borderStyle lipgloss.Style) []string {
	if w < HexMinWidth {
		w = HexMinWidth
	}
	if h < HexMinHeight {
		h = HexMinHeight
	}
	lines := make([]string, h)
	for ry := 0; ry < h; ry++ {
		indent := HexShoulderIndent(ry, h)
		left := indent
		right := w - 1 - indent
		isShoulder := ry < HexShoulderRows || (h-1-ry) < HexShoulderRows

		var sb strings.Builder
		if left > 0 {
			sb.WriteString(bgStyle.Render(strings.Repeat(" ", left)))
		}
		if isShoulder {
			span := right - left + 1
			if span > 0 {
				sb.WriteString(borderStyle.Render(strings.Repeat(" ", span)))
			}
		} else {
			// Body row: border | fill interior | border.
			sb.WriteString(borderStyle.Render(" "))
			if inner := right - left - 1; inner > 0 {
				sb.WriteString(fillStyle.Render(strings.Repeat(" ", inner)))
			}
			sb.WriteString(borderStyle.Render(" "))
		}
		if rightPad := w - 1 - right; rightPad > 0 {
			sb.WriteString(bgStyle.Render(strings.Repeat(" ", rightPad)))
		}
		lines[ry] = sb.String()
	}
	return lines
}
