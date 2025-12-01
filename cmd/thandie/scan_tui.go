package main

import (
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

// ScanTUI manages the scanning progress TUI
type ScanTUI struct {
	app           *tview.Application
	workspacePath string
	progressText  *tview.TextView
	currentDir    string
	mu            sync.Mutex
	done          bool
}

// NewScanTUI creates a new scan TUI instance
// parentApp can be nil if running standalone
func NewScanTUI(workspacePath string, parentApp *tview.Application) *ScanTUI {
	app := tview.NewApplication()

	// Create progress text view - use full screen
	progressText := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true).
		SetWordWrap(true).
		SetScrollable(true)

	// Create flex layout - use full screen
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(progressText, 0, 1, true)

	// Add border around the scan TUI
	flex.SetBorder(true).
		SetTitle(" Scan Progress ")

	scanTUI := &ScanTUI{
		app:           app,
		workspacePath: workspacePath,
		progressText:  progressText,
	}

	// Set up keyboard handlers
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlC, tcell.KeyEsc:
			app.Stop()
			return nil
		case tcell.KeyRune:
			if event.Rune() == 'q' || event.Rune() == 'Q' {
				app.Stop()
				return nil
			}
		}
		return event
	})

	// Set the root
	app.SetRoot(flex, true).SetFocus(progressText)

	return scanTUI
}

// updateProgress updates the progress display
func (s *ScanTUI) updateProgress(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.app.QueueUpdateDraw(func() {
		currentText := s.progressText.GetText(false)
		if currentText != "" {
			currentText += "\n"
		}
		s.progressText.SetText(currentText + message)
		s.progressText.ScrollToEnd()
	})
}

// Run starts the scan TUI and performs the scan
func (s *ScanTUI) Run() {
	// Start scanning in a goroutine
	go s.performScan()

	// Run the TUI
	if err := s.app.Run(); err != nil {
		logger.Error("scan TUI error", "error", err)
	}
}

// performScan performs the actual scanning and shows progress for each directory
func (s *ScanTUI) performScan() {
	s.updateProgress(fmt.Sprintf("[yellow]Scanning workspace: %s[white]\n", s.workspacePath))
	s.updateProgress("")

	// Get scanner config
	ignoreDirs := []string{".git", "node_modules", "vendor"}
	includeHidden := false
	if cfg != nil {
		ignoreDirs = cfg.Scanner.IgnoreDirs
		includeHidden = cfg.Scanner.IncludeHidden
	}

	// Read directory entries
	entries, err := os.ReadDir(s.workspacePath)
	if err != nil {
		s.updateProgress(fmt.Sprintf("[red]Error reading directory: %v[white]", err))
		s.done = true
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

		dirPath := filepath.Join(s.workspacePath, dirName)

		// Show "Scanning <dir>"
		s.updateProgress(fmt.Sprintf("[cyan]Scanning %s[white]", dirName))

		// Collect git metadata
		gitMetadata, err := scanner.CollectGitMetadata(dirPath)
		if err != nil {
			logger.Debug("failed to collect git metadata", "path", dirPath, "error", err)
			gitMetadata = &scanner.GitMetadata{IsGitRepo: false}
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
		s.updateProgress(fmt.Sprintf("[green]Completed scan of %s[white]", dirName))
		s.updateProgress("")
	}

	// Save to cache
	s.updateProgress("[yellow]Saving results to cache...[white]")
	cacheInstance, err := cache.New()
	if err == nil {
		if err := cacheInstance.SaveScanResultWithMetadata(s.workspacePath, dirInfos); err != nil {
			logger.Warn("failed to save scan results to cache", "error", err)
			s.updateProgress("[red]Warning: Failed to save cache[white]")
		} else {
			s.updateProgress("[green]Results saved to cache[white]")
		}
	}
	s.updateProgress("")

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
	summary.WriteString("\n[dim]Press ESC or Ctrl+C to close[white]")

	s.updateProgress(summary.String())
	s.done = true
}
