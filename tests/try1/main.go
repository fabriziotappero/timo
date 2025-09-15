package main

import (
	"log/slog"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

type LoginData struct {
	TimenetPassword string
	KimayID         string
	KimaiPassword   string
}

func logInit() {

	var debugMode bool
	// debugMode := os.Getenv("DEBUG") != ""

	for _, arg := range os.Args[1:] {
		if arg == "--debug" {
			debugMode = true
			break
		}
	}

	var logger *slog.Logger
	if debugMode {
		// Create/open a debug log file instead of stdout to avoid UI interference
		logFile, err := os.OpenFile("debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			// Fallback to stderr if file creation fails
			logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
				Level: slog.LevelWarn,
			}))
		} else {
			logger = slog.New(slog.NewTextHandler(logFile, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			}))
		}
		logger.Info("Running in DEBUG mode")
	} else {
		// For non-debug mode, create a minimal logger that discards output
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelError, // Only show errors
		}))
	}
	slog.SetDefault(logger)
}

func main() {

	logInit() // to get debug info use: go run . --debug

	p := tea.NewProgram(initialModel())
	_, err := p.Run()
	if err != nil {
		slog.Error("Could not start program:", err)
		os.Exit(1)
	}
}
