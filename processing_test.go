package main

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestProcessingShowsSpinnerAndStatus(t *testing.T) {
	m := newProcessingModel(defaultTheme(), "ai-assist", 50)
	m2, _ := m.Update(statusMsg("Looking up docs"))
	out := strip(m2.(processingModel).View().Content) // rendered frame
	if !strings.Contains(out, "Looking up docs") {
		t.Fatalf("status label not shown: %q", out)
	}
	if !strings.Contains(out, "▓▓▓ ai-assist") {
		t.Fatalf("same framed title expected: %q", out)
	}
}

func TestProcessingQuitsOnClose(t *testing.T) {
	m := newProcessingModel(defaultTheme(), "ai-assist", 50)
	_, cmd := m.Update(closeMsg{})
	if cmd == nil {
		t.Fatal("close must return a quit cmd")
	}
	// Verify the cmd is actually a quit command by executing it
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("close cmd must produce tea.QuitMsg, got %T", msg)
	}
}
