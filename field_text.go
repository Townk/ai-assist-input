package main

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textarea"
	"charm.land/lipgloss/v2"
)

// textField wraps a textarea.Model and implements the field interface. It
// covers both the "line" variant (singleLine=true, height=1, no scrollbar) and
// the "text" variant (singleLine=false, multiline with scrollbar).
type textField struct {
	ta          textarea.Model
	theme       Theme
	singleLine  bool
	taHeight    int
	placeholder string // original placeholder text (so viewWith can hide/restore it)
}

// taStyle carries the focus-dependent colors used to render a textField box.
// It lets the choose "other" row recolor the whole widget (border, icon, text,
// background) per focus state, and hide the placeholder when unfocused+empty.
type taStyle struct {
	icon        string // icon glyph foreground
	border      string // box border foreground
	text        string // textarea text foreground
	bg          string // box interior background ("" = none/terminal default)
	placeholder bool   // show the placeholder when the field is empty
}

// newTextField constructs a textField. value is the initial text; placeholder
// is shown when empty; height is the textarea viewport rows; singleLine true
// disables newline insertion and the scrollbar.
func newTextField(theme Theme, value, placeholder string, height int, singleLine bool) *textField {
	ta := textarea.New()
	ta.Placeholder = placeholder
	ta.ShowLineNumbers = false
	ta.DynamicHeight = false
	ta.Prompt = ""

	s := textarea.DefaultDarkStyles()
	text := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Text))
	s.Focused.Base = lipgloss.NewStyle()
	s.Blurred.Base = lipgloss.NewStyle()
	s.Focused.Text = text
	s.Blurred.Text = text
	s.Focused.CursorLine = lipgloss.NewStyle()
	s.Blurred.CursorLine = lipgloss.NewStyle()
	ta.SetStyles(s)

	if value != "" {
		ta.SetValue(value)
		ta.MoveToEnd()
	}
	ta.Focus()
	if height < 1 {
		height = 1
	}
	ta.SetWidth(60)
	ta.SetHeight(height)

	return &textField{
		ta:          ta,
		theme:       theme,
		singleLine:  singleLine,
		taHeight:    height,
		placeholder: placeholder,
	}
}

// setWidth sizes the textarea from the innerW (frame-chrome already removed by
// the caller). It subtracts the inner-box chrome (border + left pad + icon col,
// plus scroll columns for multiline).
func (f *textField) setWidth(innerW int) {
	taW := innerW - boxBorder - boxPadL - iconCol
	if !f.singleLine {
		taW -= scrollGap + scrollCol
	}
	if taW < 1 {
		taW = 1
	}
	f.ta.SetWidth(taW)
	f.ta.SetHeight(f.taHeight)
}

// handle processes one message while the field is focused, returning the
// (possibly updated) field, a fieldAction, and any bubbletea Cmd.
func (f *textField) handle(msg tea.Msg) (field, fieldAction, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.PasteMsg:
		f.ta.InsertString(msg.Content)
		return f, fieldNone, nil
	case tea.KeyPressMsg:
		key := msg.Key()
		switch {
		case key.Code == tea.KeyEscape:
			return f, fieldCancel, nil
		case msg.String() == "ctrl+c":
			return f, fieldCancel, nil
		case key.Code == tea.KeyEnter && key.Mod.Contains(tea.ModShift):
			if !f.singleLine {
				f.ta.InsertRune('\n')
			}
			return f, fieldNone, nil
		case key.Code == tea.KeyEnter:
			return f, fieldDone, nil
		}
	}
	var cmd tea.Cmd
	f.ta, cmd = f.ta.Update(msg)
	return f, fieldNone, cmd
}

// view renders the rounded inner box with icon column, textarea, and (for
// multiline) the scrollbar, using the field's default theme colors. innerW is
// the width available inside the outer frame (frame-chrome already subtracted).
func (f *textField) view(innerW int, focused bool) string {
	return f.viewWith(innerW, taStyle{
		icon:        f.theme.Accent,
		border:      f.theme.FieldBorder,
		text:        f.theme.Text,
		bg:          "",
		placeholder: true,
	})
}

