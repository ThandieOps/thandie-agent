package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

var (
	// Logger is the global logger instance
	Logger  *slog.Logger
	logFile *os.File
)

// Init initializes the logger with the specified level, format, and file output
func Init(level string, jsonOutput bool, logToFile bool) error {
	var logLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	var writer io.Writer = os.Stderr // Default to stderr

	// If logging to file, set up file writer
	if logToFile {
		logPath, err := getLogFilePath()
		if err != nil {
			return fmt.Errorf("failed to determine log file path: %w", err)
		}

		// Ensure log directory exists
		logDir := filepath.Dir(logPath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return fmt.Errorf("failed to create log directory %s: %w", logDir, err)
		}

		// Open log file in append mode
		logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to open log file %s: %w", logPath, err)
		}

		// Write to both file and stderr
		writer = io.MultiWriter(os.Stderr, logFile)
	}

	var handler slog.Handler
	if jsonOutput {
		handler = slog.NewJSONHandler(writer, opts)
	} else {
		handler = slog.NewTextHandler(writer, opts)
	}

	Logger = slog.New(handler)
	return nil
}

// getLogFilePath returns the platform-appropriate log file path
func getLogFilePath() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		// Fallback to home directory if cache dir unavailable
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		cacheDir = filepath.Join(homeDir, ".cache")
	}

	logDir := filepath.Join(cacheDir, "thandie", "logs")
	logFile := filepath.Join(logDir, "thandie.log")
	return logFile, nil
}

// GetLogFilePath returns the expected log file path (for debugging)
// This is useful to check where logs would be written without actually creating the file
func GetLogFilePath() (string, error) {
	return getLogFilePath()
}

// Sync flushes the log file to disk if it was opened
func Sync() error {
	if logFile != nil {
		return logFile.Sync()
	}
	return nil
}

// Close closes the log file if it was opened
func Close() error {
	if logFile != nil {
		return logFile.Close()
	}
	return nil
}

// Debug logs a debug message
func Debug(msg string, args ...any) {
	if Logger != nil {
		Logger.Debug(msg, args...)
	}
}

// Info logs an info message
func Info(msg string, args ...any) {
	if Logger != nil {
		Logger.Info(msg, args...)
	}
}

// Warn logs a warning message
func Warn(msg string, args ...any) {
	if Logger != nil {
		Logger.Warn(msg, args...)
	}
}

// Error logs an error message
func Error(msg string, args ...any) {
	if Logger != nil {
		Logger.Error(msg, args...)
	}
}
