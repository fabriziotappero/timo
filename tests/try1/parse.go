// parse.go
package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/yosssi/gohtml"
)

// TimenetData represents the parsed timenet data
type TimenetData struct {
	Date      string           `json:"current_date"`
	Time      string           `json:"current_time"`
	Summary   TimenetSummary   `json:"summary"`
	DailyData []TimenetDayData `json:"daily_data"`
}

// TimenetSummary represents the monthly summary
type TimenetSummary struct {
	MesAno          string `json:"mes_ano"`
	HorasPrevistas  string `json:"horas_previstas"`
	HorasTrabajadas string `json:"horas_trabajadas"`
	AcumuladoAno    string `json:"acumulado_ano"`
}

// TimenetDayData represents data for each day
type TimenetDayData struct {
	Day        int    `json:"day"`
	Previstas  string `json:"previstas"`
	Trabajadas string `json:"trabajadas"`
	Diferencia string `json:"diferencia"`
	IsWorkable bool   `json:"is_workable"`
}

// timenetParse extracts data from Timenet HTML and saves to JSON file
func timenetParse(htmlContent *string) error {
	if htmlContent == nil {
		return fmt.Errorf("HTML content is nil")
	}

	data := TimenetData{
		Date: time.Now().Format("2006-01-02"),
		Time: time.Now().Format("15:04:05"),
	}

	// Extract summary data
	summary, err := extractSummary(*htmlContent)
	if err != nil {
		return fmt.Errorf("failed to extract summary: %v", err)
	}
	data.Summary = summary

	// Extract daily data
	dailyData, err := extractDailyData(*htmlContent)
	if err != nil {
		return fmt.Errorf("failed to extract daily data: %v", err)
	}
	data.DailyData = dailyData

	// Save to JSON file
	filename := fmt.Sprintf("timenet_data_%s.json", time.Now().Format("2006-01-02"))
	err = saveToJSON(data, filename)
	if err != nil {
		return fmt.Errorf("failed to save JSON: %v", err)
	}

	slog.Info("Timenet data saved to " + filename)
	return nil
}

// extractSummary extracts the monthly summary from HTML
func extractSummary(html string) (TimenetSummary, error) {
	summary := TimenetSummary{}

	// Extract current month and year from container-date-checks
	MesAnoRe := regexp.MustCompile(`\b([A-Za-z]+ \d{4})\b`)
	MesAnoMatches := MesAnoRe.FindStringSubmatch(html)
	if len(MesAnoMatches) > 1 {
		summary.MesAno = strings.TrimSpace(MesAnoMatches[1])
	}

	// Extract "Horas previstas"
	re := regexp.MustCompile(`<td>Horas previstas:</td>\s*<td>([^<]+)</td>`)
	matches := re.FindStringSubmatch(html)
	if len(matches) > 1 {
		summary.HorasPrevistas = strings.TrimSpace(matches[1])
	}

	// Extract "Horas trabajadas"
	re = regexp.MustCompile(`<td>Horas trabajadas:</td>\s*<td[^>]*>\s*([^<]+)`)
	matches = re.FindStringSubmatch(html)
	if len(matches) > 1 {
		summary.HorasTrabajadas = strings.TrimSpace(matches[1])
	}

	// Extract "Acumulado año"
	re = regexp.MustCompile(`<td class="title-total">Acumulado año:</td>\s*<td[^>]*>[+\-]*\s*<!--[^>]*-->\s*<!--[^>]*-->\s*([^<]+)`)
	matches = re.FindStringSubmatch(html)
	if len(matches) > 1 {
		summary.AcumuladoAno = strings.TrimSpace(matches[1])
	}

	return summary, nil
}

