package tui

import (
	"path/filepath"
	"strings"

	"github.com/ThandieOps/thandie-agent/internal/cache"
	"github.com/ThandieOps/thandie-agent/internal/scanner"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	mainTitleStyle        = lipgloss.NewStyle().MarginLeft(2)
	mainItemStyle         = lipgloss.NewStyle().PaddingLeft(2)
	mainSelectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	mainPathStyle         = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	mainLabelStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	mainValueStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
)

// DirectoryItem represents an item in the directory list
type DirectoryItem struct {
	Info scanner.DirectoryInfo
}

func (d DirectoryItem) FilterValue() string {
	return filepath.Base(d.Info.Path)
}

func (d DirectoryItem) Title() string {
	return filepath.Base(d.Info.Path)
}

func (d DirectoryItem) Description() string {
	return "" // We only show the directory name
}

// ScanTriggeredMsg is sent when a scan completes and cache should be reloaded
type ScanTriggeredMsg struct{}

// MainModel represents the main TUI state
type MainModel struct {
	workspacePath string
	directories   []scanner.DirectoryInfo
	list          list.Model
	selectedIndex int
	width         int
	height        int
	leftWidth     int
	rightWidth    int
	ignoreDirs    []string
	includeHidden bool
}

// NewMainModel creates a new main TUI model
func NewMainModel(workspacePath string, directories []scanner.DirectoryInfo, ignoreDirs []string, includeHidden bool) MainModel {
	// Create list items from directories
	items := make([]list.Item, len(directories))
	for i, dir := range directories {
		items[i] = DirectoryItem{Info: dir}
	}

	// Create list model
	l := list.New(items, newDirectoryDelegate(), 0, 0)
	l.Title = "Directories"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = mainTitleStyle
	l.Styles.PaginationStyle = lipgloss.NewStyle()
	l.Styles.HelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	l.SetShowHelp(true)
	l.DisableQuitKeybindings() // We handle quit ourselves

	m := MainModel{
		workspacePath: workspacePath,
		directories:   directories,
		list:          l,
		selectedIndex: 0,
		width:         80,
		height:        24,
		leftWidth:     40,
		rightWidth:    40,
		ignoreDirs:    ignoreDirs,
		includeHidden: includeHidden,
	}

	// Set initial selection if directories exist
	if len(directories) > 0 {
		m.selectedIndex = 0
	}

	return m
}

// Init initializes the model
func (m MainModel) Init() tea.Cmd {
	return tea.WindowSize()
}

// Update handles messages and updates the model
func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Split width: 40% left, 60% right
		// Account for borders: each pane has 2 chars border (left + right = 4 total per pane)
		// But borders are rendered separately, so we split the full width
		availableWidth := msg.Width
		m.leftWidth = int(float64(availableWidth) * 0.4)
		m.rightWidth = availableWidth - m.leftWidth
		// List width needs to account for border (2 chars each side = 4 total) and padding
		listWidth := m.leftWidth - 6 // Border (4) + padding (2)
		if listWidth < 10 {
			listWidth = 10
		}
		m.list.SetWidth(listWidth)
		// List height: full height minus header (1 line) and spacing (1) and help
		listHeight := msg.Height - 3
		if listHeight < 5 {
			listHeight = 5
		}
		m.list.SetHeight(listHeight)
		// Ensure list selection is valid after resize
		if m.selectedIndex >= 0 && m.selectedIndex < len(m.directories) {
			m.list.Select(m.selectedIndex)
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "s":
			// Trigger scan
			return m, m.runScan()
		case "enter":
			// Update selected index based on list selection
			if selected := m.list.SelectedItem(); selected != nil {
				item := selected.(DirectoryItem)
				for i, dir := range m.directories {
					if dir.Path == item.Info.Path {
						m.selectedIndex = i
						break
					}
				}
			}
			return m, nil
		}
		// Pass other key messages to the list (for filtering, etc.)
		// Fall through to list.Update below

	case tea.MouseMsg:
		// Pass mouse messages to the list for scrolling and selection
		// The list component handles mouse clicks and wheel scrolling automatically
		// Fall through to list.Update below

	case ScanTriggeredMsg:
		// Reload cache after scan completes
		return m, m.reloadCache()

	case CacheReloadedMsg:
		// Update directories with new scan results
		m.directories = msg.Directories
		// Rebuild list items
		items := make([]list.Item, len(m.directories))
		for i, dir := range m.directories {
			items[i] = DirectoryItem{Info: dir}
		}
		// Create new list with updated items
		m.list.SetItems(items)
		// Reset selection
		if len(m.directories) > 0 {
			m.selectedIndex = 0
			// Select the first item in the list
			m.list.Select(0)
		} else {
			m.selectedIndex = -1
		}
		// Force a window size refresh to ensure full screen redraw
		return m, tea.WindowSize()
	}

	// Update list
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)

	// Update selected index when list selection changes
	if selected := m.list.SelectedItem(); selected != nil {
		item := selected.(DirectoryItem)
		for i, dir := range m.directories {
			if dir.Path == item.Info.Path {
				m.selectedIndex = i
				break
			}
		}
	}

	return m, cmd
}

