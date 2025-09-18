package main

import (
	"log/slog"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {

	// to get debug info use:  go run . --debug
	// track logged data with: tail -f /tmp/timo_debug.log
	debugMode := false
	for _, arg := range os.Args[1:] {
		if arg == "--debug" {
			debugMode = true
			break
		}
	}
	logInit(debugMode)

	// download Chromium if not available
	setupScraper()

	model := newModel()
	p := tea.NewProgram(model)

	// background process
	go func() {
		for {
			time.Sleep(3000 * time.Millisecond)

			// LOAD LOCAL JOSON FILE AND GENERATE TIMENET TABLE
			p.Send(resultMsg{some_text: BuildTimenetTable(), some_num: 0})
		}
	}()

	// START UI
	if _, err := p.Run(); err != nil {
		slog.Error("Error running UI:", err)
		os.Exit(1)
	}
}

func fetchTimenet(password string) error {

	// SCRAPING
	slog.Info("Starting Timenet scraping")
	_html, err := scrapeTimenet(password)
	if err != nil {
		slog.Error("Failed to scrape Timenet", "error", err)
		return err
	}
	slog.Info("Timenet HTML data fetched", "length", len(_html))

	// PARSE HTML AND SAVE IN LOCAL JSON
	slog.Info("Starting Timenet data parsing")
	err = timenetParse(&_html)
	if err != nil {
		slog.Error("Failed to parse Timenet data", "error", err)
		return err
	}

	return nil
}
