package main

import (
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textarea"
	"charm.land/lipgloss/v2"
)

// Catppuccin Mocha palette
const (
	colorMauve   = "#cba6f7"
	colorText    = "#cdd6f4"
	colorSurface0 = "#313244"
)

type model struct {
	textarea  textarea.Model
	header    string
	submitted bool
	quitting  bool
}

func initialModel(value, header string) model {
	ta := textarea.New()
	ta.Placeholder = ""
	ta.ShowLineNumbers = false
	ta.DynamicHeight = true
	ta.MinHeight = 3

	// Catppuccin Mocha styles
	s := textarea.DefaultDarkStyles()
	base := lipgloss.NewStyle().Foreground(lipgloss.Color(colorText))
	s.Focused.Text = base
	s.Focused.Base = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(colorMauve)).
		Padding(0, 1)
	s.Focused.CursorLine = lipgloss.NewStyle().Background(lipgloss.Color(colorSurface0))
	s.Blurred.Text = base
	s.Blurred.Base = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(colorSurface0)).
		Padding(0, 1)
	ta.SetStyles(s)

	if value != "" {
		ta.SetValue(value)
		ta.MoveToEnd()
	}

	ta.Focus() //nolint:errcheck // returns a Cmd we'll send in Init

	return model{
		textarea: ta,
		header:   header,
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		key := msg.Key()
		switch {
		case key.Code == tea.KeyEscape:
			m.quitting = true
			return m, tea.Quit
		case key.Code == tea.KeyEnter && key.Mod.Contains(tea.ModShift):
			// Shift+Enter: insert a newline into the textarea
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd
		case key.Code == tea.KeyEnter:
			// Plain Enter: submit
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

func (m model) View() tea.View {
	var content string

	if m.header != "" {
		headerStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorMauve)).
			Bold(true).
			MarginBottom(1)
		content = headerStyle.Render(m.header) + "\n"
	}

	content += m.textarea.View()

	if !m.quitting {
		hint := lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorSurface0)).
			Render("  enter: submit • shift+enter: newline • esc: cancel")
		content += "\n" + hint
	}

	v := tea.NewView(content)
	// Enable kitty keyboard enhancements so Shift+Enter is distinguishable
	// from plain Enter (delivers \e[13;2u for Shift+Enter).
	v.KeyboardEnhancements = tea.KeyboardEnhancements{
		ReportAllKeysAsEscapeCodes: true,
	}
	return v
}

func main() {
	var value, header string
	flag.StringVar(&value, "value", "", "initial textarea content")
	flag.StringVar(&header, "header", "", "optional one-line header shown above the textarea")
	flag.Parse()

	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "ai-assist-input: unexpected arguments: %v\n", flag.Args())
		flag.Usage()
		os.Exit(1)
	}

	m := initialModel(value, header)
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ai-assist-input: error: %v\n", err)
		os.Exit(1)
	}

	result, ok := finalModel.(model)
	if !ok {
		os.Exit(1)
	}

	if result.submitted {
		fmt.Print(result.textarea.Value())
		os.Exit(0)
	}
	// cancelled (Esc or Ctrl+C)
	os.Exit(130)
}
