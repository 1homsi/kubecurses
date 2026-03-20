package ui

import (
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// InitTheme applies a color theme based on the preference string.
// preference may be "auto", "dark", or "light".
// "auto" detects the system/terminal appearance and applies light theme when
// a light background is detected. "dark" keeps the built-in defaults.
func InitTheme(preference string) {
	switch strings.ToLower(preference) {
	case "light":
		ApplyLightTheme()
	case "dark":
		// built-in defaults are already dark — nothing to do
	default: // "auto"
		if !detectDarkMode() {
			ApplyLightTheme()
		}
	}
}

// detectDarkMode returns true when the terminal/system is using a dark theme.
// Detection order:
//  1. KUBECURSES_THEME env var ("light" / "dark")
//  2. macOS defaults read -g AppleInterfaceStyle
//  3. COLORFGBG env var (xterm, rxvt, older terminals)
//  4. Default: dark
func detectDarkMode() bool {
	// 1. Explicit env override.
	switch strings.ToLower(os.Getenv("KUBECURSES_THEME")) {
	case "light":
		return false
	case "dark":
		return true
	}

	// 2. macOS system appearance (only the "Dark" key exists when dark mode is on).
	out, err := exec.Command("defaults", "read", "-g", "AppleInterfaceStyle").Output()
	if err == nil {
		return strings.EqualFold(strings.TrimSpace(string(out)), "dark")
	}
	// If "defaults" exists but the key is missing → macOS light mode.
	if _, lookErr := exec.LookPath("defaults"); lookErr == nil {
		return false
	}

	// 3. COLORFGBG: "fg;bg" — bg index < 8 means dark terminal.
	if fgbg := os.Getenv("COLORFGBG"); fgbg != "" {
		parts := strings.Split(fgbg, ";")
		if len(parts) >= 2 {
			if n, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
				return n < 8
			}
		}
	}

	return true // default: assume dark
}

