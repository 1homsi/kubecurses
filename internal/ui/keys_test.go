package ui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestEventToAction_ExecOpen(t *testing.T) {
	ev := tcell.NewEventKey(tcell.KeyRune, 'e', tcell.ModNone)
	if got := EventToAction(ev); got != ActionExecOpen {
		t.Errorf("EventToAction('e') = %v, want ActionExecOpen (%v)", got, ActionExecOpen)
	}
}

func TestEventToAction_LogsOpen(t *testing.T) {
	ev := tcell.NewEventKey(tcell.KeyRune, 'l', tcell.ModNone)
	if got := EventToAction(ev); got != ActionLogsOpen {
		t.Errorf("EventToAction('l') = %v, want ActionLogsOpen (%v)", got, ActionLogsOpen)
	}
}

func TestEventToAction_ExecAndLogsDistinct(t *testing.T) {
	if ActionExecOpen == ActionLogsOpen {
		t.Error("ActionExecOpen and ActionLogsOpen must be distinct constants")
	}
}
