package main

import (
	"log/slog"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yosssi/gohtml"
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
		if arg == "--test" {
			//testTimenetParsing()
			//testKimaiParsing()
			test_all()
			return
		}
	}
	logInit(debugMode)

	setupScraper()

	model := newModel()
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {

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

	// clean up HTML
	cleanHTML(&_html)

	// DEBUG
	if false {
		_html = gohtml.Format(_html)
		os.WriteFile("dump.html", []byte(_html), 0644)
	}

	// PARSE HTML AND SAVE IN LOCAL JSON
	slog.Info("Starting Timenet data parsing")
	err = timenetParse(&_html)
	if err != nil {
		slog.Error("Failed to parse Timenet data", "error", err)
		return err
	}

	return nil
}

func fetchKimai(id string, password string) error {

	// SCRAPING
	slog.Info("Starting Kimai scraping")
	_html, err := scrapeKimai(id, password)
	if err != nil {
		slog.Error("Failed to scrape Kimai", "error", err)
		return err
	}

	// DEBUG
	if false {
		_html = gohtml.Format(_html)
		os.WriteFile("dump.html", []byte(_html), 0644)
	}

	// PARSE HTML AND SAVE IN LOCAL JSON
	slog.Info("Starting Kimai data parsing")
	err = kimaiParse(&_html)
	if err != nil {
		slog.Error("Failed to parse Kimai data", "error", err)
		return err
	}

	return nil
}
