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
	colorText    = tcell.NewRGBColor(220, 222, 235) // default text
	colorDim     = tcell.NewRGBColor(100, 105, 130) // dim / inactive
	colorGreen   = tcell.NewRGBColor(80, 200, 120)  // Running / Ready
	colorAmber   = tcell.NewRGBColor(210, 160, 50)  // Pending / warn
	colorOrange  = tcell.NewRGBColor(220, 120, 40)  // restart warning
	colorRed     = tcell.NewRGBColor(210, 65, 65)   // Failed / NotReady / high restarts
	colorBlue    = tcell.NewRGBColor(100, 170, 255) // version / metadata
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
)
