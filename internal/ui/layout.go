package ui

// Rect represents a rectangular region on screen.
type Rect struct {
	X, Y, W, H int
}

// TabBarHeight is the fixed height of the tab bar in rows.
const TabBarHeight = 1

// StatusBarHeight is the fixed height of the status bar in rows.
const StatusBarHeight = 1

// ContentRect returns the drawable content area given the full screen dimensions,
// accounting for tab bar at top and status bar at bottom.
func ContentRect(screenW, screenH int) Rect {
	return Rect{
		X: 0,
		Y: TabBarHeight,
		W: screenW,
		H: screenH - TabBarHeight - StatusBarHeight,
	}
}

// TabBarRect returns the tab bar rectangle.
func TabBarRect(screenW int) Rect {
	return Rect{X: 0, Y: 0, W: screenW, H: TabBarHeight}
}

// StatusBarRect returns the status bar rectangle.
func StatusBarRect(screenW, screenH int) Rect {
	return Rect{X: 0, Y: screenH - StatusBarHeight, W: screenW, H: StatusBarHeight}
}

// Clamp constrains v to [lo, hi].
func Clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
