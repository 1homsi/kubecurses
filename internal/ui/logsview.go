package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"

	"github.com/1homsi/kubecurses/internal/model"
)

var (
	styleLogsTitleBg = tcell.NewRGBColor(12, 18, 32)
	styleLogsTitle   = tcell.StyleDefault.Background(styleLogsTitleBg).Foreground(tcell.NewRGBColor(130, 190, 255)).Bold(true)
	styleLogsHint    = tcell.StyleDefault.Background(tcell.NewRGBColor(16, 18, 28)).Foreground(tcell.NewRGBColor(100, 105, 130))
	styleLogsLine    = tcell.StyleDefault.Background(tcell.NewRGBColor(13, 14, 20)).Foreground(tcell.NewRGBColor(195, 200, 218))
	styleLogsMarker  = tcell.StyleDefault.Background(tcell.NewRGBColor(13, 14, 20)).Foreground(tcell.NewRGBColor(70, 120, 210))
	styleLogsAutoOn  = tcell.StyleDefault.Background(tcell.NewRGBColor(16, 18, 28)).Foreground(tcell.NewRGBColor(80, 200, 120)).Bold(true)
	styleLogsAutoOff = tcell.StyleDefault.Background(tcell.NewRGBColor(16, 18, 28)).Foreground(tcell.NewRGBColor(100, 105, 130))
	styleLogsBorder  = tcell.StyleDefault.Background(tcell.NewRGBColor(13, 14, 20)).Foreground(tcell.NewRGBColor(50, 80, 150))
)

// WrapLines splits each element of lines into segments of at most maxW runes,
// returning a flat slice of display rows. Lines that fit within maxW are passed
// through unchanged. If maxW <= 0 the input slice is returned as-is.
func WrapLines(lines []string, maxW int) []string {
	if maxW <= 0 {
		return lines
	}
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		runes := []rune(line)
		if len(runes) <= maxW {
			result = append(result, line)
			continue
		}
		for len(runes) > maxW {
			result = append(result, string(runes[:maxW]))
			runes = runes[maxW:]
		}
		if len(runes) > 0 {
			result = append(result, string(runes))
		}
	}
	return result
}

// DrawLogsView renders a bordered log streaming box within the content rect r.
// Layout inside the box:
//
//	row 0      : top border with centred title
//	row 1      : status strip (autoscroll + hint + line count)
//	rows 2…H-2 : log content (hard-wrapped to fit the box width)
//	row H-1    : bottom border
func DrawLogsView(s *Screen, r Rect, state *model.AppState) {
	if r.W < 4 || r.H < 4 {
		return
	}

	// ── fill interior background ───────────────────────────────────────────
	for i := 1; i < r.H-1; i++ {
		s.FillRect(Rect{X: r.X + 1, Y: r.Y + i, W: r.W - 2, H: 1}, ' ', styleLogsLine)
	}

	// ── border (drawn after fill so corners overwrite interior fill) ───────
	drawBorderOnly(s, r.X, r.Y, r.W, r.H, styleLogsBorder)

	// ── title centred on top border row ───────────────────────────────────
	podLabel := state.LogsNamespace + "/" + state.LogsPod
	title := " Logs — " + podLabel
	if state.LogsContainer != "" {
		title += " [" + state.LogsContainer + "]"
	}
	title += " "
	titleX := r.X + (r.W-len([]rune(title)))/2
	if titleX < r.X+1 {
		titleX = r.X + 1
	}
	s.DrawTextTrunc(titleX, r.Y, r.W-2, styleLogsTitle, title)

	// ── status strip (first interior row) ─────────────────────────────────
	statusY := r.Y + 1
	s.FillRect(Rect{X: r.X + 1, Y: statusY, W: r.W - 2, H: 1}, ' ', styleLogsHint)

	var autoText string
	var autoStyle tcell.Style
	if state.LogsAutoScroll {
		autoText = " Autoscroll: On  "
		autoStyle = styleLogsAutoOn
	} else {
		autoText = " Autoscroll: Off "
		autoStyle = styleLogsAutoOff
	}
	s.DrawText(r.X+2, statusY, autoStyle, autoText)

	lineCountText := fmt.Sprintf("  %d lines  ", len(state.LogsLines))
	lineCountX := r.X + r.W - 1 - len([]rune(lineCountText))
	autoEndX := r.X + 2 + len([]rune(autoText))
	if lineCountX > autoEndX+4 {
		s.DrawText(lineCountX, statusY, styleLogsHint, lineCountText)
		hintText := "  j/k: scroll  PgDn/PgUp: page  s: autoscroll  Esc: close"
		hintW := lineCountX - autoEndX - 1
		s.DrawTextTrunc(autoEndX, statusY, hintW, styleLogsHint, hintText)
	}

	// ── log content (rows r.Y+2 … r.Y+r.H-2) ─────────────────────────────
	contentH := r.H - 3 // top border(1) + status strip(1) + bottom border(1)
	if contentH < 1 {
		return
	}

	// lineW reserves 2 chars for the entry marker ("▸ " or "  ").
	// Must match the width used for scroll math in app.go (w-6).
	lineW := r.W - 6

	// Compute total display rows (for offset clamping).
	totalRows := len(WrapLines(state.LogsLines, lineW))

	offset := state.LogsOffset
	if state.LogsAutoScroll {
		offset = totalRows - contentH
		if offset < 0 {
			offset = 0
		}
	} else {
		maxOff := totalRows - contentH
		if maxOff < 0 {
			maxOff = 0
		}
		if offset > maxOff {
			offset = maxOff
		}
		if offset < 0 {
			offset = 0
		}
	}

	// Render log entries with inline wrapping.
	// First display row of each entry gets "▸ "; continuation rows get "  ".
	displayRow := 0
	for _, rawLine := range state.LogsLines {
		if displayRow >= offset+contentH {
			break
		}
		runes := []rune(rawLine)
		first := true
		// Each raw line produces at least one display row (even if empty).
		for first || len(runes) > 0 {
			var chunk string
			if len(runes) <= lineW {
				chunk = string(runes)
				runes = nil
			} else {
				chunk = string(runes[:lineW])
				runes = runes[lineW:]
			}
			if displayRow >= offset {
				y := r.Y + 2 + (displayRow - offset)
				if first {
					s.DrawText(r.X+2, y, styleLogsMarker, "▸ ")
				}
				s.DrawText(r.X+4, y, styleLogsLine, chunk)
			}
			displayRow++
			first = false
			if displayRow >= offset+contentH {
				break
			}
		}
	}
}
