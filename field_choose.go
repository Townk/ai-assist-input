package main

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const maxVisibleRows = 8

// chooseField implements the field interface for a themed list (no fuzzy).
// It supports single- and multi-select, 1–9 shortcuts, windowed scroll, and
// an optional free-text "other" entry at the end.
type chooseField struct {
	theme      Theme
	variant    string
	options    []string // base options (excluding the "other" row)
	multi      bool
	otherLabel string // "" → no other row

	highlight int    // currently highlighted row index (0-based, over all rows)
	selected  int    // single mode: selected option index (-1 = none)
	toggled   []bool // multi mode: toggled[i] for options[i]
	// otherField is lazily created when the highlight first lands on the other row.
	// It is implicitly active whenever isOtherRow(highlight) is true.
	otherField *textField
}

// newChooseField constructs a chooseField. other=="" → no free-text entry.
func newChooseField(theme Theme, variant string, options []string, multi bool, other string) *chooseField {
	toggled := make([]bool, len(options))
	return &chooseField{
		theme:      theme,
		variant:    variant,
		options:    options,
		multi:      multi,
		otherLabel: other,
		highlight:  0,
		selected:   -1,
		toggled:    toggled,
	}
}

// totalRows returns the total number of visible rows (options + optional other).
func (f *chooseField) totalRows() int {
	n := len(f.options)
	if f.otherLabel != "" {
		n++
	}
	return n
}

// isOtherRow returns true if idx points to the trailing "other" row.
func (f *chooseField) isOtherRow(idx int) bool {
	return f.otherLabel != "" && idx == len(f.options)
}

// handle processes one message while the field is focused.
func (f *chooseField) handle(msg tea.Msg) (field, fieldAction, tea.Cmd) {
	kp, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return f, fieldNone, nil
	}

	c := *f
	total := c.totalRows()

	// When the highlight is on the other row, the embedded textField is implicitly
	// active. Route keys to it EXCEPT for navigation/submit/cancel keys that belong
	// to the choose layer.
	if c.isOtherRow(c.highlight) {
		// Lazily initialise the other field the first time we land on the row.
		if c.otherField == nil {
			c.otherField = newTextField(c.theme, "", c.otherLabel, 1, false)
		}

		switch {
		case kp.Code == tea.KeyEscape:
			return &c, fieldCancel, nil

		case kp.String() == "ctrl+c":
			return &c, fieldCancel, nil

		// Arrow keys leave the other row (navigate the list); they do NOT type.
		case kp.Code == tea.KeyUp:
			if c.highlight > 0 {
				c.highlight--
			}
			return &c, fieldNone, nil

		case kp.Code == tea.KeyDown:
			if c.highlight < total-1 {
				c.highlight++
			}
			return &c, fieldNone, nil

		// Shift+Enter → newline into the embedded field.
		case kp.Code == tea.KeyEnter && kp.Mod.Contains(tea.ModShift):
			f2, _, cmd := c.otherField.handle(msg)
			c.otherField = f2.(*textField)
			return &c, fieldNone, cmd

		// Plain Enter → submit the whole choose with the field's current value.
		case kp.Code == tea.KeyEnter:
			return &c, fieldDone, nil

		default:
			// Forward everything else (printable chars, Backspace, etc.) to the field.
			f2, _, cmd := c.otherField.handle(msg)
			c.otherField = f2.(*textField)
			return &c, fieldNone, cmd
		}
	}

	// Highlight is NOT the other row — original navigation/selection behavior.
	switch {
	case kp.Code == tea.KeyEscape:
		return f, fieldCancel, nil

	case kp.String() == "ctrl+c":
		return f, fieldCancel, nil

	case kp.Code == 'j' || kp.Code == tea.KeyDown:
		if c.highlight < total-1 {
			c.highlight++
			// Lazily initialise the other field when we first land on the other row.
			if c.isOtherRow(c.highlight) && c.otherField == nil {
				c.otherField = newTextField(c.theme, "", c.otherLabel, 1, false)
			}
		}
		return &c, fieldNone, nil

	case kp.Code == 'k' || kp.Code == tea.KeyUp:
		if c.highlight > 0 {
			c.highlight--
		}
		return &c, fieldNone, nil

	case kp.Code >= '1' && kp.Code <= '9':
		n := int(kp.Code-'0') - 1 // zero-based index
		if n >= total {
			return f, fieldNone, nil
		}
		c.highlight = n
		if c.isOtherRow(n) {
			// Focus the other row (focus-to-type; lazily init the field).
			if c.otherField == nil {
				c.otherField = newTextField(c.theme, "", c.otherLabel, 1, false)
			}
			return &c, fieldNone, nil
		}
		if c.multi {
			c.toggled[n] = !c.toggled[n]
			return &c, fieldNone, nil
		}
		// Single: select and done.
		c.selected = n
		return &c, fieldDone, nil

	case (kp.Code == tea.KeySpace || kp.Code == ' ') && c.multi:
		if !c.isOtherRow(c.highlight) {
			c.toggled[c.highlight] = !c.toggled[c.highlight]
		}
		return &c, fieldNone, nil

	case kp.Code == tea.KeyEnter:
		if c.multi {
			return &c, fieldDone, nil
		}
		// Single: select highlighted and done.
		c.selected = c.highlight
		return &c, fieldDone, nil
	}

	return f, fieldNone, nil
}