// viewWith renders the box with explicit focus-dependent colors. The default
// view() delegates here with the theme defaults; the choose "other" row passes
// muted colors when unfocused (and hides the placeholder) and selected
// background + bright-white foregrounds when focused.
func (f *textField) viewWith(innerW int, st taStyle) string {
	// Size the textarea from innerW each render pass.
	taW := innerW - boxBorder - boxPadL - iconCol
	if !f.singleLine {
		taW -= scrollGap + scrollCol
	}
	if taW < 1 {
		taW = 1
	}
	f.ta.SetWidth(taW)
	f.ta.SetHeight(f.taHeight)

	// Toggle the placeholder (restored before return).
	saved := f.ta.Placeholder
	if st.placeholder {
		f.ta.Placeholder = f.placeholder
	} else {
		f.ta.Placeholder = ""
	}
	defer func() { f.ta.Placeholder = saved }()

	// Apply the requested text foreground (and background, if any). Leaving the
	// text background unset lets the box-level Background fill the interior.
	s := textarea.DefaultDarkStyles()
	textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(st.text))
	s.Focused.Base = lipgloss.NewStyle()
	s.Blurred.Base = lipgloss.NewStyle()
	s.Focused.Text = textStyle
	s.Blurred.Text = textStyle
	s.Focused.CursorLine = lipgloss.NewStyle()
	s.Blurred.CursorLine = lipgloss.NewStyle()
	if st.bg != "" {
		ph := s.Focused.Placeholder.Background(lipgloss.Color(st.bg))
		s.Focused.Placeholder = ph
		s.Blurred.Placeholder = ph
	}
	f.ta.SetStyles(s)

	body := lipgloss.JoinHorizontal(lipgloss.Top, iconColumnColored(f.ta.Height(), st.icon), f.ta.View())
	if !f.singleLine {
		gap := strings.TrimRight(strings.Repeat(strings.Repeat(" ", scrollGap)+"\n", f.ta.Height()), "\n")
		body = lipgloss.JoinHorizontal(lipgloss.Top, body, gap, scrollbar(f))
	}
	box := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(st.border)).
		Padding(0, 0, 0, boxPadL)
	if st.bg != "" {
		box = box.Background(lipgloss.Color(st.bg)).BorderBackground(lipgloss.Color(st.bg))
	}
	return box.Render(body)
}

func (f *textField) value() string { return f.ta.Value() }
func (f *textField) filled() bool  { return f.value() != "" }

// lines returns the total rendered height of this field (textarea rows + box
// border rows of 2).
func (f *textField) lines(innerW int) int { return f.taHeight + boxBorder }

func (f *textField) initCmd() tea.Cmd { return textarea.Blink }

// --- helpers (moved from input.go) ------------------------------------------

func visualLineCount(f *textField) int {
	w := f.ta.Width()
	if w < 1 {
		return f.ta.LineCount()
	}
	total := 0
	for _, line := range strings.Split(f.ta.Value(), "\n") {
		rows := (lipgloss.Width(line) + w - 1) / w
		if rows < 1 {
			rows = 1
		}
		total += rows
	}
	return total
}

func scrollbar(f *textField) string {
	h := f.ta.Height()
	if h < 1 {
		h = 1
	}
	off := f.ta.ScrollYOffset()
	total := visualLineCount(f)
	if total < off+h {
		total = off + h
	}
	if total <= h {
		return strings.TrimRight(strings.Repeat(" \n", h), "\n")
	}
	thumb := h * h / total
	if thumb < 1 {
		thumb = 1
	}
	maxOff := total - h
	pos := 0
	if maxOff > 0 {
		pos = (h - thumb) * off / maxOff
	}
	track := lipgloss.NewStyle().Foreground(lipgloss.Color(f.theme.Rule))
	thumbStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(f.theme.ScrollThumb))
	rows := make([]string, h)
	for i := range rows {
		if i >= pos && i < pos+thumb {
			rows[i] = thumbStyle.Render("┃")
		} else {
			rows[i] = track.Render("│")
		}
	}
	return strings.Join(rows, "\n")
}

// iconColumnColored renders the prompt-icon column with an explicit foreground
// color (so the choose "other" row can recolor the icon per focus state).
func iconColumnColored(h int, fg string) string {
	if h < 1 {
		h = 1
	}
	icon := lipgloss.NewStyle().Foreground(lipgloss.Color(fg)).Render(promptIcon)
	rows := make([]string, h)
	rows[0] = icon + "  "
	for i := 1; i < h; i++ {
		rows[i] = strings.Repeat(" ", iconCol)
	}
	return strings.Join(rows, "\n")
}
