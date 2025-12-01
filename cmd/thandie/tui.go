package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ThandieOps/thandie-agent/internal/cache"
	"github.com/ThandieOps/thandie-agent/internal/logger"
	"github.com/ThandieOps/thandie-agent/internal/scanner"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// TUIApp manages the main TUI application
type TUIApp struct {
	app           *tview.Application
	workspacePath string
	directories   []scanner.DirectoryInfo
	filteredDirs  []scanner.DirectoryInfo
	filterString  string
	leftPane      *tview.List
	rightPane     *tview.TextView
	flex          *tview.Flex
	pages         *tview.Pages
	scanning      bool
	filterInput   *tview.InputField
	scanCancel    context.CancelFunc
	scanCancelMu  sync.Mutex
}

// NewTUIApp creates a new TUI application instance
func NewTUIApp(workspacePath string) *TUIApp {
	app := tview.NewApplication()

	// Create left pane (directory list) with border
	leftPane := tview.NewList().
		ShowSecondaryText(false).
		SetHighlightFullLine(true).
		SetSelectedBackgroundColor(tview.Styles.MoreContrastBackgroundColor).
		SetSelectedTextColor(tview.Styles.PrimaryTextColor)
	leftPane.SetBorder(true).
		SetTitle(" Directories ")

	// Create right pane (metadata display) with border
	rightPane := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true).
		SetWordWrap(true)
	rightPane.SetBorder(true).
		SetTitle(" Metadata ")

	// Create command bar at bottom
	commandBar := tview.NewTextView().
		SetDynamicColors(true).
		SetText("[dim]s: Scan  /: Filter  g/G: First/Last  q: Quit[white]").
		SetTextAlign(tview.AlignCenter)
	commandBar.SetBorder(false)

	// Create main flex layout with panes and command bar
	mainFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tview.NewFlex().
			AddItem(leftPane, 0, 1, true).
			AddItem(rightPane, 0, 2, false),
			0, 1, true).
		AddItem(commandBar, 1, 0, false)

	// Create pages for switching between main view and scan view
	pages := tview.NewPages().
		AddPage("main", mainFlex, true, true)

	tui := &TUIApp{
		app:           app,
		workspacePath: workspacePath,
		leftPane:      leftPane,
		rightPane:     rightPane,
		flex:          mainFlex,
		pages:         pages,
		filterString:  "",
	}

	// Set up keyboard handlers
	tui.setupKeyboardHandlers()

	// Set up mouse handlers
	tui.setupMouseHandlers()

	// Set up selection handler (called when item is activated/clicked)
	leftPane.SetSelectedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		tui.updateRightPane(index)
	})

	// Set up change handler (called when selection changes via keyboard/mouse)
	leftPane.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		tui.updateRightPane(index)
	})

	// Set the root
	app.SetRoot(pages, true).SetFocus(leftPane)

	return tui
}

// setupKeyboardHandlers sets up keyboard shortcuts
func (t *TUIApp) setupKeyboardHandlers() {
	t.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Check if we're on the scan page
		currentPage, _ := t.pages.GetFrontPage()
		if currentPage == "scan" {
			// On scan page, ESC cancels and returns to main
			if event.Key() == tcell.KeyEsc {
				t.cancelScan()
				return nil
			}
			return event
		}

		// Check if filter input is active
		if t.filterInput != nil && t.app.GetFocus() == t.filterInput {
			// Let filter input handle its own events, but catch ESC to close it
			if event.Key() == tcell.KeyEsc {
				t.closeFilter()
				return nil
			}
			return event
		}

		// Main page shortcuts
		switch event.Key() {
		case tcell.KeyCtrlC:
			t.app.Stop()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 's', 'S':
				if !t.scanning {
					t.startScan()
				}
				return nil
			case 'q', 'Q':
				t.app.Stop()
				return nil
			case 'g':
				// Jump to first entry
				if t.leftPane.GetItemCount() > 0 {
					t.leftPane.SetCurrentItem(0)
				}
				return nil
			case 'G':
				// Jump to last entry
				count := t.leftPane.GetItemCount()
				if count > 0 {
					t.leftPane.SetCurrentItem(count - 1)
				}
				return nil
			case '/':
				// Open filter dialog
				t.openFilter()
				return nil
			}
		}
		return event
	})
}

// setupMouseHandlers enables mouse support
func (t *TUIApp) setupMouseHandlers() {
	t.app.EnableMouse(true)
}

