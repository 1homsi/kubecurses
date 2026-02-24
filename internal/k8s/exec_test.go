package k8s

import "testing"

func TestInteractiveExecCommand(t *testing.T) {
	cmd := InteractiveExecCommand()
	if len(cmd) == 0 {
		t.Fatal("InteractiveExecCommand returned empty slice")
	}
	if cmd[0] != "/bin/sh" {
		t.Errorf("expected /bin/sh as first element, got %q", cmd[0])
	}
}
