package ui

import (
	"testing"
)

func TestHexShoulderIndent_TipAndBody(t *testing.T) {
	h := HexMinHeight // = 7 (3 top + 1 body + 3 bottom)
	tests := []struct {
		ry   int
		want int
	}{
		{0, HexShoulderRows},     // top tip — maximum indent
		{1, HexShoulderRows - 1}, // second top shoulder
		{2, HexShoulderRows - 2}, // last top shoulder (indent = 1)
		{3, 0},                   // sole body row
		{4, 1},                   // first bottom shoulder
		{5, 2},                   // second bottom shoulder
		{6, HexShoulderRows},     // bottom tip — maximum indent
	}
	for _, tt := range tests {
		got := HexShoulderIndent(tt.ry, h)
		if got != tt.want {
			t.Errorf("HexShoulderIndent(%d, %d) = %d, want %d", tt.ry, h, got, tt.want)
		}
	}
}

func TestHexShoulderIndent_BodyRowsZero(t *testing.T) {
	// All body rows between the two shoulder sets must have indent 0.
	h := 11 // 3 top + 5 body + 3 bottom
	for ry := HexShoulderRows; ry < h-HexShoulderRows; ry++ {
		if got := HexShoulderIndent(ry, h); got != 0 {
			t.Errorf("HexShoulderIndent(%d, %d) = %d, want 0 (body row)", ry, h, got)
		}
	}
}

func TestHexShoulderIndent_Symmetry(t *testing.T) {
	for h := HexMinHeight; h <= 15; h++ {
		for ry := 0; ry < h; ry++ {
			mirror := h - 1 - ry
			got := HexShoulderIndent(ry, h)
			gotM := HexShoulderIndent(mirror, h)
			if got != gotM {
				t.Errorf("h=%d: HexShoulderIndent(%d)=%d != HexShoulderIndent(%d)=%d (not symmetric)",
					h, ry, got, mirror, gotM)
			}
		}
	}
}

func TestHexVertStep_LessThanHeight(t *testing.T) {
	for h := HexMinHeight; h <= 25; h++ {
		step := HexVertStep(h)
		if step >= h {
			t.Errorf("HexVertStep(%d) = %d, want < %d (must create overlap)", h, step, h)
		}
		if step < 1 {
			t.Errorf("HexVertStep(%d) = %d, want >= 1", h, step)
		}
	}
}

func TestHexVertStep_OverlapEqualsShoulderRows(t *testing.T) {
	// The vertical overlap between successive rows must equal HexShoulderRows.
	for h := HexMinHeight; h <= 20; h++ {
		step := HexVertStep(h)
		overlap := h - step
		if overlap != HexShoulderRows {
			t.Errorf("HexVertStep(%d): overlap = %d, want HexShoulderRows = %d",
				h, overlap, HexShoulderRows)
		}
	}
}

func TestHexFootprint_Dimensions(t *testing.T) {
	w, h := 16, 9
	fp := HexFootprint(w, h)
	if len(fp) != h {
		t.Fatalf("HexFootprint rows = %d, want %d", len(fp), h)
	}
	for i, row := range fp {
		if len(row) != w {
			t.Errorf("HexFootprint row %d len = %d, want %d", i, len(row), w)
		}
	}
}

func TestHexFootprint_HorizontalSymmetry(t *testing.T) {
	w, h := 16, 9
	fp := HexFootprint(w, h)
	for ry, row := range fp {
		for cx := range row {
			if row[cx] != row[w-1-cx] {
				t.Errorf("HexFootprint horizontal asymmetry at (ry=%d, cx=%d)", ry, cx)
			}
		}
	}
}

func TestHexFootprint_VerticalSymmetry(t *testing.T) {
	w, h := 16, 9
	fp := HexFootprint(w, h)
	for ry := 0; ry < h/2; ry++ {
		mirror := fp[h-1-ry]
		for cx := range fp[ry] {
			if fp[ry][cx] != mirror[cx] {
				t.Errorf("HexFootprint vertical asymmetry at (ry=%d, cx=%d)", ry, cx)
			}
		}
	}
}

func TestHexFootprint_MinSizeNonEmpty(t *testing.T) {
	// Every row at minimum dimensions must contain at least one filled cell.
	fp := HexFootprint(HexMinWidth, HexMinHeight)
	for ry, row := range fp {
		hasCell := false
		for _, v := range row {
			if v {
				hasCell = true
				break
			}
		}
		if !hasCell {
			t.Errorf("HexFootprint at min size: row %d is entirely empty", ry)
		}
	}
}

