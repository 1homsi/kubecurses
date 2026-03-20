package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestKeyToAction_ExecOpen(t *testing.T) {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}
	if got := KeyToAction(msg); got != ActionExecOpen {
		t.Errorf("KeyToAction('e') = %v, want ActionExecOpen (%v)", got, ActionExecOpen)
	}
}

func TestKeyToAction_LogsOpen(t *testing.T) {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}
	if got := KeyToAction(msg); got != ActionLogsOpen {
		t.Errorf("KeyToAction('l') = %v, want ActionLogsOpen (%v)", got, ActionLogsOpen)
	}
}

func TestKeyToAction_ExecAndLogsDistinct(t *testing.T) {
	if ActionExecOpen == ActionLogsOpen {
		t.Error("ActionExecOpen and ActionLogsOpen must be distinct constants")
	}
}
