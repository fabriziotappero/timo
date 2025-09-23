package main

import (
	"log/slog"
	"os"
	"path/filepath"
)

// since the UI is using stdout/stderr for display, we cannot log there
// so we create a log file in the OS temp folder
func logInit(debugMode bool) {
	var logger *slog.Logger
	if debugMode {
		// Create log file in OS temp folder
		tempDir := os.TempDir()
		logFilePath := filepath.Join(tempDir, "timo_debug.log")
		logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			// Fallback to stderr if file creation fails
			logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
				Level: slog.LevelWarn,
			}))
		} else {
			logger = slog.New(slog.NewTextHandler(logFile, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			}))
			// Create a simple log file link in current directory
			//os.Symlink(logFilePath, "timo_debug.log")
		}
		logger.Info("Running in DEBUG mode", "log_file", logFilePath)
	} else {
		// For non-debug mode, create a minimal logger that discards output
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelError, // Only show errors
		}))
	}
	slog.SetDefault(logger)
}
