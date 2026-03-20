package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/1homsi/kubecurses/internal/model"
)

var (
	styleLogsTitleBg lipgloss.Color = "#0C1220"
	styleLogsTitle                  = lipgloss.NewStyle().Background(styleLogsTitleBg).Foreground(lipgloss.Color("#82BEFF")).Bold(true)
	styleLogsHint                   = lipgloss.NewStyle().Background(lipgloss.Color("#10121C")).Foreground(lipgloss.Color("#646982"))
	styleLogsLine                   = lipgloss.NewStyle().Background(lipgloss.Color("#101010")).Foreground(lipgloss.Color("#C3C8DA"))
	styleLogsMarker                 = lipgloss.NewStyle().Background(lipgloss.Color("#101010")).Foreground(lipgloss.Color("#4678D2"))
	styleLogsAutoOn                 = lipgloss.NewStyle().Background(lipgloss.Color("#10121C")).Foreground(lipgloss.Color("#50C878")).Bold(true)
	styleLogsAutoOff                = lipgloss.NewStyle().Background(lipgloss.Color("#10121C")).Foreground(lipgloss.Color("#646982"))
	styleLogsBorder                 = lipgloss.NewStyle().Background(lipgloss.Color("#101010")).Foreground(lipgloss.Color("#325096"))
)

// CachedWrapLogs returns the wrapped form of state.LogsLines for lineW, using
// a cache stored in the state to avoid recomputing when neither the width nor
// the number of lines has changed.
func CachedWrapLogs(state *model.AppState, lineW int) []string {
	if lineW == state.LogsWrapWidth && len(state.LogsLines) == state.LogsWrapCount {
		return state.LogsWrapped
	}
	state.LogsWrapped = WrapLines(state.LogsLines, lineW)
	state.LogsWrapWidth = lineW
	state.LogsWrapCount = len(state.LogsLines)
	return state.LogsWrapped
}

// WrapLines splits each element of lines into segments of at most maxW runes,
// returning a flat slice of display rows.
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

// RenderDescribeOverlay renders a scrollable describe output box and returns it as a string.
func RenderDescribeOverlay(width, height int, state *model.AppState) string {
	if width < 4 || height < 4 {
		return ""
	}

	var lines []string

	// Top border with title.
	border := RenderBorder(width, height, styleLogsBorder)
	title := " " + state.DescribeTitle + " "
	titleRunes := []rune(title)
	titleStart := (width - len(titleRunes)) / 2
	if titleStart < 1 {
		titleStart = 1
	}
	// Build top border with embedded title.
	topBorderRunes := []rune(border[0])
	for i, ch := range titleRunes {
		pos := titleStart + i
		if pos < len(topBorderRunes)-1 {
			topBorderRunes[pos] = ch
		}
	}
	lines = append(lines, styleLogsBorder.Render(string(topBorderRunes[:titleStart]))+
		styleLogsTitle.Render(title)+
		styleLogsBorder.Render(string(topBorderRunes[titleStart+len(titleRunes):])))

	// Hint/status row.
	lineCountText := fmt.Sprintf("  %d lines  ", len(state.DescribeLines))
	hintText := "  j/k: scroll  PgDn/PgUp: page  Esc: close"
	hintW := width - 2 - len([]rune(lineCountText))
	if hintW < 0 {
		hintW = 0
	}
	hintRow := styleLogsBorder.Render("│") +
		styleLogsHint.Render(PadRight(Truncate(hintText, hintW), hintW)) +
		styleLogsHint.Render(lineCountText) +
		styleLogsBorder.Render("│")
	lines = append(lines, hintRow)

	// Content rows.
	contentH := height - 3
	if contentH < 1 {
		contentH = 0
	}
	lineW := width - 6
	offset := state.DescribeOffset
	maxOff := len(state.DescribeLines) - contentH
	if maxOff < 0 {
		maxOff = 0
	}
	if offset > maxOff {
		offset = maxOff
	}
	if offset < 0 {
		offset = 0
	}
	for i := 0; i < contentH; i++ {
		idx := offset + i
		var content string
		if idx < len(state.DescribeLines) {
			marker := styleLogsMarker.Render("▸ ")
			text := styleLogsLine.Render(PadRight(Truncate(state.DescribeLines[idx], lineW), lineW))
			content = marker + text
		} else {
			content = styleLogsLine.Render(strings.Repeat(" ", lineW+2))
		}
		lines = append(lines, styleLogsBorder.Render("│")+
			styleLogsLine.Render(" ")+
			content+
			styleLogsLine.Render(" ")+
			styleLogsBorder.Render("│"))
	}

	// Bottom border.
	lines = append(lines, border[height-1])

	return strings.Join(lines, "\n")
}

