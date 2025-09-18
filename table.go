// table.go
package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// readLatestTimenetJSON finds and reads the most recent timenet JSON file
func readLatestTimenetJSON() (*TimenetData, error) {
	tempDir := os.TempDir()
	//tempDir = "" // DEBUG - using current directory

	// Find all timenet JSON files
	pattern := filepath.Join(tempDir, "timenet_data_*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to search for JSON files: %v", err)
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no timenet JSON files found in %s", tempDir)
	}

	// Sort files by modification time (newest first)
	sort.Slice(matches, func(i, j int) bool {
		infoI, errI := os.Stat(matches[i])
		infoJ, errJ := os.Stat(matches[j])
		if errI != nil || errJ != nil {
			return false
		}
		return infoI.ModTime().After(infoJ.ModTime())
	})

	latestFile := matches[0]

	slog.Info("Loading latest JSON file: " + latestFile)

	// Read and parse the JSON file
	jsonData, err := os.ReadFile(latestFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %v", latestFile, err)
	}

	var data TimenetData
	err = json.Unmarshal(jsonData, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON from %s: %v", latestFile, err)
	}

	return &data, nil
}

func BuildTimenetTable() string {

	data, err := readLatestTimenetJSON()
	if err != nil {
		return ""
	}

	var result strings.Builder
	result.WriteString("========== Timenet Summary ============\n")
	result.WriteString(fmt.Sprintf(" Last Update:      %s %s\n", redStyle.Render(data.Date), redStyle.Render(data.Time)))
	result.WriteString(fmt.Sprintf(" Reporting Month:  %s\n", data.Summary.MesAno))
	result.WriteString(fmt.Sprintf(" Required Hours:   %s\n", data.Summary.HorasPrevistas))
	result.WriteString(fmt.Sprintf(" Clocked Hours:    %s\n", data.Summary.HorasTrabajadas))
	result.WriteString(fmt.Sprintf(" Total Overtime:   %s\n", data.Summary.AcumuladoAno))
	result.WriteString("=======================================")

	return result.String()
}
