package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	styleHelpBg      = lipgloss.NewStyle().Background(lipgloss.Color("#121626")).Foreground(lipgloss.Color("#DCDEEB"))
	styleHelpTitle   = lipgloss.NewStyle().Background(lipgloss.Color("#121626")).Foreground(lipgloss.Color("#82BEFF")).Bold(true)
	styleHelpKey     = lipgloss.NewStyle().Background(lipgloss.Color("#121626")).Foreground(lipgloss.Color("#82BEFF"))
	styleHelpSection = lipgloss.NewStyle().Background(lipgloss.Color("#121626")).Foreground(lipgloss.Color("#50C878")).Bold(true)
	styleHelpDim     = lipgloss.NewStyle().Background(lipgloss.Color("#121626")).Foreground(lipgloss.Color("#646982"))
)

type helpLine struct {
	key  string // empty = section header (uses desc) or blank line (both empty)
	desc string
}

var helpLines = []helpLine{
	{key: "", desc: "Navigation"},
	{key: "j / k  ↓ / ↑", desc: "Move down / up"},
	{key: "PgDn / PgUp", desc: "Page down / up"},
	{key: "", desc: ""},
	{key: "", desc: "Tabs"},
	{key: "Tab / Shift+Tab", desc: "Next / previous tab"},
	{key: "1  2  3  4  5  6", desc: "Jump to tab directly"},
	{key: "", desc: ""},
	{key: "", desc: "Actions"},
	{key: "l", desc: "Stream logs for selected pod"},
	{key: "e", desc: "Exec command in selected pod"},
	{key: "c", desc: "Switch cluster / context"},
	{key: "/", desc: "Search / filter"},
	{key: "Esc", desc: "Clear search or close"},
	{key: "r", desc: "Manual refresh"},
	{key: "?", desc: "Toggle this help"},
	{key: "q  Ctrl+C", desc: "Quit"},
	{key: "", desc: ""},
	{key: "", desc: "Logs view"},
	{key: "s", desc: "Toggle autoscroll"},
	{key: "j / k  PgDn / PgUp", desc: "Scroll"},
	{key: "Esc", desc: "Close overlay"},
	{key: "", desc: ""},
	{key: "", desc: "Exec (interactive shell)"},
	{key: "e", desc: "Open shell — TUI suspends"},
	{key: "exit / Ctrl+D", desc: "End session, return to TUI"},
}

// RenderHelp renders a centered help overlay and returns it as a multi-line string.
func RenderHelp(screenW, screenH int) string {
	const boxW = 46
	w := boxW
	if w > screenW-4 {
		w = screenW - 4
	}
	// border(2) + blank after title(1) + content(N) + blank before hint(1) + hint(1)
	h := len(helpLines) + 5

	var lines []string

	// Top border with title.
	borderStyle := stylePickerItem
	topBorder := "╭" + strings.Repeat("─", w-2) + "╮"
	title := " Keyboard Shortcuts "
	titleRunes := []rune(title)
	titleStart := (w - len(titleRunes)) / 2
	if titleStart < 1 {
		titleStart = 1
	}
	topRunes := []rune(topBorder)
	for i, ch := range titleRunes {
		pos := titleStart + i
		if pos < len(topRunes)-1 {
			topRunes[pos] = ch
		}
	}
	lines = append(lines, borderStyle.Render(string(topRunes[:titleStart]))+
		styleHelpTitle.Render(title)+
		borderStyle.Render(string(topRunes[titleStart+len(titleRunes):])))

	// Blank row.
	lines = append(lines, borderStyle.Render("│")+styleHelpBg.Render(strings.Repeat(" ", w-2))+borderStyle.Render("│"))

	// Help content.
	const keyColW = 18
	for _, l := range helpLines {
		if l.key == "" {
			if l.desc != "" {
				content := styleHelpSection.Render(PadRight(l.desc, w-4))
				lines = append(lines, borderStyle.Render("│")+styleHelpBg.Render(" ")+content+styleHelpBg.Render(" ")+borderStyle.Render("│"))
			} else {
				lines = append(lines, borderStyle.Render("│")+styleHelpBg.Render(strings.Repeat(" ", w-2))+borderStyle.Render("│"))
			}
			continue
		}
		keyText := styleHelpKey.Render(fmt.Sprintf("%-*s", keyColW, l.key))
		descText := styleHelpBg.Render(PadRight(l.desc, w-4-keyColW))
		lines = append(lines, borderStyle.Render("│")+styleHelpBg.Render(" ")+keyText+descText+styleHelpBg.Render(" ")+borderStyle.Render("│"))
	}

	// Blank row.
	lines = append(lines, borderStyle.Render("│")+styleHelpBg.Render(strings.Repeat(" ", w-2))+borderStyle.Render("│"))

	// Hint row.
	hint := "  Press any key to close  "
	hintContent := styleHelpDim.Render(PadRight(hint, w-2))
	lines = append(lines, borderStyle.Render("│")+hintContent+borderStyle.Render("│"))

	// Bottom border.
	lines = append(lines, borderStyle.Render("╰"+strings.Repeat("─", w-2)+"╯"))

	// Pad to h.
	for len(lines) < h {
		lines = append(lines, borderStyle.Render("│")+styleHelpBg.Render(strings.Repeat(" ", w-2))+borderStyle.Render("│"))
	}

	return strings.Join(lines, "\n")
}
