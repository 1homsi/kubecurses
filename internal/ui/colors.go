// Package ui contains tcell screen management and rendering primitives.
package ui

import "github.com/gdamore/tcell/v2"

// Dark theme palette — RGB values so they look identical across terminals.
var (
	// Backgrounds
	colorBg          = tcell.NewRGBColor(13, 14, 20)   // near-black
	colorNodeBg      = tcell.NewRGBColor(22, 26, 46)   // node section header
	colorHeaderBg    = tcell.NewRGBColor(18, 22, 38)   // table column headers
	colorTabActiveBg = tcell.NewRGBColor(0, 100, 200)  // active tab
	colorTabIdleBg   = tcell.NewRGBColor(20, 22, 36)   // inactive tab
	colorStatusBg    = tcell.NewRGBColor(16, 18, 28)   // status bar
	colorSelectedBg  = tcell.NewRGBColor(0, 68, 148)   // selected row

	// Foregrounds
	colorText      = tcell.NewRGBColor(220, 222, 235) // default text
	colorDim       = tcell.NewRGBColor(100, 105, 130) // dim / inactive
	colorGreen     = tcell.NewRGBColor(80, 200, 120)  // Running / Ready
	colorAmber     = tcell.NewRGBColor(210, 160, 50)  // Pending / warn
	colorOrange    = tcell.NewRGBColor(220, 120, 40)  // restart warning
	colorRed       = tcell.NewRGBColor(210, 65, 65)   // Failed / NotReady / high restarts
	colorBlue      = tcell.NewRGBColor(100, 170, 255) // version / metadata
	colorNodeName  = tcell.NewRGBColor(130, 190, 255) // node name — bright sky blue
	colorPodName   = tcell.NewRGBColor(200, 210, 255) // pod name — light lavender white
	colorNamespace = tcell.NewRGBColor(110, 120, 160) // namespace — muted blue-gray
)

var (
	StyleDefault = tcell.StyleDefault.
			Foreground(colorText).
			Background(colorBg)

	StyleHeader = tcell.StyleDefault.
			Foreground(colorDim).
			Background(colorHeaderBg).
			Bold(true)

	StyleTabActive = tcell.StyleDefault.
			Foreground(tcell.ColorWhite).
			Background(colorTabActiveBg).
			Bold(true)

	StyleTabInactive = tcell.StyleDefault.
				Foreground(colorDim).
				Background(colorTabIdleBg)

	StyleSelected = tcell.StyleDefault.
			Foreground(tcell.ColorWhite).
			Background(colorSelectedBg).
			Bold(true)

	StyleStatusBar = tcell.StyleDefault.
			Foreground(colorDim).
			Background(colorStatusBg)

	StyleError = tcell.StyleDefault.
			Foreground(colorRed).
			Background(colorBg).
			Bold(true)

	// Node section header row
	StyleNodeHeader = tcell.StyleDefault.
			Foreground(colorText).
			Background(colorNodeBg).
			Bold(true)

	StyleNodeReadyDot = tcell.StyleDefault.
				Foreground(colorGreen).
				Background(colorNodeBg).
				Bold(true)

	StyleNodeNotReadyDot = tcell.StyleDefault.
				Foreground(colorRed).
				Background(colorNodeBg).
				Bold(true)

	StyleNodeMeta = tcell.StyleDefault.
			Foreground(colorBlue).
			Background(colorNodeBg)

	// Pod status styles
	StylePodRunning = tcell.StyleDefault.
			Foreground(colorGreen).
			Background(colorBg)

	StylePodPending = tcell.StyleDefault.
			Foreground(colorAmber).
			Background(colorBg)

	StylePodFailed = tcell.StyleDefault.
			Foreground(colorRed).
			Background(colorBg)

	StylePodDefault = tcell.StyleDefault.
			Foreground(colorText).
			Background(colorBg)

	// Restart count styles (applied to the restart count cell only)
	StyleRestartsWarn = tcell.StyleDefault.
				Foreground(colorOrange).
				Background(colorBg)

	StyleRestartsCrit = tcell.StyleDefault.
				Foreground(colorRed).
				Background(colorBg).
				Bold(true)

	// Node status (used in flat nodes table)
	StyleNodeReady = tcell.StyleDefault.
			Foreground(colorGreen).
			Background(colorBg)

	StyleNodeNotReady = tcell.StyleDefault.
				Foreground(colorRed).
				Background(colorBg)

	StyleDim = tcell.StyleDefault.
			Foreground(colorDim).
			Background(colorBg)

	StyleSeparator = tcell.StyleDefault.
			Foreground(tcell.NewRGBColor(45, 50, 75)).
			Background(colorBg)

	// Warning banner row (scheduling imbalance, etc.)
	StyleWarning = tcell.StyleDefault.
			Foreground(colorAmber).
			Background(tcell.NewRGBColor(30, 24, 8)).
			Bold(true)

	// Pending reason sub-row
	StylePendingReason = tcell.StyleDefault.
				Foreground(colorDim).
				Background(colorBg)

	// Metrics percentages — colour by severity
	StyleMetricsOK   = tcell.StyleDefault.Foreground(colorGreen).Background(colorNodeBg)
	StyleMetricsWarn = tcell.StyleDefault.Foreground(colorAmber).Background(colorNodeBg)
	StyleMetricsCrit = tcell.StyleDefault.Foreground(colorRed).Background(colorNodeBg).Bold(true)

	// Node name — bright sky blue, stands out in node header rows
	StyleNodeName = tcell.StyleDefault.
			Foreground(colorNodeName).
			Background(colorNodeBg).
			Bold(true)

	// Pod name — light lavender-white so it reads clearly on dark bg
	StylePodName = tcell.StyleDefault.
			Foreground(colorPodName).
			Background(colorBg)

	// Namespace label — muted so name and status are visually dominant
	StyleNamespace = tcell.StyleDefault.
			Foreground(colorNamespace).
			Background(colorBg)

	// Xray view — namespace section header
	StyleXrayNsHeader = tcell.StyleDefault.
				Foreground(tcell.NewRGBColor(80, 210, 200)).
				Background(tcell.NewRGBColor(12, 28, 32)).
				Bold(true)

	// Xray view — tree connector characters (├─ └─ │)
	StyleXrayConnector = tcell.StyleDefault.
				Foreground(tcell.NewRGBColor(55, 60, 85)).
				Background(colorBg)

	// Heatmap pod cells — fg color is applied to the ⬡ hexagon character.
	StyleHeatmapRunning = tcell.StyleDefault.
				Foreground(colorGreen).
				Background(colorBg)

	StyleHeatmapPending = tcell.StyleDefault.
				Foreground(colorAmber).
				Background(colorBg)

	StyleHeatmapFailed = tcell.StyleDefault.
				Foreground(colorRed).
				Background(colorBg).
				Bold(true)

	StyleHeatmapTerm = tcell.StyleDefault.
				Foreground(colorDim).
				Background(colorBg)

	StyleHeatmapDefault = tcell.StyleDefault.
				Foreground(colorBlue).
				Background(colorBg)

	// Selected node box — golden border so the active node stands out.
	StyleHeatmapNodeSel = tcell.StyleDefault.
				Foreground(tcell.NewRGBColor(255, 220, 50)).
				Background(colorBg).
				Bold(true)
)