// ApplyLightTheme reassigns every style variable in the ui package so that
// the TUI renders comfortably on a light-background terminal. It must be
// called before the first screen draw (i.e. before app.New).
func ApplyLightTheme() {
	// ── color primitives ─────────────────────────────────────────────────
	colorBg = lipgloss.Color("#F8F9FD")
	colorNodeBg = lipgloss.Color("#E8EBF8")
	colorHeaderBg = lipgloss.Color("#E1E4F2")
	colorTabActiveBg = lipgloss.Color("#005AB4")
	colorTabIdleBg = lipgloss.Color("#DADEEE")
	colorStatusBg = lipgloss.Color("#EBEEFA")
	colorSelectedBg = lipgloss.Color("#0050A0")

	colorText = lipgloss.Color("#14162D")
	colorDim = lipgloss.Color("#73789B")
	colorGreen = lipgloss.Color("#0F8741")
	colorAmber = lipgloss.Color("#966400")
	colorOrange = lipgloss.Color("#AA4B00")
	colorRed = lipgloss.Color("#AF1919")
	colorBlue = lipgloss.Color("#1E64D2")
	colorNodeName = lipgloss.Color("#145AC8")
	colorPodName = lipgloss.Color("#28378C")
	colorNamespace = lipgloss.Color("#5A6496")

	// ── exported styles (colors.go) ──────────────────────────────────────
	StyleDefault = lipgloss.NewStyle().Foreground(colorText).Background(colorBg)
	StyleHeader = lipgloss.NewStyle().Foreground(colorDim).Background(colorHeaderBg).Bold(true)
	StyleTabActive = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Background(colorTabActiveBg).Bold(true)
	StyleTabInactive = lipgloss.NewStyle().Foreground(colorDim).Background(colorTabIdleBg)
	StyleSelected = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Background(colorSelectedBg).Bold(true)
	StyleStatusBar = lipgloss.NewStyle().Foreground(colorDim).Background(colorStatusBg)
	StyleError = lipgloss.NewStyle().Foreground(colorRed).Background(colorBg).Bold(true)

	StyleNodeHeader = lipgloss.NewStyle().Foreground(colorText).Background(colorNodeBg).Bold(true)
	StyleNodeReadyDot = lipgloss.NewStyle().Foreground(colorGreen).Background(colorNodeBg).Bold(true)
	StyleNodeNotReadyDot = lipgloss.NewStyle().Foreground(colorRed).Background(colorNodeBg).Bold(true)
	StyleNodeMeta = lipgloss.NewStyle().Foreground(colorBlue).Background(colorNodeBg)

	StylePodRunning = lipgloss.NewStyle().Foreground(colorGreen).Background(colorBg)
	StylePodPending = lipgloss.NewStyle().Foreground(colorAmber).Background(colorBg)
	StylePodFailed = lipgloss.NewStyle().Foreground(colorRed).Background(colorBg)
	StylePodDefault = lipgloss.NewStyle().Foreground(colorText).Background(colorBg)

	StyleRestartsWarn = lipgloss.NewStyle().Foreground(colorOrange).Background(colorBg)
	StyleRestartsCrit = lipgloss.NewStyle().Foreground(colorRed).Background(colorBg).Bold(true)

	StyleNodeReady = lipgloss.NewStyle().Foreground(colorGreen).Background(colorBg)
	StyleNodeNotReady = lipgloss.NewStyle().Foreground(colorRed).Background(colorBg)

	StyleDim = lipgloss.NewStyle().Foreground(colorDim).Background(colorBg)
	StyleSeparator = lipgloss.NewStyle().Foreground(lipgloss.Color("#AAAFD2")).Background(colorBg)
	StyleWarning = lipgloss.NewStyle().Foreground(colorAmber).Background(lipgloss.Color("#FFF8D7")).Bold(true)
	StylePendingReason = lipgloss.NewStyle().Foreground(colorDim).Background(colorBg)

	StyleMetricsOK = lipgloss.NewStyle().Foreground(colorGreen).Background(colorNodeBg)
	StyleMetricsWarn = lipgloss.NewStyle().Foreground(colorAmber).Background(colorNodeBg)
	StyleMetricsCrit = lipgloss.NewStyle().Foreground(colorRed).Background(colorNodeBg).Bold(true)

	StyleNodeName = lipgloss.NewStyle().Foreground(colorNodeName).Background(colorNodeBg).Bold(true)
	StylePodName = lipgloss.NewStyle().Foreground(colorPodName).Background(colorBg)
	StyleNamespace = lipgloss.NewStyle().Foreground(colorNamespace).Background(colorBg)

	StyleXrayNsHeader = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00786E")).
		Background(lipgloss.Color("#D7EEEB")).
		Bold(true)
	StyleXrayConnector = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9BA0C3")).
		Background(colorBg)

	StyleHeatmapRunning = lipgloss.NewStyle().Foreground(colorGreen).Background(colorNodeBg)
	StyleHeatmapPending = lipgloss.NewStyle().Foreground(colorAmber).Background(colorNodeBg)
	StyleHeatmapFailed = lipgloss.NewStyle().Foreground(colorRed).Background(colorNodeBg).Bold(true)
	StyleHeatmapTerm = lipgloss.NewStyle().Foreground(colorDim).Background(colorNodeBg)
	StyleHeatmapDefault = lipgloss.NewStyle().Foreground(colorBlue).Background(colorNodeBg)
	StyleHeatmapNodeSel = lipgloss.NewStyle().
		Background(lipgloss.Color("#A06E00")).
		Foreground(colorBg)

	// ── logs overlay private styles (logsview.go) ────────────────────────
	logsBg := lipgloss.Color("#D7DCF0")
	styleLogsTitleBg = logsBg
	styleLogsTitle = lipgloss.NewStyle().
		Background(logsBg).
		Foreground(lipgloss.Color("#1E64D2")).
		Bold(true)
	styleLogsHint = lipgloss.NewStyle().
		Background(lipgloss.Color("#E1E4F2")).
		Foreground(lipgloss.Color("#73789B"))
	styleLogsLine = lipgloss.NewStyle().
		Background(lipgloss.Color("#F0F2FC")).
		Foreground(lipgloss.Color("#14162D"))
	styleLogsMarker = lipgloss.NewStyle().
		Background(lipgloss.Color("#F0F2FC")).
		Foreground(lipgloss.Color("#2864C8"))
	styleLogsAutoOn = lipgloss.NewStyle().
		Background(lipgloss.Color("#E1E4F2")).
		Foreground(lipgloss.Color("#0F8741")).
		Bold(true)
	styleLogsAutoOff = lipgloss.NewStyle().
		Background(lipgloss.Color("#E1E4F2")).
		Foreground(lipgloss.Color("#73789B"))
	styleLogsBorder = lipgloss.NewStyle().
		Background(lipgloss.Color("#F0F2FC")).
		Foreground(lipgloss.Color("#5078C8"))

	// ── help overlay private styles (help.go) ────────────────────────────
	helpBg := lipgloss.Color("#E6E9F8")
	styleHelpBg = lipgloss.NewStyle().Background(helpBg).Foreground(lipgloss.Color("#14162D"))
	styleHelpTitle = lipgloss.NewStyle().Background(helpBg).Foreground(lipgloss.Color("#145AC8")).Bold(true)
	styleHelpKey = lipgloss.NewStyle().Background(helpBg).Foreground(lipgloss.Color("#145AC8"))
	styleHelpSection = lipgloss.NewStyle().Background(helpBg).Foreground(lipgloss.Color("#0F8741")).Bold(true)
	styleHelpDim = lipgloss.NewStyle().Background(helpBg).Foreground(lipgloss.Color("#73789B"))

	// ── cluster picker overlay private styles (picker.go) ────────────────
	stylePickerBg = lipgloss.NewStyle().
		Background(lipgloss.Color("#F8F9FD")).
		Foreground(lipgloss.Color("#14162D"))
	stylePickerTitle = lipgloss.NewStyle().
		Background(lipgloss.Color("#F8F9FD")).
		Foreground(lipgloss.Color("#145AC8")).
		Bold(true)
	stylePickerItem = lipgloss.NewStyle().
		Background(lipgloss.Color("#E8EBF8")).
		Foreground(lipgloss.Color("#14162D"))
	stylePickerSelected = lipgloss.NewStyle().
		Background(lipgloss.Color("#0050A0")).
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true)
	stylePickerCurrent = lipgloss.NewStyle().
		Background(lipgloss.Color("#E8EBF8")).
		Foreground(lipgloss.Color("#0F8741"))
	stylePickerHint = lipgloss.NewStyle().
		Background(lipgloss.Color("#E8EBF8")).
		Foreground(lipgloss.Color("#73789B"))
}
