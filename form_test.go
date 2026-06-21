package main

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

const us, rs, gs = "\x1f", "\x1e", "\x1d"

func TestParseFormSpec(t *testing.T) {
	raw := "name" + us + "line" + us + "Your name" + us + "" + rs +
		"plan" + us + "choose" + us + "Plan" + us + "free" + gs + "pro"
	ff, err := parseFormSpec(raw)
	if err != nil || len(ff) != 2 {
		t.Fatalf("parse: %v len=%d", err, len(ff))
	}
	if ff[0].name != "name" || ff[0].ftype != "line" || ff[1].ftype != "choose" {
		t.Fatalf("bad parse: %+v", ff)
	}
	if ff[1].param != "free"+gs+"pro" {
		t.Fatalf("choose param: %q", ff[1].param)
	}
}

func TestFormTabCyclesFields(t *testing.T) {
	m := newFormModel(defaultTheme(), "Setup", []formField{
		{"a", "line", "First", ""}, {"b", "line", "Second", ""},
	}, 1, 1)
	if m.focus != 0 {
		t.Fatal("starts on field 0")
	}
	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if m2.(formModel).focus != 1 {
		t.Fatal("Tab moves to field 1")
	}
}

func TestFormRequiredNoEarlySubmit(t *testing.T) {
	// Enter on an empty required form must NOT submit; it jumps to next unfilled.
	m := newFormModel(defaultTheme(), "Setup", []formField{
		{"a", "line", "First", ""}, {"b", "line", "Second", ""},
	}, 1, 1)
	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m2.(formModel).submitted {
		t.Fatal("must not submit with empty fields")
	}
}

func TestFormRendersTabRowAndActiveFieldNoNestedTitle(t *testing.T) {
	m := newFormModel(defaultTheme(), "Setup", []formField{
		{"a", "line", "First", ""}, {"b", "confirm", "Agree?", ""},
	}, 1, 1)
	m.width = 50
	out := strip(m.render())
	if !strings.Contains(out, "▓▓▓ Setup") {
		t.Fatal("main title missing")
	}
	if !strings.Contains(out, "First") || !strings.Contains(out, "Agree?") {
		t.Fatal("tab row labels missing")
	}
	if strings.Count(out, "▓▓▓") != 1 {
		t.Fatal("must have exactly one ▓▓▓ (no nested per-field titles)")
	}
}

func TestFormOutputProtocol(t *testing.T) {
	answers := []string{"a" + us + "Alice", "b" + us + "yes"}
	if got := strings.Join(answers, rs); got != "a"+us+"Alice"+rs+"b"+us+"yes" {
		t.Fatal("output join")
	}
}
