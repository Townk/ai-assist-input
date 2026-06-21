package main

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestRenderFrameStructure(t *testing.T) {
	out := renderFrame(defaultTheme(), "default", "Quit Session",
		[]string{"Body line"}, "hint here", 40, 1, 1)
	lines := strings.Split(strip(out), "\n")
	if !strings.HasPrefix(lines[0], "╭") || !strings.HasSuffix(lines[0], "╮") {
		t.Fatalf("first line must be the rounded border top, got %q", lines[0])
	}
	if last := lines[len(lines)-1]; !strings.HasPrefix(last, "╰") {
		t.Fatalf("last line must be the rounded border bottom, got %q", last)
	}
	plain := strip(out)
	for _, want := range []string{"▓▓▓ Quit Session", "━", "Body line", "hint here"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("frame missing %q", want)
		}
	}
}

func TestRenderFrameWidth(t *testing.T) {
	out := renderFrame(defaultTheme(), "default", "T", []string{"x"}, "h", 40, 1, 1)
	for i, l := range strings.Split(out, "\n") {
		if w := lipgloss.Width(l); w != 40 {
			t.Fatalf("line %d width = %d, want 40: %q", i, w, strip(l))
		}
	}
}

func TestRenderFramePaddingRows(t *testing.T) {
	// padding=2 adds two blank rows just inside the top and bottom borders.
	// Each blank padding row must still be enclosed by the continuous border:
	// it starts with │, ends with │, and everything in between is spaces.
	out := strip(renderFrame(defaultTheme(), "default", "T", []string{"B"}, "h", 30, 2, 1))
	lines := strings.Split(out, "\n")
	// line[0]=border top, line[1]/line[2]=padding blanks, line[3]=title
	for _, idx := range []int{1, 2} {
		l := lines[idx]
		if !strings.HasPrefix(l, "│") || !strings.HasSuffix(l, "│") {
			t.Fatalf("padding row %d must be enclosed by │...│, got %q", idx, l)
		}
		inner := l[len("│") : len(l)-len("│")]
		if strings.TrimSpace(inner) != "" {
			t.Fatalf("padding row %d inner must be all spaces, got %q", idx, inner)
		}
	}
	if !strings.Contains(lines[3], "▓▓▓ T") {
		t.Fatalf("title must follow the padding rows, got %q", lines[3])
	}
}