func TestHexFootprint_MonotoneWidthToCenter(t *testing.T) {
	// Each successive row from the top to the middle must be at least as wide
	// as the previous row (strictly non-decreasing span width).
	w, h := 20, 11
	fp := HexFootprint(w, h)
	prevSpan := 0
	for ry := 0; ry <= h/2; ry++ {
		span := 0
		for _, v := range fp[ry] {
			if v {
				span++
			}
		}
		if span < prevSpan {
			t.Errorf("HexFootprint non-monotone at ry=%d: span %d < prev %d", ry, span, prevSpan)
		}
		prevSpan = span
	}
}

func TestHexPerimeterMask_ContainedInFootprint(t *testing.T) {
	w, h := 16, 9
	fp := HexFootprint(w, h)
	pm := HexPerimeterMask(w, h)
	for ry := range pm {
		for cx := range pm[ry] {
			if pm[ry][cx] && !fp[ry][cx] {
				t.Errorf("HexPerimeterMask cell (%d,%d) is perimeter but outside footprint", ry, cx)
			}
		}
	}
}

func TestHexPerimeterMask_ShoulderRowsFullPerimeter(t *testing.T) {
	// In shoulder rows, every footprint cell must be a perimeter cell.
	w, h := 16, 9
	fp := HexFootprint(w, h)
	pm := HexPerimeterMask(w, h)
	for ry := 0; ry < HexShoulderRows; ry++ {
		for cx := range fp[ry] {
			if fp[ry][cx] != pm[ry][cx] {
				t.Errorf("Top shoulder ry=%d cx=%d: footprint=%v perim=%v (must match)",
					ry, cx, fp[ry][cx], pm[ry][cx])
			}
		}
	}
	for ry := h - HexShoulderRows; ry < h; ry++ {
		for cx := range fp[ry] {
			if fp[ry][cx] != pm[ry][cx] {
				t.Errorf("Bottom shoulder ry=%d cx=%d: footprint=%v perim=%v (must match)",
					ry, cx, fp[ry][cx], pm[ry][cx])
			}
		}
	}
}

func TestHexPerimeterMask_BodyRowsExactlyTwoEdges(t *testing.T) {
	// Each body row must have exactly two perimeter cells (left and right edges).
	w, h := 16, 9
	pm := HexPerimeterMask(w, h)
	for ry := HexShoulderRows; ry < h-HexShoulderRows; ry++ {
		count := 0
		for _, v := range pm[ry] {
			if v {
				count++
			}
		}
		if count != 2 {
			t.Errorf("Body row %d has %d perimeter cells, want 2", ry, count)
		}
	}
}

func TestHexPerimeterMask_PerimeterContinuous(t *testing.T) {
	// The leftmost and rightmost perimeter cells in each row must form a
	// connected ring: consecutive rows' left/right edges may differ by at most
	// one column (the indent step).
	w, h := 20, 11
	pm := HexPerimeterMask(w, h)

	leftEdge := func(ry int) int {
		for cx, v := range pm[ry] {
			if v {
				return cx
			}
		}
		return -1
	}
	rightEdge := func(ry int) int {
		for cx := w - 1; cx >= 0; cx-- {
			if pm[ry][cx] {
				return cx
			}
		}
		return -1
	}

	for ry := 1; ry < h; ry++ {
		dl := leftEdge(ry) - leftEdge(ry-1)
		dr := rightEdge(ry) - rightEdge(ry-1)
		if dl < -1 || dl > 1 {
			t.Errorf("Left edge jump between rows %d→%d: delta=%d (want ≤1)", ry-1, ry, dl)
		}
		if dr < -1 || dr > 1 {
			t.Errorf("Right edge jump between rows %d→%d: delta=%d (want ≤1)", ry-1, ry, dr)
		}
	}
}

func TestRenderHexTile_LineCount(t *testing.T) {
	h := 9
	lines := RenderHexTile(16, h, StyleDefault, StyleNodeHeader, StyleHeatmapNodeSel)
	if len(lines) != h {
		t.Errorf("RenderHexTile returned %d lines, want %d", len(lines), h)
	}
}

func TestRenderHexTile_MinSizeClamped(t *testing.T) {
	// Passing sub-minimum dims must clamp without panic.
	lines := RenderHexTile(2, 2, StyleDefault, StyleNodeHeader, StyleHeatmapNodeSel)
	if len(lines) != HexMinHeight {
		t.Errorf("RenderHexTile clamped to %d lines, want HexMinHeight=%d", len(lines), HexMinHeight)
	}
}