// loadDirectories loads directories from cache or scans if needed
func (t *TUIApp) loadDirectories() error {
	// Try to load from cache first
	cacheInstance, err := cache.New()
	if err != nil {
		logger.Warn("failed to initialize cache", "error", err)
		return t.scanDirectories()
	}

	result, err := cacheInstance.LoadScanResult(t.workspacePath)
	if err != nil {
		logger.Debug("no cached scan result, scanning", "error", err)
		return t.scanDirectories()
	}

	t.directories = result.DirectoryInfos
	t.filteredDirs = t.directories
	t.applyFilter()
	return nil
}

// scanDirectories performs a scan and updates the UI
func (t *TUIApp) scanDirectories() error {
	// Get scanner config
	ignoreDirs := []string{".git", "node_modules", "vendor"}
	includeHidden := false
	if cfg != nil {
		ignoreDirs = cfg.Scanner.IgnoreDirs
		includeHidden = cfg.Scanner.IncludeHidden
	}

	// Scan directories
	dirInfos, err := scanner.ScanDirectoriesWithMetadata(t.workspacePath, ignoreDirs, includeHidden)
	if err != nil {
		return fmt.Errorf("failed to scan workspace: %w", err)
	}

	t.directories = dirInfos
	t.filteredDirs = t.directories
	t.applyFilter()

	// Save to cache
	cacheInstance, err := cache.New()
	if err == nil {
		if err := cacheInstance.SaveScanResultWithMetadata(t.workspacePath, dirInfos); err != nil {
			logger.Warn("failed to save scan results to cache", "error", err)
		}
	}

	return nil
}

// startScan launches the scan TUI
func (t *TUIApp) startScan() {
	if t.scanning {
		return
	}
	t.scanning = true

	// Create a cancelable context for the scan
	ctx, cancel := context.WithCancel(context.Background())
	t.scanCancelMu.Lock()
	t.scanCancel = cancel
	t.scanCancelMu.Unlock()

	// Create scan TUI components
	scanProgress := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true).
		SetWordWrap(true).
		SetScrollable(true)

	headerText := tview.NewTextView().
		SetDynamicColors(true).
		SetText("[yellow]Scanning workspace...[white]\nPress ESC to cancel").
		SetTextAlign(tview.AlignCenter)

	scanFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(headerText, 3, 0, false).
		AddItem(scanProgress, 0, 1, false)

	// Add border around the scan TUI
	scanFlex.SetBorder(true).
		SetTitle(" Scan Progress ")

	// Add scan page
	t.pages.AddPage("scan", scanFlex, true, true)
	t.app.SetFocus(scanProgress)

	// Run scan in goroutine
	go func() {
		t.performScanWithProgress(ctx, scanProgress)

		// Check if scan was canceled before switching back
		if ctx.Err() == nil {
			// Switch back to main view only if scan completed normally
			t.app.QueueUpdateDraw(func() {
				currentPage, _ := t.pages.GetFrontPage()
				if currentPage == "scan" {
					t.pages.SwitchToPage("main")
					t.app.SetFocus(t.leftPane)
				}
				t.scanning = false
			})
		}
	}()
}

// cancelScan cancels an ongoing scan and returns to main view
func (t *TUIApp) cancelScan() {
	t.scanCancelMu.Lock()
	if t.scanCancel != nil {
		t.scanCancel()
		t.scanCancel = nil
	}
	t.scanCancelMu.Unlock()

	t.app.QueueUpdateDraw(func() {
		t.pages.SwitchToPage("main")
		t.app.SetFocus(t.leftPane)
		t.scanning = false
	})
}

