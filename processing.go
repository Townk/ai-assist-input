package main

import (
	"bufio"
	"os"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Messages understood by processingModel.
type statusMsg string
type closeMsg struct{}
type submitMsg string

// processingModel is a Bubble Tea model that shows a spinner + status label
// inside the same renderFrame used by the input widget. It is swapped in
// in-place after the user submits input, so the floating pane never closes.
type processingModel struct {
	theme   Theme
	title   string
	width   int
	label   string
	inFifo  string // path to read status/close records from (empty = no re-issue)
	spinner spinner.Model
}

func newProcessingModel(theme Theme, title string, width int) processingModel {
	sp := spinner.New(
		spinner.WithSpinner(spinner.Points),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Accent))),
	)
	return processingModel{
		theme:   theme,
		title:   title,
		width:   width,
		label:   "Processing…",
		spinner: sp,
	}
}

func newProcessingModelWithFifo(theme Theme, title string, width int, inFifo string) processingModel {
	m := newProcessingModel(theme, title, width)
	m.inFifo = inFifo
	return m
}

func (m processingModel) Init() tea.Cmd {
	return tea.Tick(m.spinner.Spinner.FPS, func(t time.Time) tea.Msg {
		return spinner.TickMsg{Time: t, ID: m.spinner.ID()}
	})
}

func (m processingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case statusMsg:
		m.label = string(msg)
		// Re-issue readInFifo to consume the next record.
		var nextRead tea.Cmd
		if m.inFifo != "" {
			nextRead = readInFifo(m.inFifo)
		}
		return m, nextRead
	case closeMsg:
		return m, tea.Quit
	case tea.QuitMsg:
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m processingModel) render() string {
	spinFrame := m.spinner.View()
	body := []string{spinFrame + " " + m.label}
	hint := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Muted)).Render("processing…")
	return renderFrame(m.theme, "default", m.title, body, hint, m.width, 1, 1)
}

func (m processingModel) View() tea.View {
	return tea.NewView(m.render())
}

// readInFifo reads one framed record from path and maps:
//   - "status" → statusMsg(label)
//   - "close"  → closeMsg{}
//
// After each non-close record it re-issues itself so the caller keeps reading.
func readInFifo(path string) tea.Cmd {
	return func() tea.Msg {
		f, err := os.Open(path)
		if err != nil {
			return closeMsg{}
		}
		defer f.Close()

		sc := bufio.NewScanner(f)
		sc.Split(recordSplitFunc)
		if sc.Scan() {
			rec := sc.Text()
			cmd, args := decodeRecord(rec)
			switch cmd {
			case "close":
				return closeMsg{}
			case "status":
				label := ""
				if len(args) > 0 {
					label = strings.Join(args, recUS)
				}
				return statusMsg(label)
			}
		}
		// EOF or unknown record — keep waiting via a re-issued read
		return statusMsg("Processing…")
	}
}

// recordSplitFunc is a bufio.SplitFunc that splits on the RS byte (\x1e).
func recordSplitFunc(data []byte, atEOF bool) (advance int, token []byte, err error) {
	for i, b := range data {
		if b == recRS[0] {
			return i + 1, data[:i+1], nil
		}
	}
	if atEOF && len(data) > 0 {
		return len(data), data, nil
	}
	return 0, nil, nil
}

// writeOutFifo writes a pre-encoded record to path (best-effort; non-blocking).
func writeOutFifo(path, record string) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(record) //nolint:errcheck
}