// windowBounds returns the viewport slice [start, end) for the visible window,
// and whether up/down scroll indicators should be shown.
// This is the single source of truth used by both view() and lines().
func (f *chooseField) windowBounds() (viewStart, viewEnd int, showUp, showDown bool) {
	total := f.totalRows()
	maxVis := maxVisibleRows

	viewStart = 0
	viewEnd = total
	if total > maxVis {
		viewStart = f.highlight - maxVis/2
		if viewStart < 0 {
			viewStart = 0
		}
		viewEnd = viewStart + maxVis
		if viewEnd > total {
			viewEnd = total
			viewStart = viewEnd - maxVis
			if viewStart < 0 {
				viewStart = 0
			}
		}
		showUp = viewStart > 0
		showDown = viewEnd < total
	}
	return
}

// wrapLabel wraps labelText into lines of at most colW visible characters.
// It returns one or more strings; if the text fits in colW, it returns a
// single-element slice.  Wrapping is done at word boundaries (spaces); long
// words that exceed colW are broken at the column boundary.
func wrapLabel(labelText string, colW int) []string {
	if colW <= 0 {
		return []string{labelText}
	}
	wrapped := lipgloss.Wrap(labelText, colW, " ")
	return strings.Split(wrapped, "\n")
}

// renderOptionRow builds the visual lines for a single list option (not the
// "other" row).  It returns the lines that should be appended to rows.
//
// Layout per first visual line:
//
//	" " + num + " " + [marker] + text
//
// where the trailing " " is included in the padded width for highlighted rows.
// Continuation lines are indented to align under the label text (past the
// " num " prefix and, for multi mode, past the "● " / "○ " marker).
func (f *chooseField) renderOptionRow(
	i, innerW int,
	isHL bool,
	hlStyle, normStyle, mutedStyle, markerSelStyle lipgloss.Style,
) []string {
	num := fmt.Sprintf("%d", i+1)
	// Prefix: " " + num + " "  (e.g., " 1 " = 3 chars for single-digit)
	prefixLen := 1 + len(num) + 1
	// Trailing space is accounted for in innerW padding for highlights, or
	// appended to non-highlighted lines.
	trailingLen := 1

	opt := f.options[i]

	// Marker for multi-select (2 visible chars: "● " or "○ ").
	const markerLen = 2
	var markerPlain string // plain-text version for width calculation
	if f.multi {
		if f.toggled[i] {
			markerPlain = "● "
		} else {
			markerPlain = "○ "
		}
	}

	// Width available for the label text (including marker).
	labelColW := innerW - prefixLen - trailingLen
	if labelColW < 1 {
		labelColW = 1
	}
	// Width available for the option text itself (after marker).
	textColW := labelColW
	if f.multi {
		textColW = labelColW - markerLen
		if textColW < 1 {
			textColW = 1
		}
	}

	// Wrap the option text into textColW-wide chunks.
	textLines := wrapLabel(opt, textColW)

	// Indentation for continuation lines (aligns under label text after marker).
	contIndent := strings.Repeat(" ", prefixLen+len(markerPlain))

	prefix := " " + num + " "

	var resultLines []string
	for li, tl := range textLines {
		var lineText string
		if li == 0 {
			// First visual line: prefix + marker + text.
			lineText = prefix + markerPlain + tl
		} else {
			// Continuation: indented to label-text column.
			lineText = contIndent + tl
		}

		if isHL {
			// Pad each line to innerW with the highlight background.
			styled := hlStyle.Width(innerW).Render(lineText)
			resultLines = append(resultLines, styled)
		} else {
			// Non-highlighted: no padding, just colour.
			if li == 0 {
				numStyled := mutedStyle.Render(prefix)
				if f.multi {
					var markerStyled string
					if f.toggled[i] {
						markerStyled = markerSelStyle.Render("●") + " "
					} else {
						markerStyled = mutedStyle.Render("○") + " "
					}
					resultLines = append(resultLines, numStyled+markerStyled+normStyle.Render(tl))
				} else {
					resultLines = append(resultLines, numStyled+normStyle.Render(tl))
				}
			} else {
				resultLines = append(resultLines, normStyle.Render(lineText))
			}
			continue
		}
	}
	return resultLines
}

