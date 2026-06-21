package main

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textarea"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
)

const promptIcon = "󰧑"

const (
	boxBorder = 2 // inner rounded box, left + right
	boxPadL   = 1 // inner box left padding
	iconCol   = 3 // prompt icon (1) + 2-space gap
	scrollGap = 1 // space between input text and scroll column
	scrollCol = 1 // scroll-indicator column
)

type model struct {
	textarea   textarea.Model
	theme      Theme
	variant    string
	title      string
	singleLine bool
	width      int
	taHeight   int
	padding    int
	inset      int
	submitted  bool
	quitting   bool
}

// initialModel keeps the original signature the existing tests call (text, 1/1
// padding/inset, default theme).
func initialModel(value, title string, height int) model {
	return newInputModel(defaultTheme(), "default", title, value, "", height, 1, 1, false)
}

func newInputModel(theme Theme, variant, title, value, placeholder string, height, padding, inset int, singleLine bool) model {
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

	return model{
		textarea: ta, theme: theme, variant: variant, title: title,
		singleLine: singleLine, width: 64, taHeight: height,
		padding: padding, inset: inset,
	}
}

func (m model) Init() tea.Cmd { return textarea.Blink }

// resize sets the textarea width from the pane, subtracting the outer frame
// (border + h-padding) and the inner box (border + left pad + icon, plus the
// scroll columns for multi-line text).
func (m *model) resize() {
	chrome := frameBorder + 2*frameHPad + boxBorder + boxPadL + iconCol
	if !m.singleLine {
		chrome += scrollGap + scrollCol
	}
	innerW := m.width - chrome
	if innerW < 1 {
		innerW = 1
	}
	m.textarea.SetWidth(innerW)
	m.textarea.SetHeight(m.taHeight)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.resize()
		return m, nil
	case tea.PasteMsg:
		m.textarea.InsertString(msg.Content)
		return m, nil
	case tea.KeyPressMsg:
		key := msg.Key()
		switch {
		case key.Code == tea.KeyEscape:
			m.quitting = true
			return m, tea.Quit
		case key.Code == tea.KeyEnter && key.Mod.Contains(tea.ModShift):
			if !m.singleLine { // line never inserts newlines
				m.textarea.InsertRune('\n')
			}
			return m, nil
		case key.Code == tea.KeyEnter:
			m.submitted = true
			return m, tea.Quit
		case msg.String() == "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

// --- helpers moved verbatim from the old main.go ---------------------------

func visualLineCount(m model) int {
	w := m.textarea.Width()
	if w < 1 {
		return m.textarea.LineCount()
	}
	total := 0
	for _, line := range strings.Split(m.textarea.Value(), "\n") {
		rows := (lipgloss.Width(line) + w - 1) / w
		if rows < 1 {
			rows = 1
		}
		total += rows
	}
	return total
}

func scrollbar(m model) string {
	h := m.textarea.Height()
	if h < 1 {
		h = 1
	}
	off := m.textarea.ScrollYOffset()
	total := visualLineCount(m)
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
	track := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Rule))
	thumbStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.ScrollThumb))
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

func iconColumn(h int, theme Theme) string {
	if h < 1 {
		h = 1
	}
	icon := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Accent)).Render(promptIcon)
	rows := make([]string, h)
	rows[0] = icon + "  "
	for i := 1; i < h; i++ {
		rows[i] = strings.Repeat(" ", iconCol)
	}
	return strings.Join(rows, "\n")
}

// --- render ----------------------------------------------------------------

func (m model) hint() string {
	key := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Key))
	word := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Muted))
	seg := func(k, w string) string { return key.Render(k) + word.Render(" "+w) }
	sep := word.Render(" · ")
	if m.singleLine {
		return strings.Join([]string{seg("󰌑", "submit"), seg("󱊷", "cancel")}, sep)
	}
	return strings.Join([]string{seg("󰌑", "submit"), seg("󰘶󰌑", "newline"), seg("󱊷", "cancel")}, sep)
}

func (m model) render() string {
	body := lipgloss.JoinHorizontal(lipgloss.Top, iconColumn(m.textarea.Height(), m.theme), m.textarea.View())
	if !m.singleLine {
		gap := strings.TrimRight(strings.Repeat(strings.Repeat(" ", scrollGap)+"\n", m.textarea.Height()), "\n")
		body = lipgloss.JoinHorizontal(lipgloss.Top, body, gap, scrollbar(m))
	}
	box := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(m.theme.FieldBorder)).
		Padding(0, 0, 0, boxPadL).
		Render(body)
	return renderFrame(m.theme, m.variant, m.title, []string{box}, m.hint(), m.width, m.padding, m.inset)
}

func (m model) View() tea.View {
	v := tea.NewView(m.render())
	v.KeyboardEnhancements = tea.KeyboardEnhancements{ReportAllKeysAsEscapeCodes: true}
	return v
}

func runInput(theme Theme, variant, title, value, placeholder string, height, padding, inset int, singleLine bool) {
	fm, err := tea.NewProgram(
		newInputModel(theme, variant, title, value, placeholder, height, padding, inset, singleLine),
		tea.WithOutput(os.Stderr),
		tea.WithColorProfile(colorprofile.TrueColor),
	).Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ai-assist-input: error: %v\n", err)
		os.Exit(1)
	}
	res := fm.(model)
	if res.submitted {
		fmt.Print(res.textarea.Value())
		os.Exit(0)
	}
	os.Exit(130)
}

// TEMP stub — replaced by confirm.go in Task 5.
func runConfirm(theme Theme, variant, title, prompt, affirmative, negative string, defaultNegative bool, padding, inset int) {
	os.Exit(2)
}
