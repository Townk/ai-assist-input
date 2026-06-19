package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textarea"
	"charm.land/lipgloss/v2"
	"github.com/mattn/go-runewidth"
)

// Catppuccin Mocha palette
const (
	colorMauve    = "#cba6f7"
	colorText     = "#cdd6f4"
	colorSurface0 = "#313244" // dark grey — scroll track
	colorOverlay0 = "#6c7086"
	colorOverlay1 = "#7f849c" // lighter grey — scroll thumb
)

// Layout. The modal is rendered entirely here (the wrapper just provides the
// floating pane), top to bottom:
//
//	  ▓▓▓ <title>            title   — 2-col indent
//	  ━━━…                   rule    — 2-col indent, (width-4) wide
//	  ╭…╮                    box top — 2-col left margin
//	  │ 󰧑  <text…>      ┃│   icon column + textarea + scroll column
//	  ╰…╯
//	    󰌑 : submit • …       hint    — aligned under the box content
//
// Width budget: cells the chrome takes out of the pane WIDTH around the textarea.
// Height is fixed (from --height; see resize) so long content scrolls.
const (
	frameWidth = 2 // rounded border, left + right
	framePad   = 2 // border padding, left + right
	scrollCol  = 1 // scroll-indicator column
	iconCol    = 4 // prompt-icon column (1-space lead + "󰧑" + 2-space gap)
	boxMargin  = 2 // inset of the input box from each pane edge
)

const promptIcon = "󰧑"

type model struct {
	textarea  textarea.Model
	title     string
	width     int // pane width (from WindowSizeMsg)
	taHeight  int // textarea viewport rows (from --height; fixed)
	submitted bool
	quitting  bool
}

func initialModel(value, title string, height int) model {
	ta := textarea.New()
	ta.Placeholder = ""
	ta.ShowLineNumbers = false
	ta.DynamicHeight = false // fixed viewport → long content scrolls, doesn't grow
	ta.Prompt = ""           // the prompt icon is rendered as a separate column in View

	// Borderless textarea; View() paints the rounded frame + the scroll column so
	// they sit together. No cursor-line highlight — the editing area stays one
	// uniform color.
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

	// Height is fixed (from --height); width gets a default until the first
	// WindowSizeMsg sizes us to the pane width.
	ta.SetWidth(60)
	ta.SetHeight(height)

	return model{textarea: ta, title: title, width: 64, taHeight: height}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

// resize sets the textarea WIDTH from the pane; height stays fixed at taHeight.
func (m *model) resize() {
	innerW := m.width - frameWidth - framePad - scrollCol - iconCol - 2*boxMargin
	if innerW < 1 {
		innerW = 1
	}
	m.textarea.SetWidth(innerW)
	m.textarea.SetHeight(m.taHeight)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width // width only — height is fixed (--height)
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

// iconColumn renders the prompt-icon column beside the textarea: the icon on the
// first row, blanks on the rest, so the input's continuation lines stay aligned
// under the text rather than under the icon.
func iconColumn(h int) string {
	if h < 1 {
		h = 1
	}
	icon := lipgloss.NewStyle().Foreground(lipgloss.Color(colorMauve)).Render(promptIcon)
	rows := make([]string, h)
	rows[0] = " " + icon + "  " // 1-space lead + icon (1 cell) + 2-space gap = iconCol wide
	for i := 1; i < h; i++ {
		rows[i] = strings.Repeat(" ", iconCol)
	}
	return strings.Join(rows, "\n")
}

// render builds the full modal frame as a string (View wraps it; kept separate
// so it's testable without a TTY).
func (m model) render() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorMauve))
	ruleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorOverlay0))
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorOverlay0))

	indent := strings.Repeat(" ", boxMargin)
	ruleW := m.width - 2*boxMargin
	if ruleW < 1 {
		ruleW = 1
	}
	title := indent + titleStyle.Render("▓▓▓ "+m.title)
	rule := indent + ruleStyle.Render(strings.Repeat("━", ruleW))

	body := lipgloss.JoinHorizontal(lipgloss.Top, iconColumn(m.textarea.Height()), m.textarea.View(), scrollbar(m))
	box := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(colorMauve)).
		Padding(0, 1).
		MarginLeft(boxMargin).
		Render(body)

	// Hint indent aligns under the box content: margin + left border + left pad.
	hintIndent := strings.Repeat(" ", boxMargin+frameWidth/2+framePad/2)
	hint := hintStyle.Render(hintIndent + "󰌑 : submit • 󰘶 󰌑 : newline • 󱊷 : cancel")

	return title + "\n" + rule + "\n" + box + "\n" + hint
}

func (m model) View() tea.View {
	v := tea.NewView(m.render())
	// Kitty keyboard enhancements so Shift+Enter (delivered as \e[13;2u) is
	// distinguishable from plain Enter.
	v.KeyboardEnhancements = tea.KeyboardEnhancements{ReportAllKeysAsEscapeCodes: true}
	return v
}

func main() {
	// Force narrow (1-cell) accounting for East-Asian-ambiguous characters and
	// nerd-font glyphs (the prompt/hint icons), so lipgloss's width matches what
	// the terminal renders and the icon column stays aligned. Must run before any
	// lipgloss call.
	os.Setenv("RUNEWIDTH_EASTASIAN", "0")
	runewidth.DefaultCondition.EastAsianWidth = false

	var value, title string
	var height int
	flag.StringVar(&value, "value", "", "initial textarea content")
	flag.StringVar(&title, "title", "", "modal title shown above the input (e.g. \"ai-assist\")")
	flag.IntVar(&height, "height", 10, "textarea viewport height in rows (the popup sets this so the modal fits the float)")
	flag.Parse()
	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "ai-assist-input: unexpected arguments: %v\n", flag.Args())
		flag.Usage()
		os.Exit(1)
	}
	if height < 1 {
		height = 1
	}

	// Render the TUI to stderr and keep stdout for the result only. Our caller
	// (zellij-modal --capture) redirects this process's stdout to a file to
	// capture the submitted text; if the TUI rendered to stdout, bubbletea would
	// see a non-TTY stdout and exit immediately (the "popup blinks" bug). stderr
	// stays attached to the pane's tty in both the capture path and the inline
	// fallback, so the UI shows there and the result goes to stdout.
	finalModel, err := tea.NewProgram(initialModel(value, title, height), tea.WithOutput(os.Stderr)).Run()
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
