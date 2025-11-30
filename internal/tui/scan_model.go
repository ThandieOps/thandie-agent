package tui

import (
	"fmt"
	"strings"

	"github.com/ThandieOps/thandie-agent/internal/scanner"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	padding  = 2
	maxWidth = 80
)

var (
	titleStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Right = "├"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)
	}()

	infoStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return titleStyle.Copy().BorderStyle(b)
	}()
)

// ProgressMsg is sent when progress updates
type ProgressMsg struct {
	Current int
	Total   int
	Message string
}

// LogMsg is sent when a log message should be displayed
type LogMsg string

// ScanCompleteMsg is sent when scanning is complete
type ScanCompleteMsg struct {
	Results []scanner.DirectoryInfo
	Error   error
}

// Model represents the TUI state
type Model struct {
	workspacePath string
	ignoreDirs    []string
	includeHidden bool

	progress    progress.Model
	logViewport viewport.Model
	logs        []string
	current     int
	total       int
	currentMsg  string
	width       int
	height      int
	scanning    bool
	completed   bool
	error       error
	results     []scanner.DirectoryInfo
}

// NewModel creates a new TUI model
func NewModel(workspacePath string, ignoreDirs []string, includeHidden bool) Model {
	p := progress.New(progress.WithScaledGradient("#FF6B6B", "#4ECDC4"))
	logViewport := viewport.New(0, 0)

	m := Model{
		workspacePath: workspacePath,
		ignoreDirs:    ignoreDirs,
		includeHidden: includeHidden,
		progress:      p,
		logViewport:   logViewport,
		logs:          []string{},
		scanning:      true,
		width:         maxWidth,
		height:        24,
	}

	return m
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.WindowSize()
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progress.Width = msg.Width - padding*2 - 4
		if m.progress.Width > maxWidth {
			m.progress.Width = maxWidth
		}
		if m.progress.Width < 20 {
			m.progress.Width = 20
		}

		// Set viewport height (leave room for progress bar, title, and padding)
		// Title: 1 line, Progress bar: 3 lines (bar + text + spacing), Footer: 2 lines
		headerHeight := 6
		footerHeight := 2
		viewportHeight := msg.Height - headerHeight - footerHeight
		if viewportHeight < 5 {
			viewportHeight = 5
		}
		m.logViewport.Height = viewportHeight
		// Viewport width accounts for border (2 chars) and padding (2 chars on each side = 4)
		viewportWidth := msg.Width - padding*2 - 4
		if viewportWidth < 20 {
			viewportWidth = 20
		}
		m.logViewport.Width = viewportWidth

		return m, nil

	case ProgressMsg:
		m.current = msg.Current
		m.total = msg.Total
		m.currentMsg = msg.Message
		if msg.Message != "" {
			m.addLog(msg.Message)
		}
		return m, nil

	case LogMsg:
		m.addLog(string(msg))
		return m, nil

	case ScanCompleteMsg:
		m.scanning = false
		m.completed = true
		if msg.Error != nil {
			m.error = msg.Error
			m.addLog(fmt.Sprintf("Error: %v", msg.Error))
		} else {
			m.results = msg.Results
			m.addLog(fmt.Sprintf("Scan completed: %d directories found", len(msg.Results)))
		}
		return m, tea.Quit

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}
		if !m.scanning {
			return m, tea.Quit
		}
	}

	// Update progress bar
	var progressCmd tea.Cmd
	progressModel, progressCmd := m.progress.Update(msg)
	if p, ok := progressModel.(progress.Model); ok {
		m.progress = p
	}

	// Update viewport
	var viewportCmd tea.Cmd
	m.logViewport, viewportCmd = m.logViewport.Update(msg)

	return m, tea.Batch(progressCmd, viewportCmd)
}

// View renders the UI
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var sections []string

	// Title
	title := titleStyle.Render("Scanning Workspace")
	info := infoStyle.Render(fmt.Sprintf("Scanning: %s", m.workspacePath))
	titleBar := lipgloss.JoinHorizontal(lipgloss.Center, title, info)
	sections = append(sections, titleBar)

	// Progress bar
	if m.total > 0 {
		percent := float64(m.current) / float64(m.total)
		progressBar := m.progress.ViewAs(percent)
		progressText := fmt.Sprintf("Progress: %d/%d directories", m.current, m.total)
		if m.currentMsg != "" {
			progressText += fmt.Sprintf(" - %s", m.currentMsg)
		}
		sections = append(sections, "")
		sections = append(sections, progressBar)
		sections = append(sections, progressText)
	} else {
		sections = append(sections, "")
		sections = append(sections, "Initializing scan...")
	}

	// Log viewport
	sections = append(sections, "")
	logContent := strings.Join(m.logs, "\n")
	m.logViewport.SetContent(logContent)
	// The box width should be viewport width + border (2) + padding (4 total)
	boxWidth := m.logViewport.Width + 4
	if boxWidth > m.width-padding*2 {
		boxWidth = m.width - padding*2
	}
	logBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(boxWidth).
		Height(m.logViewport.Height + 2). // +2 for border
		Render(m.logViewport.View())
	sections = append(sections, logBox)

	// Footer
	footer := "Press 'q' or Ctrl+C to quit"
	if m.scanning {
		footer = "Scanning in progress... Press 'q' or Ctrl+C to quit"
	}
	sections = append(sections, "")
	sections = append(sections, footer)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// addLog adds a log message to the buffer and scrolls to bottom
func (m *Model) addLog(msg string) {
	m.logs = append(m.logs, msg)
	// Keep only last 1000 log lines to prevent memory issues
	if len(m.logs) > 1000 {
		m.logs = m.logs[len(m.logs)-1000:]
	}
	// Scroll to bottom
	m.logViewport.SetContent(strings.Join(m.logs, "\n"))
	m.logViewport.GotoBottom()
}

// GetResults returns the scan results
func (m Model) GetResults() []scanner.DirectoryInfo {
	return m.results
}

// GetError returns any error that occurred
func (m Model) GetError() error {
	return m.error
}
