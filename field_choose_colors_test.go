package main

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// Truecolor ANSI fragments for the default theme.
const (
	fgMuted       = "38;2;108;112;134" // theme.Muted (unselected-tab fg)
	fgText        = "38;2;205;214;244" // theme.Text (the old, too-white option fg)
	fgWhite       = "38;2;255;255;255" // ButtonSelFg (bright white)
	bgSel         = "48;2;101;106;131" // ButtonSelBg (selected background)
	fgFieldBorder = "38;2;88;91;112"   // theme.FieldBorder (default box border)
)

func topBorderLine(t *testing.T, out string) string {
	t.Helper()
	for _, ln := range strings.Split(out, "\n") {
		if strings.Contains(ln, "╭") {
			return ln
		}
	}
	t.Fatalf("no box top border (╭) found in: %q", out)
	return ""
}

// Problem 2: non-highlighted rows must use the muted (unselected-tab) fg, not
// the near-white text fg that made them indistinguishable from the selection.
func TestChooseNonHighlightedRowUsesMutedFg(t *testing.T) {
	f := newChooseField(defaultTheme(), "default", []string{"alpha", "beta"}, false, "")
	out := f.view(40, true) // highlight is on row 0; "beta" is non-highlighted
	if !strings.Contains(out, fgMuted) {
		t.Fatalf("non-highlighted rows must use the muted fg %q: %q", fgMuted, out)
	}
	if strings.Contains(out, fgText) {
		t.Fatalf("option rows must NOT use the near-white text fg %q: %q", fgText, out)
	}
}

// Problem 3 (unfocused): the other entry shows its label above the box, and the
// whole widget (label + border) renders in the muted colour with no background.
func TestChooseOtherUnfocusedIsMuted(t *testing.T) {
	f := newChooseField(defaultTheme(), "default", []string{"a", "b"}, false, "Other…")
	out := f.view(40, true) // highlight row 0 → other row (idx 2) unfocused + empty
	if !strings.Contains(strip(out), "Other…:") {
		t.Fatalf("other label must render as a heading above the box: %q", strip(out))
	}
	bl := topBorderLine(t, out)
	if strings.Contains(bl, fgFieldBorder) {
		t.Fatalf("unfocused other border must NOT use the default field-border colour: %q", bl)
	}
	if !strings.Contains(bl, fgMuted) {
		t.Fatalf("unfocused other border must use the muted fg %q: %q", fgMuted, bl)
	}
	if strings.Contains(bl, bgSel) {
		t.Fatalf("unfocused other must NOT have the selected background: %q", bl)
	}
}

// Problem 3 (focused): the other entry uses the selected background and bright
// white for everything (label + border + icon), and the background fills EVERY
// line of the box — including the icon row and the empty rows (no gaps).
func TestChooseOtherFocusedSelBgBrightWhite(t *testing.T) {
	g, _, _ := field(newChooseField(defaultTheme(), "default", []string{"a", "b"}, false, "Other…")).
		handle(tea.KeyPressMsg{Code: tea.KeyDown})
	g, _, _ = g.handle(tea.KeyPressMsg{Code: tea.KeyDown}) // onto the other row (idx 2)
	out := g.view(40, true)

	// The label heading renders bright white on the selected background.
	if !strings.Contains(strip(out), "Other…:") {
		t.Fatalf("focused other should show its label heading: %q", strip(out))
	}
	bl := topBorderLine(t, out)
	if !strings.Contains(bl, fgWhite) {
		t.Fatalf("focused other border must be bright white %q: %q", fgWhite, bl)
	}

	// Every line from the label through the box bottom must carry the selected
	// background — no cell left with the default background (Problem 2/3).
	lines := strings.Split(out, "\n")
	start := -1
	for i, ln := range lines {
		if strings.Contains(ln, "Other…") {
			start = i
			break
		}
	}
	if start < 0 || start+4 >= len(lines) {
		t.Fatalf("expected a 5-line focused other item: %q", out)
	}
	for off := 0; off <= 4; off++ { // label, top, icon row, empty row, bottom
		if !strings.Contains(lines[start+off], bgSel) {
			t.Fatalf("focused other line %d must be backed by the selected bg %q: %q",
				off, bgSel, lines[start+off])
		}
	}
}
