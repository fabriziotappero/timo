package main

import (
	"log/slog"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

var Version = "v0.3.0"

func logInit() {

	var debugMode bool

	for _, arg := range os.Args[1:] {
		if arg == "--debug" {
			debugMode = true
			break
		}
	}

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
			os.Symlink(logFilePath, "timo_debug.log")
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

func checkVersionFlag() {
	for _, arg := range os.Args[1:] {
		if arg == "--version" {
			println(Version)
			os.Exit(0)
		}
	}
}

func main() {

	checkVersionFlag()

	// to get debug info use: go run . --debug
	logInit()

	model := newModel()
	p := tea.NewProgram(model)

	// this process will run in background and send messages to the UI
	go func() {
		for {
			time.Sleep(500)
			//p.Send(resultMsg{some_text: task1(), some_num: 23})
		}
	}()

	// start UI
	if _, err := p.Run(); err != nil {
		slog.Error("Error running program:", err)
		os.Exit(1)
	}
}

func task1() string {
	food := []string{"some cashews", "some ramen"}
	return food[1] // nolint:gosec
}

func task2() string {
	return "Sending some text"
}

func fetchTimenet(password string) error {
	slog.Info("Starting Timenet scraping")
	// _html, err := scrapeTimenet(password)
	// if err != nil {
	// 	slog.Error("Failed to scrape Timenet", "error", err)
	// 	return err
	// }
	return nil
}
