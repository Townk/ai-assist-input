package main

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func key(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

func TestChooseSingleSelect(t *testing.T) {
	f := field(newChooseField(defaultTheme(), "default", []string{"alpha", "beta", "gamma"}, false, ""))
	f, _, _ = f.handle(key('j'))               // move to beta
	f2, act, _ := f.handle(tea.KeyPressMsg{Code: tea.KeyEnter})
	if act != fieldDone || f2.value() != "beta" {
		t.Fatalf("j then Enter must select beta: act=%d val=%q", act, f2.value())
	}
}

func TestChooseNumberShortcut(t *testing.T) {
	f := field(newChooseField(defaultTheme(), "default", []string{"alpha", "beta", "gamma"}, false, ""))
	f2, act, _ := f.handle(key('3'))
	if act != fieldDone || f2.value() != "gamma" {
		t.Fatalf("3 must select gamma: act=%d val=%q", act, f2.value())
	}
}

func TestChooseMultiToggle(t *testing.T) {
	f := field(newChooseField(defaultTheme(), "default", []string{"a", "b", "c"}, true, ""))
	f, _, _ = f.handle(tea.KeyPressMsg{Code: tea.KeySpace}) // toggle a
	f, _, _ = f.handle(key('j'))
	f, _, _ = f.handle(key('j'))
	f, _, _ = f.handle(tea.KeyPressMsg{Code: tea.KeySpace}) // toggle c
	f2, act, _ := f.handle(tea.KeyPressMsg{Code: tea.KeyEnter})
	if act != fieldDone {
		t.Fatalf("Enter must submit multi, act=%d", act)
	}
	if got := f2.value(); got != "a\nc" {
		t.Fatalf("multi value = %q, want \"a\\nc\"", got)
	}
}

func TestChooseRendersListNoFuzzy(t *testing.T) {
	f := field(newChooseField(defaultTheme(), "default", []string{"alpha", "beta"}, false, ""))
	out := strip(f.view(40, true))
	if !strings.Contains(out, "alpha") || !strings.Contains(out, "beta") {
		t.Fatal("options must render")
	}
	if !strings.Contains(out, "1") || !strings.Contains(out, "2") {
		t.Fatal("number shortcuts must render")
	}
}

func TestChooseOtherFreeText(t *testing.T) {
	f := field(newChooseField(defaultTheme(), "default", []string{"a", "b"}, false, "Other…"))
	// navigate to the trailing other entry (index 2) and select → enters text mode
	f, _, _ = f.handle(key('3'))
	// type a custom value
	for _, r := range "custom" {
		f, _, _ = f.handle(key(r))
	}
	f2, act, _ := f.handle(tea.KeyPressMsg{Code: tea.KeyEnter})
	if act != fieldDone || f2.value() != "custom" {
		t.Fatalf("other free-text must yield the typed value: act=%d val=%q", act, f2.value())
	}
}

func TestChooseEscCancel(t *testing.T) {
	f := field(newChooseField(defaultTheme(), "default", []string{"a"}, false, ""))
	_, act, _ := f.handle(tea.KeyPressMsg{Code: tea.KeyEscape})
	if act != fieldCancel {
		t.Fatal("Esc must cancel")
	}
}

// linesMatchView asserts that lines(innerW) equals the number of lines
// strip(view(innerW, true)) actually produces.
func linesMatchView(t *testing.T, f field, innerW int, label string) {
	t.Helper()
	rendered := strip(f.view(innerW, true))
	// count non-empty lines (view never emits trailing newlines, but be safe)
	viewLines := len(strings.Split(rendered, "\n"))
	got := f.lines(innerW)
	if got != viewLines {
		t.Errorf("%s: lines(%d)=%d but view renders %d lines\nrendered:\n%s",
			label, innerW, got, viewLines, rendered)
	}
}

func TestChooseLinesMatchViewShort(t *testing.T) {
	// Short list (≤ cap): no scroll indicators expected.
	opts := []string{"alpha", "beta", "gamma"}
	f := field(newChooseField(defaultTheme(), "default", opts, false, ""))
	linesMatchView(t, f, 40, "short list (3 options)")
}

func TestChooseLinesMatchViewLong(t *testing.T) {
	// Long list (> cap = 8): scroll indicators must be counted.
	opts := []string{"one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten"}
	f := field(newChooseField(defaultTheme(), "default", opts, false, ""))
	// Highlight is at 0, so viewStart=0 → no up-indicator; viewEnd=8 < 10 → down-indicator present.
	linesMatchView(t, f, 40, "long list (10 options, highlight=0)")

	// Move highlight to the middle so both indicators appear.
	for i := 0; i < 5; i++ {
		f, _, _ = f.handle(key('j'))
	}
	linesMatchView(t, f, 40, "long list (10 options, highlight=5, both indicators)")
}

func TestChooseOtherFilledWithActiveBuffer(t *testing.T) {
	// Activate other, type text, do NOT press Enter → value() is non-empty and filled() is true.
	// This covers the Tab-away-wedges-form bug: the form intercepts Tab before the field
	// commits, so otherText never gets set; filled() must look at the in-progress buffer.
	f := field(newChooseField(defaultTheme(), "default", []string{"a", "b"}, false, "Other…"))
	f, _, _ = f.handle(key('3')) // navigate to "other" and activate text mode
	for _, r := range "typed" {
		f, _, _ = f.handle(key(r))
	}
	// Do NOT send Enter — simulate Tab-away scenario.
	if f.value() == "" {
		t.Fatal("value() must return the in-progress buffer when other is active")
	}
	if !f.filled() {
		t.Fatal("filled() must return true when other is active with non-empty buffer")
	}
}

func TestChooseLinesMatchViewOtherActive(t *testing.T) {
	// When the "other" free-text entry is active, the embedded textField renders as
	// multiple physical lines; lines() must match the actual view() row count.
	f := field(newChooseField(defaultTheme(), "default", []string{"a", "b"}, false, "Other…"))
	f, _, _ = f.handle(key('3')) // activate other text mode
	linesMatchView(t, f, 40, "other-active (embedded textField rows must match lines())")
}

func TestChooseRowSpacingAndFullWidthHighlight(t *testing.T) {
	f := newChooseField(defaultTheme(), "default", []string{"alpha", "beta"}, false, "")
	// highlight is row 0 by default
	out := strip(f.view(30, true))
	first := strings.Split(out, "\n")[0]
	// 1 leading space before the number, 1 trailing space after the label, and
	// the highlighted row is padded to the inner width (full-width bar).
	if !strings.HasPrefix(first, " 1 alpha") {
		t.Fatalf("row must be ' <n> <label>' with a leading space: %q", first)
	}
	if lipgloss.Width(first) < 28 {
		t.Fatalf("highlighted row must span ~inner width, got width %d: %q", lipgloss.Width(first), first)
	}
}

func TestChooseLongOptionWraps(t *testing.T) {
	long := "this is a very long option label that must wrap onto a second visual line"
	f := newChooseField(defaultTheme(), "default", []string{long, "b"}, false, "")
	out := strip(f.view(24, true))
	lines := strings.Split(out, "\n")
	// the long option occupies >1 visual line, and the continuation is indented
	// under the label text (past the "  N " number column), not under the number.
	if len(lines) < 3 {
		t.Fatalf("long option must wrap to multiple lines: %q", out)
	}
	// continuation line starts with the label-column indent (spaces), not a digit
	cont := lines[1]
	if strings.TrimLeft(cont, " ") == cont {
		t.Fatalf("wrapped continuation must be indented under the label: %q", cont)
	}
}
