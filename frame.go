package main

import (
	"strings"

	"charm.land/lipgloss/v2"
)

const (
	frameHPad   = 2 // left/right padding inside the outer border
	frameBorder = 2 // the rounded outer border, left + right cells
)

// renderFrame composes the single outer-bordered modal: padding gap, the ▓▓▓
// title, the rule, an inset gap, the body sections (joined by inset gaps),
// another inset gap, then the hint — all inside one rounded border whose color
// follows the variant. body sections and hint are already-styled strings. width
// is the full pane width; the rule spans it exactly so the border width == width.
func renderFrame(t Theme, variant, title string, body []string, hint string, width, padding, inset int) string {
	innerW := width - frameBorder - 2*frameHPad
	if innerW < 1 {
		innerW = 1
	}
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(t.titleColor(variant)))
	ruleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Rule))
	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.variantColor(variant)))

	rows := []string{}

	// Top border
	rows = append(rows, borderStyle.Render("╭"+strings.Repeat("─", width-2)+"╮"))

	// Padding rows (blank, just spaces to reach full width)
	blankRow := strings.Repeat(" ", width)
	for i := 0; i < padding; i++ {
		rows = append(rows, blankRow)
	}

	// Title
	contentRow := func(content string) string {
		padded := content + strings.Repeat(" ", innerW-lipgloss.Width(content))
		return borderStyle.Render("│") + strings.Repeat(" ", frameHPad) + padded + strings.Repeat(" ", frameHPad) + borderStyle.Render("│")
	}
	rows = append(rows, contentRow(titleStyle.Render("▓▓▓ " + title)))

	// Rule
	rows = append(rows, contentRow(ruleStyle.Render(strings.Repeat("━", innerW))))

	// Inset gap
	for i := 0; i < inset; i++ {
		rows = append(rows, blankRow)
	}

	// Body sections
	for i, sec := range body {
		if i > 0 {
			for j := 0; j < inset; j++ {
				rows = append(rows, blankRow)
			}
		}
		rows = append(rows, contentRow(sec))
	}

	// Inset gap
	for i := 0; i < inset; i++ {
		rows = append(rows, blankRow)
	}

	// Hint
	rows = append(rows, contentRow(hint))

	// Padding rows (blank, with borders and spacing to reach full width)
	for i := 0; i < padding; i++ {
		rows = append(rows, blankRow)
	}

	// Bottom border
	rows = append(rows, borderStyle.Render("╰"+strings.Repeat("─", width-2)+"╯"))

	return strings.Join(rows, "\n")
}

func appendBlanks(rows []string, n int) []string {
	for i := 0; i < n; i++ {
		rows = append(rows, "")
	}
	return rows
}
