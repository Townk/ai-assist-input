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
	// Number shortcuts are removed — pressing a digit must NOT select/jump.
	f := field(newChooseField(defaultTheme(), "default", []string{"alpha", "beta", "gamma"}, false, ""))
	_, act, _ := f.handle(key('3'))
	if act == fieldDone {
		t.Fatalf("number keys must no longer select (act=%d)", act)
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
	// Number prefixes are gone; radio indicators must appear instead.
	if !strings.Contains(out, "󰄵") && !strings.Contains(out, "󰄱") {
		t.Fatal("rows must show radio indicators (󰄵 or 󰄱)")
	}
}

func TestChooseOtherFreeText(t *testing.T) {
	f := field(newChooseField(defaultTheme(), "default", []string{"a", "b"}, false, "Other…"))
	// navigate to the trailing other entry using arrow-down (focus-to-type: no Enter to activate)
	f, _, _ = f.handle(tea.KeyPressMsg{Code: tea.KeyDown})
	f, _, _ = f.handle(tea.KeyPressMsg{Code: tea.KeyDown}) // now on the other row
	// type a custom value directly (no activate step needed)
	for _, r := range "custom" {
		f, _, _ = f.handle(key(r))
	}
	f2, act, _ := f.handle(tea.KeyPressMsg{Code: tea.KeyEnter})
	if act != fieldDone || f2.value() != "custom" {
		t.Fatalf("other free-text must yield the typed value: act=%d val=%q", act, f2.value())
	}
}