// RenderLogsView renders a bordered log streaming box and returns it as a string.
func RenderLogsView(width, height int, state *model.AppState) string {
	if width < 4 || height < 4 {
		return ""
	}

	var lines []string

	// Top border with title.
	border := RenderBorder(width, height, styleLogsBorder)
	podLabel := state.LogsNamespace + "/" + state.LogsPod
	title := " Logs — " + podLabel
	if state.LogsContainer != "" {
		title += " [" + state.LogsContainer + "]"
	}
	title += " "
	titleRunes := []rune(title)
	titleStart := (width - len(titleRunes)) / 2
	if titleStart < 1 {
		titleStart = 1
	}
	topBorderRunes := []rune(border[0])
	for i, ch := range titleRunes {
		pos := titleStart + i
		if pos < len(topBorderRunes)-1 {
			topBorderRunes[pos] = ch
		}
	}
	lines = append(lines, styleLogsBorder.Render(string(topBorderRunes[:titleStart]))+
		styleLogsTitle.Render(title)+
		styleLogsBorder.Render(string(topBorderRunes[titleStart+len(titleRunes):])))

	// Status strip.
	var autoText string
	var autoStyle lipgloss.Style
	if state.LogsAutoScroll {
		autoText = " Autoscroll: On  "
		autoStyle = styleLogsAutoOn
	} else {
		autoText = " Autoscroll: Off "
		autoStyle = styleLogsAutoOff
	}
	lineCountText := fmt.Sprintf("  %d lines  ", len(state.LogsLines))
	hintText := "  j/k: scroll  PgDn/PgUp: page  s: autoscroll  Esc: close"
	middleW := width - 2 - len([]rune(autoText)) - len([]rune(lineCountText))
	if middleW < 0 {
		middleW = 0
	}
	statusRow := styleLogsBorder.Render("│") +
		autoStyle.Render(autoText) +
		styleLogsHint.Render(PadRight(Truncate(hintText, middleW), middleW)) +
		styleLogsHint.Render(lineCountText) +
		styleLogsBorder.Render("│")
	lines = append(lines, statusRow)

	// Log content.
	contentH := height - 3
	if contentH < 1 {
		contentH = 0
	}
	lineW := width - 6

	totalRows := len(CachedWrapLogs(state, lineW))
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
	displayRow := 0
	for _, rawLine := range state.LogsLines {
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
			if displayRow >= offset && displayRow < offset+contentH {
				var marker string
				if first {
					marker = styleLogsMarker.Render("▸ ")
				} else {
					marker = styleLogsLine.Render("  ")
				}
				content := styleLogsLine.Render(PadRight(chunk, lineW))
				lines = append(lines, styleLogsBorder.Render("│")+
					styleLogsLine.Render(" ")+
					marker+
					content+
					styleLogsLine.Render(" ")+
					styleLogsBorder.Render("│"))
			}
			displayRow++
			first = false
			if displayRow >= offset+contentH {
				break
			}
		}
	}

	// Fill remaining content rows.
	for len(lines) < height-1 {
		emptyContent := styleLogsLine.Render(strings.Repeat(" ", width-2))
		lines = append(lines, styleLogsBorder.Render("│")+emptyContent+styleLogsBorder.Render("│"))
	}

	// Bottom border.
	lines = append(lines, border[height-1])

	return strings.Join(lines, "\n")
}
