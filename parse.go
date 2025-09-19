package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// KIMAI DATA STRUCTURE
type KimaiData struct {
	PresentDate string           `json:"present_date"`
	PresentTime string           `json:"present_time"`
	Summary     KimaiSummary     `json:"summary"`
	DailyData   []KimaiDailyData `json:"daily_data"`
}

type KimaiSummary struct {
	ReportingDateFrom string `json:"reporting_date_from"`
	ReportingDateTo   string `json:"reporting_date_to"`
	WorkedHours       string `json:"worked_hours"`
}

type KimaiDailyData struct {
	Date        int    `json:"date"`
	In          string `json:"in"`
	Out         string `json:"out"`
	WorkedHours string `json:"worked_hours"`
	Customer    string `json:"customer"`
	Project     string `json:"project"`
	Activity    string `json:"activity"`
}

// TIMENET DATA STRUCTURE
type TimenetData struct {
	Date      string             `json:"current_date"`
	Time      string             `json:"current_time"`
	Summary   TimenetSummary     `json:"summary"`
	DailyData []TimenetDailyData `json:"daily_data"`
}

type TimenetSummary struct {
	ReportingDate           string `json:"reporting_date"`
	ExpectedHoursInMonth    string `json:"expected_hours_in_month"`
	ExpectedHoursInYear     string `json:"expected_hours_in_year"`
	WorkedHoursInMonth      string `json:"worked_hours_in_month"`
	WorkedHoursInYear       string `json:"worked_hours_in_year"`
	AccumuletedHoursInMonth string `json:"accumuleted_hours_in_month"`
	AccumuletedHoursInYear  string `json:"accumuleted_hours_in_year"`
}

type TimenetDailyData struct {
	Day           int    `json:"day"`
	ExpectedHours string `json:"expected_hours"`
	WorkedHours   string `json:"worked_hours"`
	Difference    string `json:"difference"`
	IsWorkable    bool   `json:"is_workable"`
}

// timenetParse extracts data from Timenet HTML and saves to JSON file
func timenetParse(htmlContent *string) error {
	if htmlContent == nil {
		return fmt.Errorf("HTML content is nil")
	}

	data := TimenetData{
		Date: time.Now().Format("2006/01/02"),
		Time: time.Now().Format("15:04"),
	}

	// NewDocumentFromReader takes a io.Reader not a string
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(*htmlContent))
	if err != nil {
		return err
	}

	// type TimenetDailyData struct {
	// 	Day           int    `json:"day"`
	// 	ExpectedHours string `json:"expected_hours"`
	// 	WorkedHours   string `json:"worked_hours"`
	// 	Difference    string `json:"difference"`
	// 	IsWorkable    bool   `json:"is_workable"`

	data.Summary.ReportingDate = doc.Find("div.container-mes-checks h2").First().Text()
	data.Summary.ExpectedHoursInMonth = doc.Find("table.table-resum-hores tbody tr").First().Find("td").Eq(1).Text()
	data.Summary.ExpectedHoursInYear = doc.Find("table.table-resum-hores tbody tr").First().Find("td").Eq(2).Text()
	data.Summary.WorkedHoursInMonth = doc.Find("table.table-resum-hores tbody tr").Eq(1).Find("td").Eq(1).Text()
	data.Summary.WorkedHoursInYear = doc.Find("table.table-resum-hores tbody tr").Eq(1).Find("td").Eq(2).Text()
	data.Summary.AccumuletedHoursInMonth = "nod defined"
	data.Summary.AccumuletedHoursInYear = doc.Find("table.table-resum-hores tbody tr").Eq(2).Find("td").Eq(2).Text()

	// Save to JSON file
	filename := fmt.Sprintf("timenet_data_%s.json", time.Now().Format("2006-01-02"))
	err = saveToJSON(data, filename)
	if err != nil {
		return fmt.Errorf("failed to save JSON: %v", err)
	}

	slog.Info("Timenet data saved to " + filename)
	return nil
}

// Save data to a JSON file in the OS temp folder
func saveToJSON(data any, filename string) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	tempDir := os.TempDir()
	fullPath := filepath.Join(tempDir, filename)

	err = os.WriteFile(fullPath, jsonData, 0644)
	if err != nil {
		return err
	}

	return nil
}

// removes unwanted elements and formats HTML in place
func cleanHTML(html *string) {
	if html == nil {
		return
	}

	// trim carriage returns and new lines FIRST
	*html = strings.ReplaceAll(*html, "\r", "")
	*html = strings.ReplaceAll(*html, "\n", "")

	// Remove empty HTML comments
	*html = strings.ReplaceAll(*html, "<!---->", "")

	// replace non-breaking spaces with regular spaces
	*html = strings.ReplaceAll(*html, "\u00A0", " ")

	// remove extra whitespace
	wsre := regexp.MustCompile(`[\s\p{Zs}]{2,}`)
	*html = wsre.ReplaceAllString(*html, " ")

	// remove white space after >
	wsare := regexp.MustCompile(`>\s+`)
	*html = wsare.ReplaceAllString(*html, ">")

	// remove white space before <
	wsbre := regexp.MustCompile(`\s+<`)
	*html = wsbre.ReplaceAllString(*html, "<")

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

	// Remove inline style attributes
	styleAttrRe := regexp.MustCompile(`\s+style="[^"]*"`)
	*html = styleAttrRe.ReplaceAllString(*html, "")

	// Format the cleaned HTML
	//*html = gohtml.Format(*html)
}

// extracts data from Kimai HTML and saves to JSON file
func kimaiParse(htmlContent *string) error {
	if htmlContent == nil {
		return fmt.Errorf("HTML content is nil")
	}

	data := KimaiData{
		PresentDate: time.Now().Format("2006/01/02"),
		PresentTime: time.Now().Format("15:04"),
	}

	// NewDocumentFromReader takes a io.Reader not a string
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(*htmlContent))
	if err != nil {
		return err
	}

	// Extract summary data
	data.Summary.ReportingDateFrom = doc.Find("#pick_in").AttrOr("value", "")
	data.Summary.ReportingDateTo = doc.Find("#pick_out").AttrOr("value", "")
	data.Summary.WorkedHours = doc.Find("#display_total").Text()

	// Save to JSON file
	filename := fmt.Sprintf("kimai_data_%s.json", time.Now().Format("2006-01-02"))
	err = saveToJSON(data, filename)
	if err != nil {
		return fmt.Errorf("failed to save JSON: %v", err)
	}

	slog.Info("Kimai data saved to " + filename)
	return nil
}
