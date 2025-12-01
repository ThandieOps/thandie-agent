package main

import (
	"fmt"
	"os"

	"github.com/ThandieOps/thandie-agent/internal/cache"
	"github.com/ThandieOps/thandie-agent/internal/logger"
	"github.com/ThandieOps/thandie-agent/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

// runMainTUI runs the main TUI application
func runMainTUI(workspacePath string) error {
	// Load cached scan results
	cacheInstance, err := cache.New()
	if err != nil {
		return fmt.Errorf("failed to initialize cache: %w", err)
	}

	// Try to load cached results
	scanResult, err := cacheInstance.LoadScanResult(workspacePath)
	if err != nil {
		// No cached results - prompt user to run scan first
		logger.Warn("no cached scan results found", "workspace", workspacePath, "error", err)
		fmt.Fprintf(os.Stderr, "No cached scan results found for workspace: %s\n", workspacePath)
		fmt.Fprintf(os.Stderr, "Please run 'thandie scan --workspace %s' first to scan the workspace.\n", workspacePath)
		os.Exit(1)
	}

	// Get scanner config from global config
	ignoreDirs := []string{".git", "node_modules", "vendor"} // default
	includeHidden := false                                   // default
	if cfg != nil {
		ignoreDirs = cfg.Scanner.IgnoreDirs
		includeHidden = cfg.Scanner.IncludeHidden
	}

	// Create TUI model with cached directories and scanner config
	model := tui.NewMainModel(workspacePath, scanResult.DirectoryInfos, ignoreDirs, includeHidden)

	// Run the TUI program with mouse support
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}

