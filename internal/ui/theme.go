package ui

import (
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
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
	colorBg = tcell.NewRGBColor(248, 249, 253)
	colorNodeBg = tcell.NewRGBColor(232, 235, 248)
	colorHeaderBg = tcell.NewRGBColor(225, 228, 242)
	colorTabActiveBg = tcell.NewRGBColor(0, 90, 180)
	colorTabIdleBg = tcell.NewRGBColor(218, 222, 238)
	colorStatusBg = tcell.NewRGBColor(235, 238, 250)
	colorSelectedBg = tcell.NewRGBColor(0, 80, 160)

	colorText = tcell.NewRGBColor(20, 22, 45)
	colorDim = tcell.NewRGBColor(115, 120, 155)
	colorGreen = tcell.NewRGBColor(15, 135, 65)
	colorAmber = tcell.NewRGBColor(150, 100, 0)
	colorOrange = tcell.NewRGBColor(170, 75, 0)
	colorRed = tcell.NewRGBColor(175, 25, 25)
	colorBlue = tcell.NewRGBColor(30, 100, 210)
	colorNodeName = tcell.NewRGBColor(20, 90, 200)
	colorPodName = tcell.NewRGBColor(40, 55, 140)
	colorNamespace = tcell.NewRGBColor(90, 100, 150)

	// ── exported styles (colors.go) ──────────────────────────────────────
	StyleDefault = tcell.StyleDefault.Foreground(colorText).Background(colorBg)
	StyleHeader = tcell.StyleDefault.Foreground(colorDim).Background(colorHeaderBg).Bold(true)
	StyleTabActive = tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(colorTabActiveBg).Bold(true)
	StyleTabInactive = tcell.StyleDefault.Foreground(colorDim).Background(colorTabIdleBg)
	StyleSelected = tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(colorSelectedBg).Bold(true)
	StyleStatusBar = tcell.StyleDefault.Foreground(colorDim).Background(colorStatusBg)
	StyleError = tcell.StyleDefault.Foreground(colorRed).Background(colorBg).Bold(true)

	StyleNodeHeader = tcell.StyleDefault.Foreground(colorText).Background(colorNodeBg).Bold(true)
	StyleNodeReadyDot = tcell.StyleDefault.Foreground(colorGreen).Background(colorNodeBg).Bold(true)
	StyleNodeNotReadyDot = tcell.StyleDefault.Foreground(colorRed).Background(colorNodeBg).Bold(true)
	StyleNodeMeta = tcell.StyleDefault.Foreground(colorBlue).Background(colorNodeBg)

	StylePodRunning = tcell.StyleDefault.Foreground(colorGreen).Background(colorBg)
	StylePodPending = tcell.StyleDefault.Foreground(colorAmber).Background(colorBg)
	StylePodFailed = tcell.StyleDefault.Foreground(colorRed).Background(colorBg)
	StylePodDefault = tcell.StyleDefault.Foreground(colorText).Background(colorBg)

	StyleRestartsWarn = tcell.StyleDefault.Foreground(colorOrange).Background(colorBg)
	StyleRestartsCrit = tcell.StyleDefault.Foreground(colorRed).Background(colorBg).Bold(true)

	StyleNodeReady = tcell.StyleDefault.Foreground(colorGreen).Background(colorBg)
	StyleNodeNotReady = tcell.StyleDefault.Foreground(colorRed).Background(colorBg)

	StyleDim = tcell.StyleDefault.Foreground(colorDim).Background(colorBg)
	StyleSeparator = tcell.StyleDefault.Foreground(tcell.NewRGBColor(170, 175, 210)).Background(colorBg)
	StyleWarning = tcell.StyleDefault.Foreground(colorAmber).Background(tcell.NewRGBColor(255, 248, 215)).Bold(true)
	StylePendingReason = tcell.StyleDefault.Foreground(colorDim).Background(colorBg)

	StyleMetricsOK = tcell.StyleDefault.Foreground(colorGreen).Background(colorNodeBg)
	StyleMetricsWarn = tcell.StyleDefault.Foreground(colorAmber).Background(colorNodeBg)
	StyleMetricsCrit = tcell.StyleDefault.Foreground(colorRed).Background(colorNodeBg).Bold(true)

	StyleNodeName = tcell.StyleDefault.Foreground(colorNodeName).Background(colorNodeBg).Bold(true)
	StylePodName = tcell.StyleDefault.Foreground(colorPodName).Background(colorBg)
	StyleNamespace = tcell.StyleDefault.Foreground(colorNamespace).Background(colorBg)

	StyleXrayNsHeader = tcell.StyleDefault.
		Foreground(tcell.NewRGBColor(0, 120, 110)).
		Background(tcell.NewRGBColor(215, 238, 235)).
		Bold(true)
	StyleXrayConnector = tcell.StyleDefault.
		Foreground(tcell.NewRGBColor(155, 160, 195)).
		Background(colorBg)

	StyleHeatmapRunning = tcell.StyleDefault.Foreground(colorGreen).Background(colorBg)
	StyleHeatmapPending = tcell.StyleDefault.Foreground(colorAmber).Background(colorBg)
	StyleHeatmapFailed = tcell.StyleDefault.Foreground(colorRed).Background(colorBg).Bold(true)
	StyleHeatmapTerm = tcell.StyleDefault.Foreground(colorDim).Background(colorBg)
	StyleHeatmapDefault = tcell.StyleDefault.Foreground(colorBlue).Background(colorBg)
	StyleHeatmapNodeSel = tcell.StyleDefault.
		Foreground(tcell.NewRGBColor(180, 130, 0)).
		Background(colorBg).
		Bold(true)

	// ── logs overlay private styles (logsview.go) ────────────────────────
	logsBg := tcell.NewRGBColor(215, 220, 240)
	styleLogsTitleBg = logsBg
	styleLogsTitle = tcell.StyleDefault.
		Background(logsBg).
		Foreground(tcell.NewRGBColor(30, 100, 210)).
		Bold(true)
	styleLogsHint = tcell.StyleDefault.
		Background(tcell.NewRGBColor(225, 228, 242)).
		Foreground(tcell.NewRGBColor(115, 120, 155))
	styleLogsLine = tcell.StyleDefault.
		Background(tcell.NewRGBColor(240, 242, 252)).
		Foreground(tcell.NewRGBColor(20, 22, 45))
	styleLogsMarker = tcell.StyleDefault.
		Background(tcell.NewRGBColor(240, 242, 252)).
		Foreground(tcell.NewRGBColor(40, 100, 200))
	styleLogsAutoOn = tcell.StyleDefault.
		Background(tcell.NewRGBColor(225, 228, 242)).
		Foreground(tcell.NewRGBColor(15, 135, 65)).
		Bold(true)
	styleLogsAutoOff = tcell.StyleDefault.
		Background(tcell.NewRGBColor(225, 228, 242)).
		Foreground(tcell.NewRGBColor(115, 120, 155))
	styleLogsBorder = tcell.StyleDefault.
		Background(tcell.NewRGBColor(240, 242, 252)).
		Foreground(tcell.NewRGBColor(80, 120, 200))

	// ── help overlay private styles (help.go) ────────────────────────────
	helpBg := tcell.NewRGBColor(230, 233, 248)
	styleHelpBg = tcell.StyleDefault.Background(helpBg).Foreground(tcell.NewRGBColor(20, 22, 45))
	styleHelpTitle = tcell.StyleDefault.Background(helpBg).Foreground(tcell.NewRGBColor(20, 90, 200)).Bold(true)
	styleHelpKey = tcell.StyleDefault.Background(helpBg).Foreground(tcell.NewRGBColor(20, 90, 200))
	styleHelpSection = tcell.StyleDefault.Background(helpBg).Foreground(tcell.NewRGBColor(15, 135, 65)).Bold(true)
	styleHelpDim = tcell.StyleDefault.Background(helpBg).Foreground(tcell.NewRGBColor(115, 120, 155))

	// ── cluster picker overlay private styles (picker.go) ────────────────
	stylePickerBg = tcell.StyleDefault.
		Background(tcell.NewRGBColor(248, 249, 253)).
		Foreground(tcell.NewRGBColor(20, 22, 45))
	stylePickerTitle = tcell.StyleDefault.
		Background(tcell.NewRGBColor(248, 249, 253)).
		Foreground(tcell.NewRGBColor(20, 90, 200)).
		Bold(true)
	stylePickerItem = tcell.StyleDefault.
		Background(tcell.NewRGBColor(232, 235, 248)).
		Foreground(tcell.NewRGBColor(20, 22, 45))
	stylePickerSelected = tcell.StyleDefault.
		Background(tcell.NewRGBColor(0, 80, 160)).
		Foreground(tcell.ColorWhite).
		Bold(true)
	stylePickerCurrent = tcell.StyleDefault.
		Background(tcell.NewRGBColor(232, 235, 248)).
		Foreground(tcell.NewRGBColor(15, 135, 65))
	stylePickerHint = tcell.StyleDefault.
		Background(tcell.NewRGBColor(232, 235, 248)).
		Foreground(tcell.NewRGBColor(115, 120, 155))
}