// View renders the UI
func (m MainModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Header with workspace path (no border)
	header := lipgloss.NewStyle().
		Width(m.width).
		Padding(0, 1).
		Render("Thandie - " + m.workspacePath)

	// Calculate pane height: full height minus header (1 line) and spacing (1)
	paneHeight := m.height - 2

	// Left pane: directory list with border
	leftPaneContent := m.list.View()
	leftPane := lipgloss.NewStyle().
		Width(m.leftWidth).
		Height(paneHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Render(leftPaneContent)

	// Right pane: directory details with border
	// Content width needs to account for border (2 chars each side = 4 total) and padding (1 each side = 2 total)
	rightContentWidth := m.rightWidth - 6 // Border (4) + Padding (2)
	if rightContentWidth < 10 {
		rightContentWidth = 10
	}
	rightPaneContent := m.renderDetailsWithWidth(rightContentWidth)
	rightPane := lipgloss.NewStyle().
		Width(m.rightWidth).
		Height(paneHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 1).
		Render(rightPaneContent)

	// Combine header and panes
	panes := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
	return lipgloss.JoinVertical(lipgloss.Left, header, "", panes)
}

// renderDetails renders the details pane content (without border, border is added in View)
func (m MainModel) renderDetails() string {
	return m.renderDetailsWithWidth(m.rightWidth - 6) // Account for border and padding
}

// renderDetailsWithWidth renders the details pane content with a specific width
func (m MainModel) renderDetailsWithWidth(contentWidth int) string {
	if len(m.directories) == 0 {
		return "No directories found"
	}

	if m.selectedIndex < 0 || m.selectedIndex >= len(m.directories) {
		return "Select a directory"
	}

	dir := m.directories[m.selectedIndex]
	var sections []string

	// Path at top - wrap if needed
	pathLines := wrapText(dir.Path, contentWidth)
	for _, line := range pathLines {
		sections = append(sections, mainPathStyle.Render(line))
	}
	sections = append(sections, "")

	// Git metadata if available
	if dir.GitMetadata != nil && dir.GitMetadata.IsGitRepo {
		sections = append(sections, mainLabelStyle.Render("Git Repository:"))
		sections = append(sections, mainValueStyle.Render("  Yes"))
		sections = append(sections, "")

		if dir.GitMetadata.CurrentBranch != "" {
			sections = append(sections, mainLabelStyle.Render("Branch:"))
			sections = append(sections, mainValueStyle.Render("  "+dir.GitMetadata.CurrentBranch))
			sections = append(sections, "")
		}

		if dir.GitMetadata.RemoteURL != "" {
			sections = append(sections, mainLabelStyle.Render("Remote:"))
			// Wrap remote URL if needed
			remoteLines := wrapText(dir.GitMetadata.RemoteURL, contentWidth-2)
			for _, line := range remoteLines {
				sections = append(sections, mainValueStyle.Render("  "+line))
			}
			sections = append(sections, "")
		}

		if dir.GitMetadata.HasUncommitted {
			sections = append(sections, mainLabelStyle.Render("Status:"))
			sections = append(sections, mainValueStyle.Render("  Uncommitted changes"))
			if dir.GitMetadata.StatusSummary != "" && dir.GitMetadata.StatusSummary != "clean" {
				sections = append(sections, "")
				// Parse status summary and display as bulleted list
				files, moreText := parseStatusSummary(dir.GitMetadata.StatusSummary)
				for _, file := range files {
					sections = append(sections, mainValueStyle.Render("  - "+file))
				}
				// Display "... (X more)" on its own line after the list
				if moreText != "" {
					sections = append(sections, "")
					sections = append(sections, mainValueStyle.Render("  "+moreText))
				}
			}
			sections = append(sections, "")
		} else {
			sections = append(sections, mainLabelStyle.Render("Status:"))
			sections = append(sections, mainValueStyle.Render("  Clean"))
			sections = append(sections, "")
		}
	} else {
		sections = append(sections, mainLabelStyle.Render("Git Repository:"))
		sections = append(sections, mainValueStyle.Render("  No"))
		sections = append(sections, "")
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// parseStatusSummary parses the git status summary and returns a list of filenames and the "more" text
// Format: "XY filename1; XY filename2; ... (X more)"
// Returns: (files, moreText)
func parseStatusSummary(summary string) ([]string, string) {
	if summary == "" || summary == "clean" {
		return []string{}, ""
	}

	var files []string
	var moreText string
	// Split by "; " to get individual entries
	parts := strings.Split(summary, "; ")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Check if this part contains "... (X more)" - it might be attached to the last filename
		moreIdx := strings.Index(part, " ... (")
		if moreIdx > 0 {
			// Split the filename and the "more" text
			filenamePart := part[:moreIdx]
			moreText = part[moreIdx+1:] // Include the "..." part

			// Extract filename from the filename part
			spaceIdx := strings.Index(filenamePart, " ")
			if spaceIdx > 0 && spaceIdx < len(filenamePart)-1 {
				filename := filenamePart[spaceIdx+1:]
				files = append(files, filename)
			} else {
				files = append(files, filenamePart)
			}
			continue
		}

		// Check if this is just the "... (X more)" part (standalone)
		if strings.HasPrefix(part, "...") {
			moreText = part
			continue
		}

		// Format is "XY filename" where XY are status codes
		// Extract filename (everything after the first space)
		spaceIdx := strings.Index(part, " ")
		if spaceIdx > 0 && spaceIdx < len(part)-1 {
			filename := part[spaceIdx+1:]
			files = append(files, filename)
		} else {
			// Fallback: use the whole part if no space found
			files = append(files, part)
		}
	}

	return files, moreText
}

// runScan runs the scan operation and returns a command that sends ScanTriggeredMsg when done
func (m MainModel) runScan() tea.Cmd {
	return func() tea.Msg {
		// Run scan with TUI (this will show the progress bar)
		dirInfos, err := RunScanWithTUI(m.workspacePath, m.ignoreDirs, m.includeHidden)
		if err != nil {
			// Return error message - we could handle this better in the future
			return ScanTriggeredMsg{} // Still reload cache even on error
		}

		// Scan completed successfully, signal to reload cache
		_ = dirInfos // Results are already saved to cache by RunScanWithTUI
		return ScanTriggeredMsg{}
	}
}

// reloadCache reloads the cache and updates the directory list
func (m MainModel) reloadCache() tea.Cmd {
	return func() tea.Msg {
		cacheInstance, err := cache.New()
		if err != nil {
			return nil // Silently fail - cache might not be available
		}

		scanResult, err := cacheInstance.LoadScanResult(m.workspacePath)
		if err != nil {
			return nil // Silently fail - no cached results
		}

		// Return a message with the new directories
		return CacheReloadedMsg{Directories: scanResult.DirectoryInfos}
	}
}

// CacheReloadedMsg is sent when cache is reloaded with new directories
type CacheReloadedMsg struct {
	Directories []scanner.DirectoryInfo
}

// wrapText wraps text to fit within a given width
func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	var lines []string
	currentLine := ""

	for _, word := range words {
		testLine := currentLine
		if testLine != "" {
			testLine += " "
		}
		testLine += word

		if len(testLine) > width {
			if currentLine != "" {
				lines = append(lines, currentLine)
				currentLine = word
			} else {
				// Word is longer than width, just add it
				lines = append(lines, word)
				currentLine = ""
			}
		} else {
			currentLine = testLine
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}

// newDirectoryDelegate creates a delegate for rendering directory items
func newDirectoryDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()
	d.Styles.SelectedTitle = mainSelectedItemStyle
	d.Styles.SelectedDesc = mainSelectedItemStyle
	d.Styles.NormalTitle = mainItemStyle
	d.Styles.NormalDesc = mainItemStyle
	d.ShowDescription = false // Only show directory name
	d.SetSpacing(0)           // Remove spacing between items for compact display
	return d
}