// performScanWithProgress performs the scan and updates the progress text view
func (t *TUIApp) performScanWithProgress(ctx context.Context, progressText *tview.TextView) {
	updateProgress := func(message string) {
		// Check if canceled before updating
		if ctx.Err() != nil {
			return
		}
		t.app.QueueUpdateDraw(func() {
			currentText := progressText.GetText(false)
			if currentText != "" {
				currentText += "\n"
			}
			progressText.SetText(currentText + message)
			progressText.ScrollToEnd()
		})
	}

	updateProgress(fmt.Sprintf("[yellow]Scanning workspace: %s[white]", t.workspacePath))
	updateProgress("")

	// Get scanner config
	ignoreDirs := []string{".git", "node_modules", "vendor"}
	includeHidden := false
	if cfg != nil {
		ignoreDirs = cfg.Scanner.IgnoreDirs
		includeHidden = cfg.Scanner.IncludeHidden
	}

	// Read directory entries
	entries, err := os.ReadDir(t.workspacePath)
	if err != nil {
		updateProgress(fmt.Sprintf("[red]Error reading directory: %v[white]", err))
		return
	}

	// Create ignore map
	ignoreMap := make(map[string]bool)
	for _, dir := range ignoreDirs {
		ignoreMap[dir] = true
	}

	var dirInfos []scanner.DirectoryInfo
	repoCount := 0
	uncommittedCount := 0

	// Scan each directory individually to show progress
	for _, entry := range entries {
		// Check if canceled
		if ctx.Err() != nil {
			updateProgress("[yellow]Scan canceled[white]")
			return
		}

		if !entry.IsDir() {
			continue
		}

		dirName := entry.Name()

		// Skip hidden directories if includeHidden is false
		if !includeHidden && len(dirName) > 0 && dirName[0] == '.' {
			continue
		}

		// Skip directories in the ignore list
		if ignoreMap[dirName] {
			continue
		}

		dirPath := filepath.Join(t.workspacePath, dirName)

		// Show "Scanning <dir>"
		updateProgress(fmt.Sprintf("[cyan]Scanning %s[white]", dirName))

		// Check if canceled before collecting metadata
		if ctx.Err() != nil {
			updateProgress("[yellow]Scan canceled[white]")
			return
		}

		// Collect git metadata
		gitMetadata, err := scanner.CollectGitMetadata(dirPath)
		if err != nil {
			logger.Debug("failed to collect git metadata", "path", dirPath, "error", err)
			gitMetadata = &scanner.GitMetadata{IsGitRepo: false}
		}

		// Check if canceled after collecting metadata
		if ctx.Err() != nil {
			updateProgress("[yellow]Scan canceled[white]")
			return
		}

		dirInfo := scanner.DirectoryInfo{
			Path:        dirPath,
			GitMetadata: gitMetadata,
		}
		dirInfos = append(dirInfos, dirInfo)

		// Count statistics
		if gitMetadata != nil && gitMetadata.IsGitRepo {
			repoCount++
			if gitMetadata.HasUncommitted {
				uncommittedCount++
			}
		}

		// Show "Completed scan of <dir>"
		updateProgress(fmt.Sprintf("[green]Completed scan of %s[white]", dirName))
		updateProgress("")
	}

	// Check if canceled before saving
	if ctx.Err() != nil {
		updateProgress("[yellow]Scan canceled[white]")
		return
	}

	// Save to cache
	updateProgress("[yellow]Saving results to cache...[white]")
	cacheInstance, err := cache.New()
	if err == nil {
		if err := cacheInstance.SaveScanResultWithMetadata(t.workspacePath, dirInfos); err != nil {
			logger.Warn("failed to save scan results to cache", "error", err)
			updateProgress("[red]Warning: Failed to save cache[white]")
		} else {
			updateProgress("[green]Results saved to cache[white]")
		}
	}
	updateProgress("")

	// Check if canceled before updating UI
	if ctx.Err() != nil {
		return
	}

	// Update directories and UI
	t.directories = dirInfos
	t.filteredDirs = t.directories
	t.app.QueueUpdateDraw(func() {
		t.applyFilter()
	})

	// Display summary
	totalDirs := len(dirInfos)
	var summary strings.Builder
	summary.WriteString("[green]═══════════════════════════════════════[white]\n")
	summary.WriteString("[green]           Scan Summary[white]\n")
	summary.WriteString("[green]═══════════════════════════════════════[white]\n\n")
	summary.WriteString(fmt.Sprintf("  Directories scanned: [yellow]%d[white]\n", totalDirs))
	summary.WriteString(fmt.Sprintf("  Git repositories:    [cyan]%d[white]\n", repoCount))
	summary.WriteString(fmt.Sprintf("  With uncommitted:    [red]%d[white]\n", uncommittedCount))
	summary.WriteString("\n[green]═══════════════════════════════════════[white]\n")
	summary.WriteString("\n[dim]Press ESC to return to main view[white]")

	updateProgress(summary.String())
}

// updateLeftPane updates the left pane with directory names
func (t *TUIApp) updateLeftPane() {
	t.leftPane.Clear()

	// Use filtered directories if filter is active, otherwise use all directories
	dirsToShow := t.directories
	if t.filterString != "" {
		dirsToShow = t.filteredDirs
	}

	for _, dir := range dirsToShow {
		// Extract just the directory name from the path
		dirName := filepath.Base(dir.Path)
		t.leftPane.AddItem(dirName, "", 0, nil)
	}

	// Select first item if available
	if len(dirsToShow) > 0 {
		t.leftPane.SetCurrentItem(0)
		t.updateRightPane(0)
	}
}

