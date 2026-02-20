package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ProgressModel struct {
	total      int64
	current    int64
	startTime  time.Time
	progress   progress.Model
	filesFound int
	errors     int
	done       bool
	quitting   bool
	width      int
}

type ProgressMsg struct {
	FilesFound int
	Errors     int
	Bytes      int64
}

type DoneMsg struct{}

func NewProgressModel() ProgressModel {
	prog := progress.New(progress.WithDefaultGradient())
	prog.Width = 60

	return ProgressModel{
		progress:  prog,
		startTime: time.Now(),
	}
}

func (m ProgressModel) Init() tea.Cmd {
	return nil
}

func (m ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.progress.Width = min(60, msg.Width-20)
		return m, nil

	case ProgressMsg:
		m.filesFound = msg.FilesFound
		m.errors = msg.Errors
		m.current += msg.Bytes
		return m, nil

	case DoneMsg:
		m.done = true
		return m, tea.Quit

	default:
		var cmd tea.Cmd
		newModel, cmd := m.progress.Update(msg)
		if newProg, ok := newModel.(progress.Model); ok {
			m.progress = newProg
		}
		return m, cmd
	}
}

func (m ProgressModel) View() string {
	if m.quitting {
		return "\nScan interrupted.\n"
	}

	if m.done {
		duration := time.Since(m.startTime).Round(time.Millisecond)
		return fmt.Sprintf(
			"\n✓ Scan complete\n  Files: %d | Errors: %d | Duration: %v\n",
			m.filesFound, m.errors, duration,
		)
	}

	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	b.WriteString(titleStyle.Render("\n🔍 FileHound Scanning...\n\n"))

	if m.total > 0 {
		percent := float64(m.current) / float64(m.total)
		if percent > 1.0 {
			percent = 1.0
		}
		b.WriteString(m.progress.ViewAs(percent) + "\n")
		b.WriteString(fmt.Sprintf("  %.1f MB", float64(m.current)/(1024*1024)))
		if m.total > 0 {
			b.WriteString(fmt.Sprintf(" / %.1f MB", float64(m.total)/(1024*1024)))
		}
		b.WriteString("\n\n")
	}

	statsStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	b.WriteString(statsStyle.Render(fmt.Sprintf(
		"  Files: %d | Errors: %d | Elapsed: %v\n",
		m.filesFound, m.errors, time.Since(m.startTime).Round(time.Second),
	)))

	helpStyle := lipgloss.NewStyle().Faint(true)
	b.WriteString(helpStyle.Render("\n  Press q to quit\n"))

	return b.String()
}

func (m *ProgressModel) Increment(filesFound, errors int, bytes int64) {
	m.filesFound += filesFound
	m.errors += errors
	m.current += bytes
}

func (m *ProgressModel) SetTotal(total int64) {
	m.total = total
}

func (m *ProgressModel) Done() {
	m.done = true
}

type Program struct {
	program *tea.Program
	model   *ProgressModel
}

func NewProgressProgram() *Program {
	model := NewProgressModel()
	return &Program{
		program: tea.NewProgram(model),
		model:   &model,
	}
}

func (p *Program) Start() {
	go func() {
		_, _ = p.program.Run()
	}()
}

func (p *Program) Send(msg interface{}) {
	p.program.Send(msg)
}

func (p *Program) Quit() {
	p.program.Send(DoneMsg{})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
