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

// Problem 3 (unfocused): the other box renders entirely in the muted colour and
// shows no placeholder when empty.
func TestChooseOtherUnfocusedIsMutedNoPlaceholder(t *testing.T) {
	f := newChooseField(defaultTheme(), "default", []string{"a", "b"}, false, "Other…")
	out := f.view(40, true) // highlight row 0 → other row (idx 2) unfocused + empty
	if strings.Contains(strip(out), "Other…") {
		t.Fatalf("unfocused empty other must NOT show the placeholder: %q", strip(out))
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

// Problem 3 (focused): the other box uses the selected background and bright
// white for everything (border included), and shows the placeholder.
func TestChooseOtherFocusedSelBgBrightWhite(t *testing.T) {
	g, _, _ := field(newChooseField(defaultTheme(), "default", []string{"a", "b"}, false, "Other…")).
		handle(tea.KeyPressMsg{Code: tea.KeyDown})
	g, _, _ = g.handle(tea.KeyPressMsg{Code: tea.KeyDown}) // onto the other row (idx 2)
	out := g.view(40, true)
	bl := topBorderLine(t, out)
	if !strings.Contains(bl, fgWhite) {
		t.Fatalf("focused other border must be bright white %q: %q", fgWhite, bl)
	}
	if !strings.Contains(bl, bgSel) {
		t.Fatalf("focused other box must use the selected background %q: %q", bgSel, bl)
	}
	if !strings.Contains(strip(out), "Other…") {
		t.Fatalf("focused empty other should show the placeholder: %q", strip(out))
	}
}
