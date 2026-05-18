package tui

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// RunWithSpinner runs `work` in a goroutine and shows a spinner while it
// progresses. The spinner reports `label` and updates to the latest
// progress string sent through the returned channel from `work`.
//
// `work(ctx, log)` returns nil on success; any error is rendered as a final
// failure line. The function blocks until work completes (or fails) and
// prints a single ✓/✗ summary line afterwards.
func RunWithSpinner(label string, work func(ctx context.Context, log func(string)) error) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logs := make(chan string, 32)
	resultCh := make(chan error, 1)

	go func() {
		resultCh <- work(ctx, func(s string) {
			select {
			case logs <- s:
			default:
			}
		})
		close(logs)
	}()

	m := spinnerModel{
		label:  label,
		spin:   spinner.New(),
		logs:   logs,
		result: resultCh,
	}
	m.spin.Spinner = spinner.Dot
	m.spin.Style = Brand

	p := tea.NewProgram(m)
	finalRaw, err := p.Run()
	if err != nil {
		return err
	}
	final := finalRaw.(spinnerModel)
	if final.err != nil {
		fmt.Println(X.Render(IconErr) + " " + label + Dim.Render(" — "+final.err.Error()))
		return final.err
	}
	if final.lastLog != "" {
		fmt.Println(OK.Render(IconOK) + " " + label + Dim.Render(" — "+final.lastLog))
	} else {
		fmt.Println(OK.Render(IconOK) + " " + label)
	}
	return nil
}

type spinnerModel struct {
	mu      sync.Mutex
	label   string
	spin    spinner.Model
	logs    chan string
	result  chan error
	lastLog string
	err     error
	done    bool
}

type logMsg string
type doneMsg struct{ err error }

func (m spinnerModel) Init() tea.Cmd {
	return tea.Batch(m.spin.Tick, m.pollLog, m.pollDone)
}

func (m spinnerModel) pollLog() tea.Msg {
	s, ok := <-m.logs
	if !ok {
		return nil
	}
	return logMsg(s)
}

func (m spinnerModel) pollDone() tea.Msg {
	err := <-m.result
	return doneMsg{err: err}
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.KeyMsg:
		if v.String() == "ctrl+c" {
			m.err = fmt.Errorf("cancelled")
			return m, tea.Quit
		}
	case logMsg:
		m.lastLog = string(v)
		return m, m.pollLog
	case doneMsg:
		m.err = v.err
		m.done = true
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m spinnerModel) View() string {
	if m.done {
		return ""
	}
	line := m.spin.View() + " " + m.label
	if m.lastLog != "" {
		line += Dim.Render("  " + m.lastLog)
	}
	return line
}

// WaitBriefly is a tiny helper so RunWithSpinner's caller can simulate work in tests.
func WaitBriefly() { time.Sleep(120 * time.Millisecond) }
