package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"

	"github.com/1homsi/kubecurses/internal/model"
)

var (
	styleExecTitleBg = tcell.NewRGBColor(10, 22, 18)
	styleExecTitle   = tcell.StyleDefault.Background(styleExecTitleBg).Foreground(tcell.NewRGBColor(80, 220, 140)).Bold(true)
	styleExecHint    = tcell.StyleDefault.Background(tcell.NewRGBColor(12, 20, 16)).Foreground(tcell.NewRGBColor(100, 105, 130))
	styleExecLine    = tcell.StyleDefault.Background(tcell.NewRGBColor(10, 16, 12)).Foreground(tcell.NewRGBColor(195, 210, 200))
	styleExecMarker  = tcell.StyleDefault.Background(tcell.NewRGBColor(10, 16, 12)).Foreground(tcell.NewRGBColor(60, 200, 120))
	styleExecAutoOn  = tcell.StyleDefault.Background(tcell.NewRGBColor(12, 20, 16)).Foreground(tcell.NewRGBColor(80, 200, 120)).Bold(true)
	styleExecAutoOff = tcell.StyleDefault.Background(tcell.NewRGBColor(12, 20, 16)).Foreground(tcell.NewRGBColor(100, 105, 130))
	styleExecBorder  = tcell.StyleDefault.Background(tcell.NewRGBColor(10, 16, 12)).Foreground(tcell.NewRGBColor(40, 160, 90))
)

// CachedWrapExec returns the wrapped form of state.ExecLines for lineW, using
// a cache stored in state to avoid recomputing when nothing has changed.
func CachedWrapExec(state *model.AppState, lineW int) []string {
	if lineW == state.ExecWrapWidth && len(state.ExecLines) == state.ExecWrapCount {
		return state.ExecWrapped
	}
	state.ExecWrapped = WrapLines(state.ExecLines, lineW)
	state.ExecWrapWidth = lineW
	state.ExecWrapCount = len(state.ExecLines)
	return state.ExecWrapped
}

// DrawExecView renders the exec output overlay within content rect r.
// Layout mirrors DrawLogsView; a green-tinted border distinguishes it from logs.
//
//	row 0      : top border with centred title
//	row 1      : status strip (autoscroll + hint + line count)
//	rows 2…H-2 : command output (hard-wrapped to fit width)
//	row H-1    : bottom border
func DrawExecView(s *Screen, r Rect, state *model.AppState) {
	if r.W < 4 || r.H < 4 {
		return
	}

	// ── fill interior background ───────────────────────────────────────────
	for i := 1; i < r.H-1; i++ {
		s.FillRect(Rect{X: r.X + 1, Y: r.Y + i, W: r.W - 2, H: 1}, ' ', styleExecLine)
	}

	// ── border ────────────────────────────────────────────────────────────
	drawBorderOnly(s, r.X, r.Y, r.W, r.H, styleExecBorder)

	// ── title centred on top border row ───────────────────────────────────
	podLabel := state.ExecNamespace + "/" + state.ExecPod
	title := " Exec — " + podLabel
	if state.ExecContainer != "" {
		title += " [" + state.ExecContainer + "]"
	}
	title += " "
	titleX := r.X + (r.W-len([]rune(title)))/2
	if titleX < r.X+1 {
		titleX = r.X + 1
	}
	s.DrawTextTrunc(titleX, r.Y, r.W-2, styleExecTitle, title)

	// ── status strip (first interior row) ─────────────────────────────────
	statusY := r.Y + 1
	s.FillRect(Rect{X: r.X + 1, Y: statusY, W: r.W - 2, H: 1}, ' ', styleExecHint)

	var autoText string
	var autoStyle tcell.Style
	if state.ExecAutoScroll {
		autoText = " Autoscroll: On  "
		autoStyle = styleExecAutoOn
	} else {
		autoText = " Autoscroll: Off "
		autoStyle = styleExecAutoOff
	}
	s.DrawText(r.X+2, statusY, autoStyle, autoText)

	lineCountText := fmt.Sprintf("  %d lines  ", len(state.ExecLines))
	lineCountX := r.X + r.W - 1 - len([]rune(lineCountText))
	autoEndX := r.X + 2 + len([]rune(autoText))
	if lineCountX > autoEndX+4 {
		s.DrawText(lineCountX, statusY, styleExecHint, lineCountText)
		hintText := "  j/k: scroll  PgDn/PgUp: page  s: autoscroll  Esc: close"
		hintW := lineCountX - autoEndX - 1
		s.DrawTextTrunc(autoEndX, statusY, hintW, styleExecHint, hintText)
	}

	// ── output content (rows r.Y+2 … r.Y+r.H-2) ──────────────────────────
	contentH := r.H - 3 // top border(1) + status strip(1) + bottom border(1)
	if contentH < 1 {
		return
	}

	lineW := r.W - 6 // 2 indent + 2 marker + 2 border margin

	totalRows := len(CachedWrapExec(state, lineW))

	offset := state.ExecOffset
	if state.ExecAutoScroll {
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

	displayRow := 0
	for _, rawLine := range state.ExecLines {
		if displayRow >= offset+contentH {
			break
		}
		runes := []rune(rawLine)
		first := true
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
					s.DrawText(r.X+2, y, styleExecMarker, "▸ ")
				}
				s.DrawText(r.X+4, y, styleExecLine, chunk)
			}
			displayRow++
			first = false
			if displayRow >= offset+contentH {
				break
			}
		}
	}
}
