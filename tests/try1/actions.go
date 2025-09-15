// actions.go
package main

import (
	"log/slog"
	"os"
)

func submitAction(password string) (string, error) {
	slog.Info("Starting Timenet scraping")
	_html, err := scrapeTimenet(password)
	if err != nil {
		slog.Error("Failed to scrape Timenet", "error", err)
		// Try to return existing data if available
		if existingData, tableErr := ShowTimenetTable(); tableErr == nil {
			return existingData, err // Return existing data with the scraping error
		}
		return "", err // No existing data available
	}

	slog.Info("Starting Timenet data parsing")
	err = timenetParse(&_html)
	if err != nil {
		slog.Error("Failed to parse Timenet data", "error", err)
		// Try to return existing data if available
		if existingData, tableErr := ShowTimenetTable(); tableErr == nil {
			return existingData, err
		}
		return "", err
	}

	slog.Info("Cleaning and saving Timenet HTML")
	cleanHTML(&_html)
	err = os.WriteFile("debug.html", []byte(_html), 0644)
	if err != nil {
		slog.Error("Failed to write debug HTML", "error", err)
		// This is not critical, continue with table generation
	}

	slog.Info("Populate Timenet Table")
	tableData, err := ShowTimenetTable()
	if err != nil {
		slog.Error("Failed to show Timenet table", "error", err)
		return "", err
	}

	// Return the fresh table data
	return tableData, nil
}
