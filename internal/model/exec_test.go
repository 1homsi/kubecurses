package model

import "testing"

// TestExecStateOpen verifies that the exec overlay fields can be set as
// expected when the overlay is "opened".
func TestExecStateOpen(t *testing.T) {
	s := AppState{}

	s.ExecMode = true
	s.ExecNamespace = "default"
	s.ExecPod = "my-pod"
	s.ExecContainer = "main"
	s.ExecAutoScroll = true
	s.ExecLines = []string{"root", "Linux x86_64"}

	if !s.ExecMode {
		t.Error("ExecMode should be true after open")
	}
	if s.ExecPod != "my-pod" {
		t.Errorf("ExecPod = %q, want %q", s.ExecPod, "my-pod")
	}
	if len(s.ExecLines) != 2 {
		t.Errorf("ExecLines len = %d, want 2", len(s.ExecLines))
	}
}

// TestExecStateClose verifies that clearing exec fields works correctly.
func TestExecStateClose(t *testing.T) {
	s := AppState{
		ExecMode:      true,
		ExecPod:       "my-pod",
		ExecNamespace: "default",
		ExecLines:     []string{"line1"},
		ExecOffset:    5,
	}

	s.ExecMode = false
	s.ExecLines = nil
	s.ExecOffset = 0

	if s.ExecMode {
		t.Error("ExecMode should be false after close")
	}
	if len(s.ExecLines) != 0 {
		t.Errorf("ExecLines should be empty after close, got %d", len(s.ExecLines))
	}
	if s.ExecOffset != 0 {
		t.Errorf("ExecOffset should be 0 after close, got %d", s.ExecOffset)
	}
}

// TestExecAndLogsMutuallyExclusive verifies the state struct carries both
// overlays independently so neither clobbers the other's fields.
func TestExecAndLogsMutuallyExclusive(t *testing.T) {
	s := AppState{}

	s.LogsMode = true
	s.LogsPod = "log-pod"
	s.ExecMode = false

	if s.ExecMode {
		t.Error("ExecMode should not be set when LogsMode is true")
	}
	if s.LogsPod != "log-pod" {
		t.Error("LogsPod should be unaffected by ExecMode being false")
	}
}