// view renders the list rows with optional windowed scroll.
// innerW is the width available inside the outer frame.
func (f *chooseField) view(innerW int, focused bool) string {
	viewStart, viewEnd, showUp, showDown := f.windowBounds()

	selBg, selFg := f.theme.ButtonSelBg, f.theme.ButtonSelFg
	switch f.variant {
	case "danger":
		selBg = f.theme.Danger
	case "warning":
		selBg = f.theme.Warning
		selFg = f.theme.Base
	}

	hlStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(selBg)).
		Foreground(lipgloss.Color(selFg))
	normStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(f.theme.Text))
	mutedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(f.theme.Muted))
	markerSelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(f.theme.Accent))

	var rows []string

	// Scroll indicator at top if clipped.
	if showUp {
		rows = append(rows, mutedStyle.Render("  ↑ more"))
	}

	for i := viewStart; i < viewEnd; i++ {
		num := fmt.Sprintf("%d", i+1)
		isHL := focused && i == f.highlight

		if f.isOtherRow(i) {
			if isHL && f.otherField != nil {
				// Render inline text field with aligned gutter.
				// The number gutter (" N ") is prefixed on the box's FIRST line;
				// continuation lines get equal-width spaces so the box's left
				// border stays vertically consistent.
				gutterText := " " + num + " "
				gutterLen := 1 + len(num) + 1
				gutterBlank := strings.Repeat(" ", gutterLen)
				boxW := innerW - gutterLen
				if boxW < 1 {
					boxW = 1
				}
				boxView := f.otherField.view(boxW, true)
				boxLines := strings.Split(boxView, "\n")
				var guttered []string
				for li, bl := range boxLines {
					if li == 0 {
						guttered = append(guttered, mutedStyle.Render(gutterText)+bl)
					} else {
						guttered = append(guttered, gutterBlank+bl)
					}
				}
				rows = append(rows, guttered...)
			} else {
				label := fmt.Sprintf(" %s %s ", num, f.otherLabel)
				if isHL {
					rows = append(rows, hlStyle.Width(innerW).Render(label))
				} else {
					rows = append(rows, mutedStyle.Render(label))
				}
			}
			continue
		}

		optRows := f.renderOptionRow(i, innerW, isHL, hlStyle, normStyle, mutedStyle, markerSelStyle)
		rows = append(rows, optRows...)
	}

	// Scroll indicator at bottom if clipped.
	if showDown {
		rows = append(rows, mutedStyle.Render("  ↓ more"))
	}

	return strings.Join(rows, "\n")
}