// extractDailyData extracts daily data for each day of the current month
func extractDailyData(html string) ([]TimenetDayData, error) {
	var dailyData []TimenetDayData

	// Split by container-line-checks to get individual day sections
	parts := regexp.MustCompile(`<div class="container-line-checks`).Split(html, -1)

	for i := 1; i < len(parts); i++ { // Skip first empty part
		dayHTML := "<div class=\"container-line-checks" + parts[i]

		// Take first 2000 characters to ensure we get the complete day data
		if len(dayHTML) > 2000 {
			dayHTML = dayHTML[:2000]
		}
		// Extract day number
		dayRe := regexp.MustCompile(`<div class="day-value">(\d+)</div>`)
		dayNumMatches := dayRe.FindStringSubmatch(dayHTML)
		if len(dayNumMatches) < 2 {
			continue
		}

		dayNum, err := strconv.Atoi(dayNumMatches[1])
		if err != nil {
			continue
		}

		dayData := TimenetDayData{
			Day: dayNum,
		}

		// Check if it's a workable day
		isWorkable := strings.Contains(dayHTML, `class="container-line-checks workable-day`)
		dayData.IsWorkable = isWorkable

		if isWorkable {
			// Extract "Previstas" - look for pattern: Previstas: 8h
			preRe := regexp.MustCompile(`Previstas:\s*([^<\n\r]+)`)
			preMatches := preRe.FindStringSubmatch(dayHTML)
			if len(preMatches) > 1 {
				dayData.Previstas = strings.TrimSpace(preMatches[1])
				//fmt.Printf("Found Previstas: '%s'\n", dayData.Previstas)
			} else {
				fmt.Printf("No Previstas match found\n")
			}

			// Extract "Trabajadas" - look for pattern: Trabajadas: 9h 14m
			workedRe := regexp.MustCompile(`Trabajadas:\s*([^<\n\r]+)`)
			workedMatches := workedRe.FindStringSubmatch(dayHTML)
			if len(workedMatches) > 1 {
				trabajadas := strings.TrimSpace(workedMatches[1])
				// Clean up any trailing whitespace or dots
				dayData.Trabajadas = strings.TrimRight(trabajadas, ". \t\n\r")
				//fmt.Printf("Found Trabajadas: '%s'\n", dayData.Trabajadas)
			} else {
				fmt.Printf("No Trabajadas match found\n")
			}

			// Extract "Diferencia" - capture only the time value, excluding HTML comments
			diffRe := regexp.MustCompile(`Diferencia:.*?([+\-]?).*?(\d+[hm])`)
			diffMatches := diffRe.FindStringSubmatch(dayHTML)
			if len(diffMatches) > 2 {
				sign := diffMatches[1]
				time := diffMatches[2]
				dayData.Diferencia = sign + time
				//fmt.Printf("Found Diferencia: '%s'\n", dayData.Diferencia)
			} else {
				//fmt.Printf("No Diferencia match found\n")
			}
		} else {
			// Non-workable day
			dayData.Previstas = "N/A"
			dayData.Trabajadas = "N/A"
			dayData.Diferencia = "N/A"
		}

		dailyData = append(dailyData, dayData)
	}

	return dailyData, nil
}

// Save data to a JSON file in the OS temp folder
func saveToJSON(data TimenetData, filename string) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	tempDir := os.TempDir()
	tempDir = "" // DEBUG
	fullPath := filepath.Join(tempDir, filename)

	err = os.WriteFile(fullPath, jsonData, 0644)
	if err != nil {
		return err
	}

	return nil
}

// cleanHTML removes unwanted elements and formats HTML in place
func cleanHTML(html *string) {
	if html == nil {
		return
	}

	// Remove empty HTML comments
	*html = strings.ReplaceAll(*html, "<!---->", "")

	// Remove script tags and their content
	scriptRe := regexp.MustCompile(`(?s)<script[^>]*>.*?</script>`)
	*html = scriptRe.ReplaceAllString(*html, "")

	// Remove head tags and their content
	headRe := regexp.MustCompile(`(?s)<head[^>]*>.*?</head>`)
	*html = headRe.ReplaceAllString(*html, "")

	// Remove noscript tags and their content
	noscriptRe := regexp.MustCompile(`(?s)<noscript[^>]*>.*?</noscript>`)
	*html = noscriptRe.ReplaceAllString(*html, "")

	// Remove link tags (self-closing)
	linkRe := regexp.MustCompile(`<link[^>]*>`)
	*html = linkRe.ReplaceAllString(*html, "")

	// Remove style tags and their content
	styleRe := regexp.MustCompile(`(?s)<style[^>]*>.*?</style>`)
	*html = styleRe.ReplaceAllString(*html, "")

	// Format the cleaned HTML
	*html = gohtml.Format(*html)
}

// readLatestTimenetJSON finds and reads the most recent timenet JSON file
func readLatestTimenetJSON() (*TimenetData, error) {
	tempDir := os.TempDir()
	tempDir = "" // DEBUG - using current directory

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
	slog.Debug("Reading latest JSON file: " + latestFile)

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
