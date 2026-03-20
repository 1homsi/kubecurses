package ui

import tea "github.com/charmbracelet/bubbletea"

// Action represents a logical user action, decoupled from the physical key.
type Action int

const (
	ActionNone Action = iota
	ActionQuit
	ActionNextTab
	ActionPrevTab
	ActionTab1
	ActionTab2
	ActionTab3
	ActionTab4
	ActionMoveUp
	ActionMoveDown
	ActionMoveLeft
	ActionMoveRight
	ActionPageUp
	ActionPageDown
	ActionRefresh
	ActionSearchOpen
	ActionSearchCancel
	ActionSearchCommit
	ActionSearchBack
	ActionSearchAppend
	ActionHelp
	ActionLogsOpen
	ActionLogsToggleScroll
	ActionSwitchCluster
	ActionConfirm
	ActionTab5
	ActionExecOpen
	ActionTab6
	ActionSwitchNamespace
	ActionDescribe
)

// KeyToAction converts a tea.KeyMsg into a logical Action for normal mode.
// Returns ActionNone for unrecognised keys.
func KeyToAction(msg tea.KeyMsg) Action {
	switch msg.Type {
	case tea.KeyCtrlC:
		return ActionQuit
	case tea.KeyTab:
		return ActionNextTab
	case tea.KeyShiftTab:
		return ActionPrevTab
	case tea.KeyUp:
		return ActionMoveUp
	case tea.KeyDown:
		return ActionMoveDown
	case tea.KeyLeft:
		return ActionMoveLeft
	case tea.KeyRight:
		return ActionMoveRight
	case tea.KeyPgUp:
		return ActionPageUp
	case tea.KeyPgDown:
		return ActionPageDown
	case tea.KeyEscape:
		return ActionSearchCancel
	case tea.KeyEnter:
		return ActionConfirm
	}

	switch msg.String() {
	case "q", "Q":
		return ActionQuit
	case "j":
		return ActionMoveDown
	case "k":
		return ActionMoveUp
	case "h":
		return ActionMoveLeft
	case "l":
		return ActionLogsOpen
	case "s":
		return ActionLogsToggleScroll
	case "c":
		return ActionSwitchCluster
	case "r":
		return ActionRefresh
	case "/":
		return ActionSearchOpen
	case "?":
		return ActionHelp
	case "1":
		return ActionTab1
	case "2":
		return ActionTab2
	case "3":
		return ActionTab3
	case "4":
		return ActionTab4
	case "5":
		return ActionTab5
	case "6":
		return ActionTab6
	case "e":
		return ActionExecOpen
	case "n":
		return ActionSwitchNamespace
	case "d":
		return ActionDescribe
	}

	return ActionNone
}

// SearchKeyToAction converts a tea.KeyMsg while in search mode.
func SearchKeyToAction(msg tea.KeyMsg) Action {
	switch msg.Type {
	case tea.KeyCtrlC:
		return ActionQuit
	case tea.KeyEscape:
		return ActionSearchCancel
	case tea.KeyEnter:
		return ActionSearchCommit
	case tea.KeyBackspace:
		return ActionSearchBack
	}

	if len(msg.Runes) > 0 {
		return ActionSearchAppend
	}

	return ActionNone
}
