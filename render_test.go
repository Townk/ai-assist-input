package main

import (
	"regexp"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

var ansiRE = regexp.MustCompile("\x1b\\[[0-9;]*m")

func strip(s string) string { return ansiRE.ReplaceAllString(s, "") }

// TestRenderLayout pins the modal frame: title (no leading blank) → rule → boxed
// input with the prompt icon → hint, in that order.
func TestRenderLayout(t *testing.T) {
	m := initialModel("hello world", "ai-assist", 5)
	m.width = 60
	m.resize()
	lines := strings.Split(strip(m.render()), "\n")

	if !strings.HasPrefix(lines[0], "  ▓▓▓ ai-assist") {
		t.Fatalf("first line must be the title (no leading blank), got %q", lines[0])
	}
	if !strings.Contains(lines[1], "━") {
		t.Fatalf("second line must be the rule, got %q", lines[1])
	}
	out := strings.Join(lines, "\n")
	if !strings.Contains(out, promptIcon) {
		t.Fatal("prompt icon missing from the modal")
	}
	if !strings.Contains(out, "╭") || !strings.Contains(out, "╰") {
		t.Fatal("input box border missing")
	}
	last := lines[len(lines)-1]
	if !strings.Contains(last, "󰌑 : submit") || !strings.Contains(last, "󱊷 : cancel") {
		t.Fatalf("hint line wrong: %q", last)
	}
}

// TestPopupInputArea pins the chrome budget: ai-assist-popup opens a 55-col
// float (→ 53-col content area), and the input area must come out 40x3.
func TestPopupInputArea(t *testing.T) {
	m := initialModel("", "ai-assist", 3)
	m.width = 53 // 55-col float minus the pane border
	m.resize()
	if w, h := m.textarea.Width(), m.textarea.Height(); w != 40 || h != 3 {
		t.Fatalf("input area = %dx%d, want 40x3 (chrome assumes a 55-col float)", w, h)
	}
}

// TestScrollbarWrappedLines verifies the scrollbar reflects soft-wrapped rows,
// not just logical lines: one long line (no newlines) that wraps past the
// viewport must still show a thumb.
func TestScrollbarWrappedLines(t *testing.T) {
	m := initialModel(strings.Repeat("x", 200), "ai-assist", 3) // 1 logical line → ~5 wrapped rows at width 40
	m.width = 53
	m.resize()
	if vc := visualLineCount(m); vc <= m.textarea.Height() {
		t.Fatalf("visualLineCount = %d, want > %d (content wraps past the viewport)", vc, m.textarea.Height())
	}
	if m.textarea.LineCount() != 1 {
		t.Fatalf("precondition: expected 1 logical line, got %d", m.textarea.LineCount())
	}
	if sb := scrollbar(m); !strings.Contains(sb, "┃") {
		t.Fatalf("scrollbar should show a thumb for wrapped content, got %q", strip(sb))
	}
}

// TestRenderFitsPane verifies no rendered line exceeds the pane width.
func TestRenderFitsPane(t *testing.T) {
	m := initialModel("a long enough value to exercise wrapping across the textarea width", "ai-assist", 4)
	m.width = 50
	m.resize()
	for i, l := range strings.Split(m.render(), "\n") {
		if w := lipgloss.Width(l); w > m.width {
			t.Fatalf("line %d width %d exceeds pane width %d: %q", i, w, m.width, strip(l))
		}
	}
}
