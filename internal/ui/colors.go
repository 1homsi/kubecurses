// Package ui contains rendering primitives and styles for the Bubble Tea UI.
package ui

import "github.com/charmbracelet/lipgloss"

// Dark theme palette — hex colour values so they look identical across terminals.
var (
	// Backgrounds
	colorBg          = lipgloss.Color("#101010") // near-black
	colorNodeBg      = lipgloss.Color("#161A2E") // node section header
	colorHeaderBg    = lipgloss.Color("#121626") // table column headers
	colorTabActiveBg = lipgloss.Color("#0064C8") // active tab
	colorTabIdleBg   = lipgloss.Color("#141624") // inactive tab
	colorStatusBg    = lipgloss.Color("#10121C") // status bar
	colorSelectedBg  = lipgloss.Color("#004494") // selected row

	// Foregrounds
	colorText      = lipgloss.Color("#DCDEEB") // default text
	colorDim       = lipgloss.Color("#646982") // dim / inactive
	colorGreen     = lipgloss.Color("#50C878") // Running / Ready
	colorAmber     = lipgloss.Color("#D2A032") // Pending / warn
	colorOrange    = lipgloss.Color("#DC7828") // restart warning
	colorRed       = lipgloss.Color("#D24141") // Failed / NotReady / high restarts
	colorBlue      = lipgloss.Color("#64AAFF") // version / metadata
	colorNodeName  = lipgloss.Color("#82BEFF") // node name — bright sky blue
	colorPodName   = lipgloss.Color("#C8D2FF") // pod name — light lavender white
	colorNamespace = lipgloss.Color("#6E78A0") // namespace — muted blue-gray
)

var (
	StyleDefault = lipgloss.NewStyle().
			Foreground(colorText).
			Background(colorBg)

	StyleHeader = lipgloss.NewStyle().
			Foreground(colorDim).
			Background(colorHeaderBg).
			Bold(true)

	StyleTabActive = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorTabActiveBg).
			Bold(true)

	StyleTabInactive = lipgloss.NewStyle().
				Foreground(colorDim).
				Background(colorTabIdleBg)

	StyleSelected = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorSelectedBg).
			Bold(true)

	StyleStatusBar = lipgloss.NewStyle().
			Foreground(colorDim).
			Background(colorStatusBg)

	StyleError = lipgloss.NewStyle().
			Foreground(colorRed).
			Background(colorBg).
			Bold(true)

	// Node section header row
	StyleNodeHeader = lipgloss.NewStyle().
			Foreground(colorText).
			Background(colorNodeBg).
			Bold(true)

	StyleNodeReadyDot = lipgloss.NewStyle().
				Foreground(colorGreen).
				Background(colorNodeBg).
				Bold(true)

	StyleNodeNotReadyDot = lipgloss.NewStyle().
				Foreground(colorRed).
				Background(colorNodeBg).
				Bold(true)

	StyleNodeMeta = lipgloss.NewStyle().
			Foreground(colorBlue).
			Background(colorNodeBg)

	// Pod status styles
	StylePodRunning = lipgloss.NewStyle().
			Foreground(colorGreen).
			Background(colorBg)

	StylePodPending = lipgloss.NewStyle().
			Foreground(colorAmber).
			Background(colorBg)

	StylePodFailed = lipgloss.NewStyle().
			Foreground(colorRed).
			Background(colorBg)

	StylePodDefault = lipgloss.NewStyle().
			Foreground(colorText).
			Background(colorBg)

	// Restart count styles (applied to the restart count cell only)
	StyleRestartsWarn = lipgloss.NewStyle().
				Foreground(colorOrange).
				Background(colorBg)

	StyleRestartsCrit = lipgloss.NewStyle().
				Foreground(colorRed).
				Background(colorBg).
				Bold(true)

	// Node status (used in flat nodes table)
	StyleNodeReady = lipgloss.NewStyle().
			Foreground(colorGreen).
			Background(colorBg)

	StyleNodeNotReady = lipgloss.NewStyle().
				Foreground(colorRed).
				Background(colorBg)

	StyleDim = lipgloss.NewStyle().
			Foreground(colorDim).
			Background(colorBg)

	StyleSeparator = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#2D324B")).
			Background(colorBg)

	// Warning banner row (scheduling imbalance, etc.)
	StyleWarning = lipgloss.NewStyle().
			Foreground(colorAmber).
			Background(lipgloss.Color("#1E1808")).
			Bold(true)

	// Pending reason sub-row
	StylePendingReason = lipgloss.NewStyle().
				Foreground(colorDim).
				Background(colorBg)

	// Metrics percentages — colour by severity
	StyleMetricsOK   = lipgloss.NewStyle().Foreground(colorGreen).Background(colorNodeBg)
	StyleMetricsWarn = lipgloss.NewStyle().Foreground(colorAmber).Background(colorNodeBg)
	StyleMetricsCrit = lipgloss.NewStyle().Foreground(colorRed).Background(colorNodeBg).Bold(true)

	// Node name — bright sky blue, stands out in node header rows
	StyleNodeName = lipgloss.NewStyle().
			Foreground(colorNodeName).
			Background(colorNodeBg).
			Bold(true)

	// Pod name — light lavender-white so it reads clearly on dark bg
	StylePodName = lipgloss.NewStyle().
			Foreground(colorPodName).
			Background(colorBg)

	// Namespace label — muted so name and status are visually dominant
	StyleNamespace = lipgloss.NewStyle().
			Foreground(colorNamespace).
			Background(colorBg)

	// Xray view — namespace section header
	StyleXrayNsHeader = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#50D2C8")).
				Background(lipgloss.Color("#0C1C20")).
				Bold(true)

	// Xray view — tree connector characters (├─ └─ │)
	StyleXrayConnector = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#373C55")).
				Background(colorBg)

	// Heatmap pod cells — fg color is applied to the ⬢ hexagon character.
	StyleHeatmapRunning = lipgloss.NewStyle().
				Foreground(colorGreen).
				Background(colorNodeBg)

	StyleHeatmapPending = lipgloss.NewStyle().
				Foreground(colorAmber).
				Background(colorNodeBg)

	StyleHeatmapFailed = lipgloss.NewStyle().
				Foreground(colorRed).
				Background(colorNodeBg).
				Bold(true)

	StyleHeatmapTerm = lipgloss.NewStyle().
				Foreground(colorDim).
				Background(colorNodeBg)

	StyleHeatmapDefault = lipgloss.NewStyle().
				Foreground(colorBlue).
				Background(colorNodeBg)

	StyleHeatmapBorder = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6F89C8")).
				Background(colorNodeBg)

	// Selected node box — amber background on perimeter cells so the ring glows.
	StyleHeatmapNodeSel = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F0BE42")).
				Background(colorNodeBg).
				Bold(true)
)
