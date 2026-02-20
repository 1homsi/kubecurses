package ui

import "github.com/gdamore/tcell/v2"

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
	ActionSearchOpen        // '/' — enter search mode
	ActionSearchCancel      // Esc — clear and exit search mode
	ActionSearchCommit      // Enter — commit query and exit search mode
	ActionSearchBack        // Backspace — delete last char in query
	ActionSearchAppend      // any printable rune while in search mode
	ActionHelp              // '?' — toggle help overlay
	ActionLogsOpen          // 'l' — open logs for selected pod/container
	ActionLogsToggleScroll  // 's' — toggle autoscroll in logs view
	ActionSwitchCluster     // 'c' — open the cluster/context picker
	ActionConfirm           // Enter — confirm selection in overlays
	ActionTab5              // '5' — jump to Heatmap tab
)

// EventToAction converts a tcell event into a logical Action for normal mode.
// Returns ActionNone for unrecognised events.
func EventToAction(ev tcell.Event) Action {
	evKey, ok := ev.(*tcell.EventKey)
	if !ok {
		return ActionNone
	}

	if evKey.Key() == tcell.KeyCtrlC {
		return ActionQuit
	}

	switch evKey.Key() {
	case tcell.KeyTab:
		return ActionNextTab
	case tcell.KeyBacktab:
		return ActionPrevTab
	case tcell.KeyUp:
		return ActionMoveUp
	case tcell.KeyDown:
		return ActionMoveDown
	case tcell.KeyLeft:
		return ActionMoveLeft
	case tcell.KeyRight:
		return ActionMoveRight
	case tcell.KeyPgUp:
		return ActionPageUp
	case tcell.KeyPgDn:
		return ActionPageDown
	case tcell.KeyEsc:
		return ActionSearchCancel
	}

	switch evKey.Key() {
	case tcell.KeyEnter:
		return ActionConfirm
	}

	switch evKey.Rune() {
	case 'q', 'Q':
		return ActionQuit
	case 'j':
		return ActionMoveDown
	case 'k':
		return ActionMoveUp
	case 'h':
		return ActionMoveLeft
	case 'l':
		return ActionLogsOpen
	case 's':
		return ActionLogsToggleScroll
	case 'c':
		return ActionSwitchCluster
	case 'r':
		return ActionRefresh
	case '/':
		return ActionSearchOpen
	case '?':
		return ActionHelp
	case '1':
		return ActionTab1
	case '2':
		return ActionTab2
	case '3':
		return ActionTab3
	case '4':
		return ActionTab4
	case '5':
		return ActionTab5
	}

	return ActionNone
}

// SearchEventToAction converts a tcell event while in search mode.
func SearchEventToAction(ev tcell.Event) Action {
	evKey, ok := ev.(*tcell.EventKey)
	if !ok {
		return ActionNone
	}

	if evKey.Key() == tcell.KeyCtrlC {
		return ActionQuit
	}

	switch evKey.Key() {
	case tcell.KeyEsc:
		return ActionSearchCancel
	case tcell.KeyEnter:
		return ActionSearchCommit
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		return ActionSearchBack
	}

	if evKey.Rune() != 0 {
		return ActionSearchAppend
	}

	return ActionNone
}