// value returns the selected value(s).
// Single: selected option or typed other text.
// Multi: \n-joined selected options (+ other text if set).
func (f *chooseField) value() string {
	// When the highlight is on the other row and there's an active embedded field,
	// return its current buffer (this covers both the in-progress and submitted cases).
	if f.isOtherRow(f.highlight) && f.otherField != nil {
		otherVal := f.otherField.value()
		if f.multi && otherVal != "" {
			var parts []string
			for i, opt := range f.options {
				if f.toggled[i] {
					parts = append(parts, opt)
				}
			}
			parts = append(parts, otherVal)
			return strings.Join(parts, "\n")
		}
		return otherVal
	}
	if f.multi {
		var parts []string
		for i, opt := range f.options {
			if f.toggled[i] {
				parts = append(parts, opt)
			}
		}
		return strings.Join(parts, "\n")
	}
	// Single
	if f.selected >= 0 && f.selected < len(f.options) {
		return f.options[f.selected]
	}
	return ""
}

// filled returns true if a selection has been made (single) or ≥1 selected (multi).
// When the other row is highlighted and the in-progress buffer is non-empty, we treat
// that as filled — otherwise Tab-away from the other field wedges a required form
// (the form intercepts Tab before the field can commit).
func (f *chooseField) filled() bool {
	if f.isOtherRow(f.highlight) && f.otherField != nil && f.otherField.value() != "" {
		return true
	}
	if f.multi {
		for _, t := range f.toggled {
			if t {
				return true
			}
		}
		return false
	}
	return f.selected >= 0
}

// optionLineCount returns the number of visual lines a single option row at
// index i occupies given innerW, accounting for label wrapping.
// The "other" row always occupies 1 line (or otherField.lines() when active,
// handled separately in lines()).
func (f *chooseField) optionLineCount(i, innerW int) int {
	if f.isOtherRow(i) {
		return 1
	}
	num := fmt.Sprintf("%d", i+1)
	prefixLen := 1 + len(num) + 1
	trailingLen := 1
	const markerLen = 2
	textColW := innerW - prefixLen - trailingLen
	if f.multi {
		textColW -= markerLen
	}
	if textColW < 1 {
		textColW = 1
	}
	lines := wrapLabel(f.options[i], textColW)
	return len(lines)
}

// lines returns the rendered height of this field.
// It mirrors the row count that view() emits: window rows + indicator rows.
// When the other row is highlighted and the embedded textField is visible, its
// multi-line box (border + textarea height) is substituted for the single-row
// placeholder.  When option labels wrap, each extra visual line is counted.
func (f *chooseField) lines(innerW int) int {
	viewStart, viewEnd, showUp, showDown := f.windowBounds()
	count := 0
	for i := viewStart; i < viewEnd; i++ {
		if f.isOtherRow(i) && f.isOtherRow(f.highlight) && f.otherField != nil {
			// The gutter takes `1 + len(num) + 1` chars; pass remaining width to the box.
			num := fmt.Sprintf("%d", i+1)
			gutterLen := 1 + len(num) + 1
			boxW := innerW - gutterLen
			if boxW < 1 {
				boxW = 1
			}
			count += f.otherField.lines(boxW)
		} else {
			count += f.optionLineCount(i, innerW)
		}
	}
	if showUp {
		count++
	}
	if showDown {
		count++
	}
	return count
}

// initCmd returns nil — the choose field needs no cursor blink.
func (f *chooseField) initCmd() tea.Cmd { return nil }