func TestChooseOtherFocusToType(t *testing.T) {
	f := field(newChooseField(defaultTheme(), "default", []string{"a", "b"}, false, "Other…"))
	// move highlight onto the "other" row (index 2 → key '3' navigates+… but we
	// want focus-to-type, so use arrow-down twice to land on it WITHOUT selecting)
	f, _, _ = f.handle(tea.KeyPressMsg{Code: tea.KeyDown})
	f, _, _ = f.handle(tea.KeyPressMsg{Code: tea.KeyDown}) // now on the other row
	// typing goes straight into the field (no Enter to activate)
	for _, r := range "custom" {
		f, _, _ = f.handle(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	// Enter submits the whole choose with the typed value
	f2, act, _ := f.handle(tea.KeyPressMsg{Code: tea.KeyEnter})
	if act != fieldDone || f2.value() != "custom" {
		t.Fatalf("focus-to-type other must submit typed value: act=%d val=%q", act, f2.value())
	}
}

func TestChooseOtherShiftEnterNewline(t *testing.T) {
	f := field(newChooseField(defaultTheme(), "default", []string{"a"}, false, "Other…"))
	f, _, _ = f.handle(tea.KeyPressMsg{Code: tea.KeyDown}) // onto other row
	f, _, _ = f.handle(tea.KeyPressMsg{Code: 'x', Text: "x"})
	f, _, _ = f.handle(tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModShift})
	if !strings.Contains(f.value(), "\n") {
		t.Fatalf("Shift+Enter in other must insert a newline: %q", f.value())
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
	// Focus the other row, type text, do NOT press Enter → value() is non-empty and filled() is true.
	// This covers the Tab-away-wedges-form bug: the form intercepts Tab before the field
	// commits, so otherText never gets set; filled() must look at the in-progress buffer.
	f := field(newChooseField(defaultTheme(), "default", []string{"a", "b"}, false, "Other…"))
	// navigate to "other" row via arrows (focus-to-type flow)
	f, _, _ = f.handle(tea.KeyPressMsg{Code: tea.KeyDown})
	f, _, _ = f.handle(tea.KeyPressMsg{Code: tea.KeyDown})
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
	// When the "other" row is highlighted (focus-to-type), the embedded textField
	// renders as multiple physical lines; lines() must match the actual view() row count.
	f := field(newChooseField(defaultTheme(), "default", []string{"a", "b"}, false, "Other…"))
	// navigate to "other" row via arrows (focus-to-type flow)
	f, _, _ = f.handle(tea.KeyPressMsg{Code: tea.KeyDown})
	f, _, _ = f.handle(tea.KeyPressMsg{Code: tea.KeyDown})
	linesMatchView(t, f, 40, "other-active (embedded textField rows must match lines())")
}

func TestChooseRowSpacingAndFullWidthHighlight(t *testing.T) {
	f := newChooseField(defaultTheme(), "default", []string{"alpha", "beta"}, false, "")
	// highlight is row 0 by default
	out := strip(f.view(30, true))
	first := strings.Split(out, "\n")[0]
	// Layout is now " <indicator> <label> " — leading space + indicator glyph + space.
	// The highlighted row must contain the label and be padded to the inner width.
	if !strings.Contains(first, "alpha") {
		t.Fatalf("row must contain the label 'alpha': %q", first)
	}
	if !strings.HasPrefix(first, " ") {
		t.Fatalf("row must start with a leading space: %q", first)
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

func TestChooseMultiSpaceAsRune(t *testing.T) {
	f := field(newChooseField(defaultTheme(), "default", []string{"a", "b", "c"}, true, ""))
	// a real space keypress arrives as Code=' ' (0x20), Text=" "
	f2, _, _ := f.handle(tea.KeyPressMsg{Code: ' ', Text: " "})
	if f2.value() != "a" {
		t.Fatalf("space (rune) must toggle the highlighted row; value=%q", f2.value())
	}
}

func TestChooseMultiSelectionsPreservedWhenEmptyOtherFocused(t *testing.T) {
	// Regression: toggling options then arrowing onto the (empty) other row and
	// pressing Enter must return the toggled options, NOT "".
	f := field(newChooseField(defaultTheme(), "default", []string{"a", "b", "c"}, true, "Other…"))
	// Toggle "a" (highlight=0)
	f, _, _ = f.handle(tea.KeyPressMsg{Code: tea.KeySpace})
	// Move to "c" and toggle it
	f, _, _ = f.handle(key('j'))
	f, _, _ = f.handle(key('j'))
	f, _, _ = f.handle(tea.KeyPressMsg{Code: tea.KeySpace})
	// Arrow down onto the "other" row (index 3), leaving its buffer empty
	f, _, _ = f.handle(tea.KeyPressMsg{Code: tea.KeyDown})
	// Press Enter — should submit with "a\nc", not ""
	f2, act, _ := f.handle(tea.KeyPressMsg{Code: tea.KeyEnter})
	if act != fieldDone {
		t.Fatalf("Enter must submit, act=%d", act)
	}
	if got := f2.value(); got != "a\nc" {
		t.Fatalf("multi value with empty other row = %q, want \"a\\nc\"", got)
	}
}

// I1: the worst-case (max) measured height of a choose with --other must
// account for the other row being expanded (embedded textField visible), not
// the 1-line collapsed placeholder.
func TestChooseOtherMaxLinesAccountsForExpandedRow(t *testing.T) {
	const innerW = 50
	// A choose with 2 options + other row.
	f := newChooseField(defaultTheme(), "default", []string{"a", "b"}, false, "Other…")

	// Collapsed (other row not focused): measure the collapsed line count.
	collapsedLines := f.lines(innerW)

	// Navigate onto the other row so it expands.
	fExpanded := field(newChooseField(defaultTheme(), "default", []string{"a", "b"}, false, "Other…"))
	fExpanded, _, _ = fExpanded.handle(tea.KeyPressMsg{Code: tea.KeyDown})
	fExpanded, _, _ = fExpanded.handle(tea.KeyPressMsg{Code: tea.KeyDown}) // on other row

	expandedLines := fExpanded.lines(innerW)

	if expandedLines <= collapsedLines {
		t.Fatalf("expanded other row must be taller than collapsed: collapsed=%d expanded=%d", collapsedLines, expandedLines)
	}

	// maxLines() on the original (not-yet-navigated) field must equal expandedLines —
	// it must report the worst-case height regardless of current highlight position.
	got := f.maxLines(innerW)
	if got != expandedLines {
		t.Fatalf("maxLines(%d)=%d want %d (expanded other row height)", innerW, got, expandedLines)
	}
}

func TestChooseHintRangeAndEscGlyph(t *testing.T) {
	h := chooseHint(defaultTheme(), 3 /*rows*/, false /*multi*/)
	plain := strip(h)
	// Number range is gone.
	if strings.Contains(plain, "1-3") || strings.Contains(plain, "pick") {
		t.Fatalf("hint must not mention a number range/pick: %q", plain)
	}
	// Must still have move and dismiss glyph.
	if !strings.Contains(plain, "move") {
		t.Fatalf("hint must still show 'move': %q", plain)
	}
	if !strings.Contains(plain, "󱊷") {
		t.Fatalf("hint must use the 󱊷 ESC glyph: %q", plain)
	}
	if strings.Contains(plain, "⎋") {
		t.Fatalf("hint must not use the ⎋ glyph: %q", plain)
	}
}

func TestChooseSingleShowsRadio(t *testing.T) {
	f := newChooseField(defaultTheme(), "default", []string{"alpha", "beta"}, false, "")
	// highlight is row 0; single-select radio: checked on highlighted row, empty elsewhere
	out := strip(f.view(30, true))
	lines := strings.Split(out, "\n")
	if !strings.Contains(lines[0], "󰄵") { // checked radio on the highlighted row
		t.Fatalf("highlighted single row must show the checked radio 󰄵: %q", lines[0])
	}
	if !strings.Contains(lines[1], "󰄱") { // empty radio on the other row
		t.Fatalf("non-highlighted single row must show the empty radio 󰄱: %q", lines[1])
	}
}

func TestChooseMultiShowsAndTogglesCheckbox(t *testing.T) {
	f := field(newChooseField(defaultTheme(), "default", []string{"a", "b", "c"}, true, ""))
	// initially all empty checkboxes
	if !strings.Contains(strip(f.view(30, true)), "󰄰") {
		t.Fatal("multi rows must show the empty checkbox 󰄰")
	}
	// space toggles the highlighted row → checked checkbox visible
	f2, _, _ := f.handle(tea.KeyPressMsg{Code: ' ', Text: " "})
	if !strings.Contains(strip(f2.view(30, true)), "󰄳") {
		t.Fatalf("after toggling, a checked checkbox 󰄳 must be visible: %q", strip(f2.view(30, true)))
	}
}

func TestChooseNoNumberShortcuts(t *testing.T) {
	f := field(newChooseField(defaultTheme(), "default", []string{"a", "b", "c"}, false, ""))
	// pressing "2" must NOT select/jump (numbers are gone) — value stays unset, no fieldDone
	_, act, _ := f.handle(tea.KeyPressMsg{Code: '2', Text: "2"})
	if act == fieldDone {
		t.Fatal("number keys must no longer select")
	}
	// the rendered rows must not contain a digit prefix
	out := strip(f.view(30, true))
	if strings.Contains(out, "1 ") || strings.Contains(out, "2 ") {
		t.Fatalf("rows must not show number prefixes: %q", out)
	}
}

func TestChooseHintNoNumberRange(t *testing.T) {
	h := strip(chooseHint(defaultTheme(), 3, false))
	if strings.Contains(h, "1-3") || strings.Contains(h, "pick") {
		t.Fatalf("hint must not mention a number range/pick: %q", h)
	}
	if !strings.Contains(h, "move") || !strings.Contains(h, "󱊷") {
		t.Fatalf("hint must still show move + 󱊷 dismiss: %q", h)
	}
}
