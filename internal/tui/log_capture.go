package tui

import (
	"context"
	"io"
	"log/slog"
	"sync"
)

// LogCapture is a custom handler that captures log messages and sends them to the TUI
type LogCapture struct {
	handler slog.Handler
	ch      chan<- LogMsg
	mu      sync.Mutex
}

// NewLogCapture creates a new log capture handler
func NewLogCapture(handler slog.Handler, logChan chan<- LogMsg) *LogCapture {
	return &LogCapture{
		handler: handler,
		ch:      logChan,
	}
}

// Enabled reports whether the handler handles records at the given level
func (lc *LogCapture) Enabled(ctx context.Context, level slog.Level) bool {
	return lc.handler.Enabled(ctx, level)
}

// Handle handles the record
func (lc *LogCapture) Handle(ctx context.Context, r slog.Record) error {
	// Format message for TUI display
	msg := r.Message
	if r.NumAttrs() > 0 {
		attrs := make([]string, 0)
		r.Attrs(func(a slog.Attr) bool {
			attrs = append(attrs, a.String())
			return true
		})
		if len(attrs) > 0 {
			msg += " " + joinAttrs(attrs)
		}
	}

	// Send to TUI channel (non-blocking)
	select {
	case lc.ch <- LogMsg(msg):
	default:
		// Channel full, skip this log message
	}

	// Also pass through to original handler
	return lc.handler.Handle(ctx, r)
}

// WithAttrs returns a new handler with the given attributes
func (lc *LogCapture) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &LogCapture{
		handler: lc.handler.WithAttrs(attrs),
		ch:      lc.ch,
	}
}

// WithGroup returns a new handler with the given group
func (lc *LogCapture) WithGroup(name string) slog.Handler {
	return &LogCapture{
		handler: lc.handler.WithGroup(name),
		ch:      lc.ch,
	}
}

// joinAttrs joins attribute strings
func joinAttrs(attrs []string) string {
	if len(attrs) == 0 {
		return ""
	}
	result := ""
	for i, attr := range attrs {
		if i > 0 {
			result += " "
		}
		result += attr
	}
	return result
}

// LogWriter wraps an io.Writer to capture log output
type LogWriter struct {
	writer io.Writer
	ch     chan<- LogMsg
}

// NewLogWriter creates a new log writer that captures output
func NewLogWriter(writer io.Writer, logChan chan<- LogMsg) *LogWriter {
	return &LogWriter{
		writer: writer,
		ch:     logChan,
	}
}

// Write implements io.Writer
func (lw *LogWriter) Write(p []byte) (n int, err error) {
	// Send to TUI channel (non-blocking)
	msg := string(p)
	// Remove trailing newline if present
	if len(msg) > 0 && msg[len(msg)-1] == '\n' {
		msg = msg[:len(msg)-1]
	}
	select {
	case lw.ch <- LogMsg(msg):
	default:
		// Channel full, skip
	}

	// Also write to original writer
	return lw.writer.Write(p)
}
