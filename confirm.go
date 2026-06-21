package main

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
)

type confirmModel struct {
	theme          Theme
	variant        string
	title          string
	prompt         string
	affirmative    string
	negative       string
	affKey, negKey rune
	focus          int // 0 = affirmative (left), 1 = negative (right)
	width          int
	padding, inset int
	done           bool
	cancelled      bool
	result         string // "yes" | "no"
}

func newConfirmModel(theme Theme, variant, title, prompt, affirmative, negative string, defaultNegative bool, padding, inset int) confirmModel {
	aff, neg := deriveKeys(affirmative, negative)
	focus := 0
	if defaultNegative {
		focus = 1
	}
	return confirmModel{
		theme: theme, variant: variant, title: title, prompt: prompt,
		affirmative: affirmative, negative: negative,
		affKey: aff, negKey: neg, focus: focus,
		width: 54, padding: padding, inset: inset,
	}
}

func (m confirmModel) Init() tea.Cmd { return nil }

func confirmKeyString(msg tea.KeyPressMsg) string {
	switch msg.Key().Code {
	case tea.KeyEscape:
		return "esc"
	case tea.KeyEnter:
		return "enter"
	case tea.KeyTab:
		return "tab"
	case tea.KeyLeft:
		return "left"
	case tea.KeyRight:
		return "right"
	}
	return msg.String()
}

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	case tea.KeyPressMsg:
		switch resolveConfirmKey(confirmKeyString(msg), m.affKey, m.negKey) {
		case actAffirm:
			m.done, m.result = true, "yes"
			return m, tea.Quit
		case actNegate:
			m.done, m.result = true, "no"
			return m, tea.Quit
		case actSubmit:
			m.done = true
			if m.focus == 0 {
				m.result = "yes"
			} else {
				m.result = "no"
			}
			return m, tea.Quit
		case actFocusLeft:
			m.focus = 0
		case actFocusRight:
			m.focus = 1
		case actToggle:
			m.focus = 1 - m.focus
		case actCancel:
			m.cancelled = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m confirmModel) button(label string, focused bool) string {
	st := lipgloss.NewStyle().Padding(0, 2)
	if focused {
		bg, fg := m.theme.ButtonSelBg, m.theme.ButtonSelFg
		switch m.variant {
		case "danger":
			bg = m.theme.Danger
		case "warning":
			bg, fg = m.theme.Warning, m.theme.Base
		}
		return st.Background(lipgloss.Color(bg)).Foreground(lipgloss.Color(fg)).Render(label)
	}
	return st.Background(lipgloss.Color(m.theme.ButtonBg)).Foreground(lipgloss.Color(m.theme.ButtonFg)).Render(label)
}

func (m confirmModel) hint() string {
	key := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Key))
	word := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Muted))
	seg := func(k, w string) string { return key.Render(k) + word.Render(" "+w) }
	sep := word.Render(" · ")
	return strings.Join([]string{
		seg("󱊷", "dismiss"),
		seg(string(m.affKey), strings.ToLower(m.affirmative)),
		seg(string(m.negKey), strings.ToLower(m.negative)),
	}, sep)
}

func (m confirmModel) render() string {
	prompt := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Text)).Render(m.prompt)
	buttons := m.button(m.affirmative, m.focus == 0) + "    " + m.button(m.negative, m.focus == 1)
	return renderFrame(m.theme, m.variant, m.title, []string{prompt, buttons}, m.hint(), m.width, m.padding, m.inset)
}

func (m confirmModel) View() tea.View { return tea.NewView(m.render()) }

func runConfirm(theme Theme, variant, title, prompt, affirmative, negative string, defaultNegative bool, padding, inset int) {
	fm, err := tea.NewProgram(
		newConfirmModel(theme, variant, title, prompt, affirmative, negative, defaultNegative, padding, inset),
		tea.WithOutput(os.Stderr),
		tea.WithColorProfile(colorprofile.TrueColor),
	).Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ai-assist-input: error: %v\n", err)
		os.Exit(1)
	}
	res := fm.(confirmModel)
	if res.cancelled {
		os.Exit(130)
	}
	fmt.Print(res.result)
	if res.result == "yes" {
		os.Exit(0)
	}
	os.Exit(1)
}
