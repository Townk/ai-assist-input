package main

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
)

// chooseModel is the thin bubbletea wrapper over a chooseField. It owns the
// frame (title/prompt/variant/theme/width/padding/inset) and delegates all key
// handling to the field.
type chooseModel struct {
	fld       *chooseField
	theme     Theme
	variant   string
	title     string
	prompt    string
	multi     bool
	width     int
	padding   int
	inset     int
	cancelled bool
	done      bool
}

func newChooseModel(theme Theme, variant, title, prompt string, options []string, multi bool, other string, padding, inset int) chooseModel {
	fld := newChooseField(theme, variant, options, multi, other)
	return chooseModel{
		fld:     fld,
		theme:   theme,
		variant: variant,
		title:   title,
		prompt:  prompt,
		multi:   multi,
		width:   54,
		padding: padding,
		inset:   inset,
	}
}

func (m chooseModel) Init() tea.Cmd { return nil }

func (m chooseModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	case tea.KeyPressMsg:
		f2, act, cmd := m.fld.handle(msg)
		m.fld = f2.(*chooseField)
		switch act {
		case fieldDone:
			m.done = true
			return m, tea.Quit
		case fieldCancel:
			m.cancelled = true
			return m, tea.Quit
		}
		return m, cmd
	}
	return m, nil
}

func (m chooseModel) innerW() int {
	w := m.width - frameBorder - 2*frameHPad
	if w < 1 {
		w = 1
	}
	return w
}

// chooseHint builds the keyboard-hint line for a choose dialog.
// Only key glyphs are rendered in theme.Key (bright white); separators,
// dashes, and descriptive words are in theme.Muted (dark grey).
// rows is the total number of selectable rows; it caps the shortcut range at 9.
func chooseHint(t Theme, rows int, multi bool) string {
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Key))
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Muted))
	seg := func(k, w string) string { return keyStyle.Render(k) + mutedStyle.Render(w) }
	sep := mutedStyle.Render(" · ")

	// Move segment: ↑↓ / jk move
	move := seg("↑↓", "") + mutedStyle.Render("/") + seg("jk", " move")

	// Pick segment: 1-N pick (or just 1 pick when only 1 row)
	var pick string
	n := rows
	if n > 9 {
		n = 9
	}
	if n <= 1 {
		pick = seg("1", " pick")
	} else {
		pick = keyStyle.Render("1") + mutedStyle.Render("-") + keyStyle.Render(fmt.Sprintf("%d", n)) + mutedStyle.Render(" pick")
	}

	segs := []string{move, pick}

	if multi {
		segs = append(segs, seg("space", " toggle"))
	}

	segs = append(segs, seg("↵", " ok"), seg("󱊷", " dismiss"))
	return strings.Join(segs, sep)
}

func (m chooseModel) hint() string {
	return chooseHint(m.theme, m.fld.totalRows(), m.multi)
}

func (m chooseModel) render() string {
	iW := m.innerW()
	sections := []string{}
	if m.prompt != "" {
		sections = append(sections, lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Text)).Render(m.prompt))
	}
	sections = append(sections, m.fld.view(iW, true))
	return renderFrame(m.theme, m.variant, m.title, sections, m.hint(), m.width, m.padding, m.inset)
}

func (m chooseModel) View() tea.View { return tea.NewView(m.render()) }

func runChoose(theme Theme, variant, title, prompt string, options []string, multi bool, other string, padding, inset int) {
	fm, err := tea.NewProgram(
		newChooseModel(theme, variant, title, prompt, options, multi, other, padding, inset),
		tea.WithOutput(os.Stderr),
		tea.WithColorProfile(colorprofile.TrueColor),
	).Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ai-assist-input: error: %v\n", err)
		os.Exit(1)
	}
	res := fm.(chooseModel)
	if res.cancelled || !res.done {
		os.Exit(130)
	}
	val := res.fld.value()
	if val == "" {
		os.Exit(130)
	}
	fmt.Print(val)
	os.Exit(0)
}
