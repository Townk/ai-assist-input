package main

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
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

// model is the thin standalone bubbletea model. It wraps a single field and
// owns the frame (title/prompt/variant/theme/width/padding/inset).
type model struct {
	fld       field
	theme     Theme
	variant   string
	title     string
	prompt    string
	width     int
	padding   int
	inset     int
	submitted bool
	quitting  bool
	// singleLine is kept for hint rendering only.
	singleLine bool
}

// initialModel keeps the original signature the existing tests call (text, 1/1
// padding/inset, default theme).
func initialModel(value, title string, height int) model {
	return newInputModel(defaultTheme(), "default", title, "", value, "", height, 1, 1, false)
}

func newInputModel(theme Theme, variant, title, prompt, value, placeholder string, height, padding, inset int, singleLine bool) model {
	fld := newTextField(theme, value, placeholder, height, singleLine)
	return model{
		fld: fld, theme: theme, variant: variant, title: title, prompt: prompt,
		singleLine: singleLine, width: 64,
		padding: padding, inset: inset,
	}
}

func (m model) Init() tea.Cmd { return m.fld.initCmd() }

// innerW computes the width available inside the outer frame for the field.
func (m *model) innerW() int {
	w := m.width - frameBorder - 2*frameHPad
	if w < 1 {
		w = 1
	}
	return w
}

// resize re-sizes the field from the current pane width.
func (m *model) resize() {
	if tf, ok := m.fld.(*textField); ok {
		tf.setWidth(m.innerW())
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.resize()
		return m, nil
	}
	f, act, cmd := m.fld.handle(msg)
	m.fld = f
	switch act {
	case fieldDone:
		m.submitted = true
		return m, tea.Quit
	case fieldCancel:
		m.quitting = true
		return m, tea.Quit
	}
	return m, cmd
}

// --- render ------------------------------------------------------------------

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
	iW := m.innerW()
	sections := []string{}
	if m.prompt != "" {
		sections = append(sections, lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Text)).Render(m.prompt))
	}
	sections = append(sections, m.fld.view(iW, true))
	return renderFrame(m.theme, m.variant, m.title, sections, m.hint(), m.width, m.padding, m.inset)
}

func (m model) View() tea.View {
	v := tea.NewView(m.render())
	v.KeyboardEnhancements = tea.KeyboardEnhancements{ReportAllKeysAsEscapeCodes: true}
	return v
}

func runInput(theme Theme, variant, title, prompt, value, placeholder string, height, padding, inset int, singleLine bool) {
	fm, err := tea.NewProgram(
		newInputModel(theme, variant, title, prompt, value, placeholder, height, padding, inset, singleLine),
		tea.WithOutput(os.Stderr),
		tea.WithColorProfile(colorprofile.TrueColor),
	).Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ai-assist-input: error: %v\n", err)
		os.Exit(1)
	}
	res := fm.(model)
	if res.submitted {
		fmt.Print(res.fld.value())
		os.Exit(0)
	}
	os.Exit(130)
}
