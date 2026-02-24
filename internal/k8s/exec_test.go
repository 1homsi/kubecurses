package k8s

import "testing"

func TestDefaultExecCommand(t *testing.T) {
	cmd := DefaultExecCommand()
	if len(cmd) == 0 {
		t.Fatal("DefaultExecCommand returned empty slice")
	}
	if cmd[0] != "/bin/sh" {
		t.Errorf("expected /bin/sh as first element, got %q", cmd[0])
	}
	// Must have at least a shell flag and a command string.
	if len(cmd) < 3 {
		t.Errorf("expected at least 3 elements, got %d: %v", len(cmd), cmd)
	}
}
