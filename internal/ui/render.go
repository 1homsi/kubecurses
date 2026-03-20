package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// PadRight pads s with spaces to width. If s is already wider, it is returned as-is.
func PadRight(s string, width int) string {
	n := lipgloss.Width(s)
	if n >= width {
		return s
	}
	return s + strings.Repeat(" ", width-n)
}

// Truncate truncates s to maxW visible columns, appending "…" if truncated.
func Truncate(s string, maxW int) string {
	if maxW <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= maxW {
		return s
	}
	if maxW <= 1 {
		return "…"
	}
	return string(r[:maxW-1]) + "…"
}

// Segment is a styled text fragment within a row.
type Segment struct {
	Text  string
	Style lipgloss.Style
}

// RenderRow composes segments into a single row string of exactly width columns.
// Excess is truncated; shortfall is padded with the last segment's style (or StyleDefault).
func RenderRow(width int, segments []Segment) string {
	if width <= 0 {
		return ""
	}
	var b strings.Builder
	remaining := width
	var lastStyle lipgloss.Style
	for _, seg := range segments {
		if remaining <= 0 {
			break
		}
		lastStyle = seg.Style
		r := []rune(seg.Text)
		if len(r) > remaining {
			r = r[:remaining]
		}
		b.WriteString(seg.Style.Render(string(r)))
		remaining -= len(r)
	}
	if remaining > 0 {
		b.WriteString(lastStyle.Render(strings.Repeat(" ", remaining)))
	}
	return b.String()
}

// FillWidth returns a string of width spaces rendered with the given style.
func FillWidth(width int, style lipgloss.Style) string {
	if width <= 0 {
		return ""
	}
	return style.Render(strings.Repeat(" ", width))
}

// OverlayCenter places an overlay string (multi-line) centered on a base string
// (multi-line). Both are expected to be width columns wide, height rows tall.
func OverlayCenter(base string, width, height int, overlay string) string {
	baseLines := strings.Split(base, "\n")
	overlayLines := strings.Split(overlay, "\n")

	oW := 0
	for _, line := range overlayLines {
		if w := lipgloss.Width(line); w > oW {
			oW = w
		}
	}
	oH := len(overlayLines)

	startY := (height - oH) / 2
	startX := (width - oW) / 2
	if startY < 0 {
		startY = 0
	}
	if startX < 0 {
		startX = 0
	}

	// Ensure baseLines has enough rows.
	for len(baseLines) < height {
		baseLines = append(baseLines, strings.Repeat(" ", width))
	}

	for i, oLine := range overlayLines {
		y := startY + i
		if y >= len(baseLines) {
			break
		}
		bRunes := []rune(baseLines[y])
		oRunes := []rune(oLine)

		// Build new line: base prefix + overlay + base suffix
		var line strings.Builder
		// Characters before the overlay
		if startX > 0 && startX <= len(bRunes) {
			line.WriteString(string(bRunes[:startX]))
		} else if startX > 0 {
			line.WriteString(string(bRunes))
			line.WriteString(strings.Repeat(" ", startX-len(bRunes)))
		}
		line.WriteString(string(oRunes))
		endX := startX + len(oRunes)
		if endX < len(bRunes) {
			line.WriteString(string(bRunes[endX:]))
		}
		baseLines[y] = line.String()
	}

	return strings.Join(baseLines[:height], "\n")
}
