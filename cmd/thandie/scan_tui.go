package main

import (
	"fmt"

	"github.com/ThandieOps/thandie-agent/internal/cache"
	"github.com/ThandieOps/thandie-agent/internal/scanner"
	"github.com/ThandieOps/thandie-agent/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

// runScanWithTUI runs the scan operation with a TUI
func runScanWithTUI(wsPath string, ignoreDirs []string, includeHidden bool) ([]scanner.DirectoryInfo, error) {
	// Create channels for communication between scanner and TUI
	progressChan := make(chan tui.ProgressMsg, 10)
	logChan := make(chan tui.LogMsg, 100)
	completeChan := make(chan tui.ScanCompleteMsg, 1)

	// Create TUI model
	model := tui.NewModel(wsPath, ignoreDirs, includeHidden)

	// Set up log capture
	// We'll need to wrap the logger to capture messages
	// For now, we'll send initial log messages manually
	logChan <- tui.LogMsg(fmt.Sprintf("Scanning workspace: %s", wsPath))
	logChan <- tui.LogMsg(fmt.Sprintf("Ignore directories: %v", ignoreDirs))
	logChan <- tui.LogMsg(fmt.Sprintf("Include hidden: %v", includeHidden))

	// Start scanning in a goroutine
	go func() {
		defer close(progressChan)
		defer close(logChan)
		defer close(completeChan)

		// Create progress callback
		progressCallback := func(current, total int, message string) {
			select {
			case progressChan <- tui.ProgressMsg{Current: current, Total: total, Message: message}:
			default:
			}
			// Also send as log message
			if message != "" {
				select {
				case logChan <- tui.LogMsg(message):
				default:
				}
			}
		}

		// Perform the scan
		dirInfos, err := scanner.ScanDirectoriesWithMetadata(wsPath, ignoreDirs, includeHidden, progressCallback)
		if err != nil {
			completeChan <- tui.ScanCompleteMsg{Error: err}
			return
		}

		// Save to cache
		cacheInstance, err := cache.New()
		if err != nil {
			logChan <- tui.LogMsg(fmt.Sprintf("Warning: failed to initialize cache: %v", err))
		} else {
			if err := cacheInstance.SaveScanResultWithMetadata(wsPath, dirInfos); err != nil {
				logChan <- tui.LogMsg(fmt.Sprintf("Warning: failed to save scan results to cache: %v", err))
			} else {
				logChan <- tui.LogMsg(fmt.Sprintf("Scan results cached: %d directories", len(dirInfos)))
			}
		}

		completeChan <- tui.ScanCompleteMsg{Results: dirInfos}
	}()

	// Create a custom model that handles messages from channels
	tuiModel := &scanTUIWrapper{
		model:        model,
		progressChan: progressChan,
		logChan:      logChan,
		completeChan: completeChan,
	}

	// Run the TUI program
	p := tea.NewProgram(tuiModel, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("TUI error: %w", err)
	}

	// Extract results from final model
	wrapper := finalModel.(*scanTUIWrapper)
	if wrapper.model.GetError() != nil {
		return nil, wrapper.model.GetError()
	}

	return wrapper.model.GetResults(), nil
}

// scanTUIWrapper wraps the TUI model to handle channel messages
type scanTUIWrapper struct {
	model        tui.Model
	progressChan <-chan tui.ProgressMsg
	logChan      <-chan tui.LogMsg
	completeChan <-chan tui.ScanCompleteMsg
}

func (w *scanTUIWrapper) Init() tea.Cmd {
	return tea.Batch(
		w.waitForProgress(),
		w.waitForLog(),
		w.waitForComplete(),
	)
}

func (w *scanTUIWrapper) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tui.ProgressMsg:
		updatedModel, cmd := w.model.Update(msg)
		w.model = updatedModel.(tui.Model)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		cmds = append(cmds, w.waitForProgress())

	case tui.LogMsg:
		updatedModel, cmd := w.model.Update(msg)
		w.model = updatedModel.(tui.Model)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		cmds = append(cmds, w.waitForLog())

	case tui.ScanCompleteMsg:
		updatedModel, cmd := w.model.Update(msg)
		w.model = updatedModel.(tui.Model)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		// Don't wait for more messages after completion

	default:
		updatedModel, cmd := w.model.Update(msg)
		w.model = updatedModel.(tui.Model)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return w, tea.Batch(cmds...)
}

func (w *scanTUIWrapper) View() string {
	return w.model.View()
}

func (w *scanTUIWrapper) waitForProgress() tea.Cmd {
	return func() tea.Msg {
		if msg, ok := <-w.progressChan; ok {
			return msg
		}
		return nil
	}
}

func (w *scanTUIWrapper) waitForLog() tea.Cmd {
	return func() tea.Msg {
		if msg, ok := <-w.logChan; ok {
			return msg
		}
		return nil
	}
}

func (w *scanTUIWrapper) waitForComplete() tea.Cmd {
	return func() tea.Msg {
		if msg, ok := <-w.completeChan; ok {
			return msg
		}
		return nil
	}
}