// openFilter opens the filter input dialog
func (t *TUIApp) openFilter() {
	// Create filter input field
	filterInput := tview.NewInputField().
		SetLabel("Filter: ").
		SetFieldWidth(30).
		SetText(t.filterString).
		SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyEnter || key == tcell.KeyEsc {
				t.closeFilter()
			}
		})

	// Set up change handler for incremental filtering
	filterInput.SetChangedFunc(func(text string) {
		t.filterString = text
		t.applyFilter()
	})

	// Create flex layout for filter (centered overlay)
	filterFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(filterInput, 40, 0, true).
			AddItem(nil, 0, 1, false),
			3, 0, true).
		AddItem(nil, 0, 1, false)

	t.filterInput = filterInput
	t.pages.AddPage("filter", filterFlex, true, true)
	t.app.SetFocus(filterInput)
}

// closeFilter closes the filter dialog
func (t *TUIApp) closeFilter() {
	if t.filterInput != nil {
		t.pages.RemovePage("filter")
		t.filterInput = nil
		t.app.SetFocus(t.leftPane)
	}
}

// applyFilter applies the current filter string to directories
func (t *TUIApp) applyFilter() {
	if t.filterString == "" {
		t.filteredDirs = t.directories
	} else {
		filterLower := strings.ToLower(t.filterString)
		t.filteredDirs = []scanner.DirectoryInfo{}
		for _, dir := range t.directories {
			dirName := strings.ToLower(filepath.Base(dir.Path))
			if strings.Contains(dirName, filterLower) {
				t.filteredDirs = append(t.filteredDirs, dir)
			}
		}
	}
	t.updateLeftPane()
}

// updateRightPane updates the right pane with metadata for the selected directory
func (t *TUIApp) updateRightPane(index int) {
	// Use filtered directories if filter is active, otherwise use all directories
	dirsToShow := t.directories
	if t.filterString != "" {
		dirsToShow = t.filteredDirs
	}

	if index < 0 || index >= len(dirsToShow) {
		t.rightPane.Clear()
		return
	}

	dir := dirsToShow[index]
	var content strings.Builder

	// Directory path
	content.WriteString("[yellow]Path:[white]\n")
	content.WriteString(dir.Path)
	content.WriteString("\n\n")

	// Git metadata
	if dir.GitMetadata != nil && dir.GitMetadata.IsGitRepo {
		content.WriteString("[yellow]Git Repository:[white]\n")

		if dir.GitMetadata.CurrentBranch != "" {
			content.WriteString(fmt.Sprintf("Branch: [green]%s[white]\n", dir.GitMetadata.CurrentBranch))
		}

		if dir.GitMetadata.RemoteURL != "" {
			content.WriteString(fmt.Sprintf("Remote: %s\n", dir.GitMetadata.RemoteURL))
		}

		if dir.GitMetadata.HasUncommitted {
			content.WriteString("\n[yellow]Uncommitted Files:[white]\n")

			// Parse StatusSummary to extract file list
			statusSummary := dir.GitMetadata.StatusSummary
			if statusSummary != "" && statusSummary != "clean" {
				// Check if there's a "... (X more)" pattern
				morePattern := "... ("
				moreIdx := strings.Index(statusSummary, morePattern)
				var moreCount int
				var fileListStr string

				if moreIdx != -1 {
					// Extract the file list part (before "... (X more)")
					fileListStr = statusSummary[:moreIdx]
					// Extract the count
					morePart := statusSummary[moreIdx+len(morePattern):]
					morePart = strings.TrimSuffix(morePart, " more)")
					fmt.Sscanf(morePart, "%d", &moreCount)
				} else {
					fileListStr = statusSummary
				}

				// Split by semicolon to get individual file entries
				parts := strings.Split(fileListStr, "; ")
				fileCount := 0

				for _, part := range parts {
					part = strings.TrimSpace(part)
					if part == "" {
						continue
					}
					// Extract filename from status line (format: "XY filename")
					fields := strings.Fields(part)
					if len(fields) >= 2 {
						filename := strings.Join(fields[1:], " ")
						content.WriteString(fmt.Sprintf("  %s\n", filename))
						fileCount++
					} else if len(fields) == 1 {
						// Sometimes just the filename
						content.WriteString(fmt.Sprintf("  %s\n", fields[0]))
						fileCount++
					}
				}

				// Show "... (X more)" on a separate line if there are more files
				if moreCount > 0 {
					content.WriteString(fmt.Sprintf("\n[dim]... (%d more)[white]\n", moreCount))
				}
			}
		} else {
			content.WriteString("\n[green]No uncommitted changes[white]\n")
		}
	} else {
		content.WriteString("[dim]Not a git repository[white]\n")
	}

	t.rightPane.SetText(content.String())
}

// Run starts the TUI application
func (t *TUIApp) Run() error {
	// Load directories
	if err := t.loadDirectories(); err != nil {
		return err
	}

	return t.app.Run()
}
