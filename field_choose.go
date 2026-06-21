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

	highlight int       // currently highlighted row index (0-based, over all rows)
	selected  int       // single mode: selected option index (-1 = none)
	toggled   []bool    // multi mode: toggled[i] for options[i]
	otherText string    // text typed when "other" was chosen
	otherActive bool    // true when the embedded textField is focused
	otherField  *textField
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
	// Delegate to the embedded textField when other-mode is active.
	if f.otherActive && f.otherField != nil {
		f2, act, cmd := f.otherField.handle(msg)
		c := *f
		c.otherField = f2.(*textField)
		switch act {
		case fieldDone:
			c.otherText = c.otherField.value()
			c.otherActive = false
			return &c, fieldDone, nil
		case fieldCancel:
			c.otherActive = false
			c.otherField = nil
			c.otherText = ""
			return &c, fieldNone, nil
		}
		return &c, fieldNone, cmd
	}

	kp, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return f, fieldNone, nil
	}

	c := *f
	total := c.totalRows()

	switch {
	case kp.Code == tea.KeyEscape:
		return f, fieldCancel, nil

	case kp.String() == "ctrl+c":
		return f, fieldCancel, nil

	case kp.Code == 'j' || kp.Code == tea.KeyDown:
		if c.highlight < total-1 {
			c.highlight++
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
			// Activate the other free-text entry.
			c.otherField = newTextField(c.theme, "", "", 1, true)
			c.otherActive = true
			return &c, fieldNone, nil
		}
		if c.multi {
			c.toggled[n] = !c.toggled[n]
			return &c, fieldNone, nil
		}
		// Single: select and done.
		c.selected = n
		return &c, fieldDone, nil

	case kp.Code == tea.KeySpace && c.multi:
		if !c.isOtherRow(c.highlight) {
			c.toggled[c.highlight] = !c.toggled[c.highlight]
		}
		return &c, fieldNone, nil

	case kp.Code == tea.KeyEnter:
		if c.isOtherRow(c.highlight) {
			// Activate the other free-text entry.
			c.otherField = newTextField(c.theme, "", "", 1, true)
			c.otherActive = true
			return &c, fieldNone, nil
		}
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
			if f.otherActive && f.otherField != nil {
				// Render inline text field.
				label := mutedStyle.Render(num+" ") + f.otherField.view(innerW-4, true)
				rows = append(rows, label)
			} else {
				label := fmt.Sprintf("%s %s", num, f.otherLabel)
				if isHL {
					rows = append(rows, hlStyle.Render(label))
				} else {
					rows = append(rows, mutedStyle.Render(label))
				}
			}
			continue
		}

		opt := f.options[i]
		var marker string
		if f.multi {
			if f.toggled[i] {
				marker = markerSelStyle.Render("●") + " "
			} else {
				marker = mutedStyle.Render("○") + " "
			}
		}

		if isHL {
			numPart := hlStyle.Render(num)
			sep := hlStyle.Render(" ")
			var row string
			if f.multi {
				row = numPart + sep + marker + hlStyle.Render(opt)
			} else {
				row = numPart + sep + hlStyle.Render(opt)
			}
			rows = append(rows, row)
		} else {
			numPart := mutedStyle.Render(num)
			var row string
			if f.multi {
				row = numPart + " " + marker + normStyle.Render(opt)
			} else {
				row = numPart + " " + normStyle.Render(opt)
			}
			rows = append(rows, row)
		}
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
	if f.otherActive && f.otherField != nil {
		return f.otherField.value()
	}
	if f.otherText != "" {
		if !f.multi {
			return f.otherText
		}
	}
	if f.multi {
		var parts []string
		for i, opt := range f.options {
			if f.toggled[i] {
				parts = append(parts, opt)
			}
		}
		if f.otherText != "" {
			parts = append(parts, f.otherText)
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
func (f *chooseField) filled() bool {
	if f.multi {
		for _, t := range f.toggled {
			if t {
				return true
			}
		}
		return f.otherText != ""
	}
	return f.selected >= 0 || f.otherText != ""
}

// lines returns the rendered height of this field.
// It mirrors the row count that view() emits: window rows + indicator rows.
func (f *chooseField) lines(innerW int) int {
	viewStart, viewEnd, showUp, showDown := f.windowBounds()
	count := viewEnd - viewStart
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
