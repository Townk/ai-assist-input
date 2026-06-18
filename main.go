package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textarea"
	"charm.land/lipgloss/v2"
)

// Catppuccin Mocha palette
const (
	colorMauve    = "#cba6f7"
	colorText     = "#cdd6f4"
	colorSurface0 = "#313244" // dark grey — scroll track
	colorOverlay0 = "#6c7086"
	colorOverlay1 = "#7f849c" // lighter grey — scroll thumb
)

// Frame budget: how many cells the chrome takes out of the pane.
const (
	frameWidth = 2 // rounded border, left + right
	framePad   = 2 // border padding, left + right
	scrollCol  = 1 // scroll-indicator column
	chromeRows = 3 // top + bottom border, plus the hint line below the box
)

type model struct {
	textarea  textarea.Model
	width     int
	height    int
	submitted bool
	quitting  bool
}

func initialModel(value string) model {
	ta := textarea.New()
	ta.Placeholder = ""
	ta.ShowLineNumbers = false
	ta.DynamicHeight = false // fixed viewport → long content scrolls, doesn't grow
	ta.Prompt = ""           // no per-line prompt column (the "inner left border")

	// The textarea is borderless; the model paints the rounded frame in View()
	// so the scroll indicator can sit inside it. No cursor-line highlight either
	// — the editing area stays one uniform color. No header: the zellij-modal
	// title block already labels the popup, so a header here would be redundant.
	s := textarea.DefaultDarkStyles()
	text := lipgloss.NewStyle().Foreground(lipgloss.Color(colorText))
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

	// Focus the model NewProgram actually runs (not a discarded Init() copy).
	ta.Focus()

	// Sensible defaults until the first WindowSizeMsg sizes us to the pane.
	ta.SetWidth(60)
	ta.SetHeight(8)

	return model{textarea: ta, width: 64, height: 11}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m *model) resize() {
	innerW := m.width - frameWidth - framePad - scrollCol
	if innerW < 1 {
		innerW = 1
	}
	taH := m.height - chromeRows
	if taH < 1 {
		taH = 1
	}
	m.textarea.SetWidth(innerW)
	m.textarea.SetHeight(taH)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
		return m, nil
	case tea.KeyPressMsg:
		key := msg.Key()
		switch {
		case key.Code == tea.KeyEscape:
			m.quitting = true
			return m, tea.Quit
		case key.Code == tea.KeyEnter && key.Mod.Contains(tea.ModShift):
			// Shift+Enter: insert a newline explicitly. Forwarding the key to the
			// textarea does nothing — its InsertNewline binding matches only plain
			// Enter, not the shifted chord.
			m.textarea.InsertRune('\n')
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

// scrollbar renders a single-column, viewport-height indicator beside the
// textarea. It reserves the column with blanks when everything fits, and shows
// a proportional mauve thumb once the content scrolls past the viewport.
func scrollbar(m model) string {
	h := m.textarea.Height()
	if h < 1 {
		h = 1
	}
	total := m.textarea.LineCount()
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
		pos = (h - thumb) * m.textarea.ScrollYOffset() / maxOff
	}

	track := lipgloss.NewStyle().Foreground(lipgloss.Color(colorSurface0))
	thumbStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorOverlay1))
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

func (m model) View() tea.View {
	body := lipgloss.JoinHorizontal(lipgloss.Top, m.textarea.View(), scrollbar(m))
	box := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(colorMauve)).
		Padding(0, 1).
		Render(body)

	hint := lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorOverlay0)).
		Render("  󰌑: submit • 󰘶 󰌑: newline • 󱊷: cancel")

	v := tea.NewView(box + "\n" + hint)
	// Kitty keyboard enhancements so Shift+Enter (delivered as \e[13;2u) is
	// distinguishable from plain Enter.
	v.KeyboardEnhancements = tea.KeyboardEnhancements{ReportAllKeysAsEscapeCodes: true}
	return v
}

func main() {
	var value string
	flag.StringVar(&value, "value", "", "initial textarea content")
	flag.Parse()
	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "ai-assist-input: unexpected arguments: %v\n", flag.Args())
		flag.Usage()
		os.Exit(1)
	}

	finalModel, err := tea.NewProgram(initialModel(value)).Run()
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
	os.Exit(130) // cancelled (Esc or Ctrl+C)
}
