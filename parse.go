package main

import (
	"encoding/json"
	"fmt"
	"log"
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
	PresentDate string             `json:"present_date"`
	PresentTime string             `json:"present_time"`
	Summary     KimaiSummary       `json:"summary"`
	MonthlyData []KimaiMonthlyData `json:"monthly_data"`
}

type KimaiSummary struct {
	ReportingDateFrom string `json:"reporting_date_from"`
	ReportingDateTo   string `json:"reporting_date_to"`
	WorkedHours       string `json:"worked_hours"`
}

type KimaiMonthlyData struct {
	Date        string `json:"date"`
	In          string `json:"in"`
	Out         string `json:"out"`
	WorkedHours string `json:"worked_hours"`
	Customer    string `json:"customer"`
	Project     string `json:"project"`
	Activity    string `json:"activity"`
}

// TIMENET DATA STRUCTURE
type TimenetData struct {
	Date           string               `json:"current_date"`
	Time           string               `json:"current_time"`
	ReportingMonth string               `json:"reporting_month"`
	Summary        TimenetSummary       `json:"summary"`
	MonthlyData    []TimenetMonthlyData `json:"monthly_data"`
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

type TimenetMonthlyData struct {
	Date          string `json:"date"`
	ExpectedHours string `json:"expected_hours"`
	WorkedHours   string `json:"worked_hours"`
	Overtime      string `json:"overtime"`
	IsWorkingDay  bool   `json:"is_working_day"`
	IsHoliday     bool   `json:"is_holiday"`
	IsVacation    bool   `json:"is_vacation"`
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

	data.Summary.ReportingDate = strings.TrimSpace(doc.Find("div.container-mes-checks h2").First().Text())
	data.ReportingMonth = data.Summary.ReportingDate // Store the month at top level too
	data.Summary.ExpectedHoursInMonth = strings.TrimSpace(doc.Find("table.table-resum-hores tbody tr").First().Find("td").Eq(1).Text())
	data.Summary.ExpectedHoursInYear = strings.TrimSpace(doc.Find("table.table-resum-hores tbody tr").First().Find("td").Eq(2).Text())
	data.Summary.WorkedHoursInMonth = strings.TrimSpace(doc.Find("table.table-resum-hores tbody tr").Eq(1).Find("td").Eq(1).Text())
	data.Summary.WorkedHoursInYear = strings.TrimSpace(doc.Find("table.table-resum-hores tbody tr").Eq(1).Find("td").Eq(2).Text())
	data.Summary.AccumuletedHoursInMonth = strings.TrimSpace(doc.Find("table.table-resum-hores tbody tr").Eq(2).Find("td").Eq(1).Text())
	data.Summary.AccumuletedHoursInYear = strings.TrimSpace(doc.Find("table.table-resum-hores tbody tr").Eq(2).Find("td").Eq(2).Text())

	// extract daily data
	monthlyRows := doc.Find("table.table-checks tbody tr")
	slog.Info("Found and extracting daily rows: ", "count", monthlyRows.Length())

	monthlyRows.Each(func(i int, row *goquery.Selection) {
		monthlyData := TimenetMonthlyData{}

		// store data into format YYYY/MM/DD
		monthlyData.Date = convertDateFormat(strings.TrimSpace(row.Find(".day-value").Text()))

		dayTypeName := strings.TrimSpace(row.Find(".day-type-name").Text())
		monthlyData.IsHoliday = strings.Contains(dayTypeName, "Festivo") || strings.Contains(dayTypeName, "Bank Holiday")

		monthlyData.IsVacation = strings.Contains(dayTypeName, "Vacation") ||
			strings.Contains(dayTypeName, "Vacaciones") ||
			strings.Contains(dayTypeName, "Ausencia") ||
			(dayTypeName != "" && dayTypeName != "Laborable" && dayTypeName != "non working day" && !monthlyData.IsHoliday)

		monthlyData.ExpectedHours = strings.TrimSpace(row.Find(".prevision-day-check").Text())
		monthlyData.WorkedHours = strings.TrimSpace(row.Find(".total-day-check span").Text())
		monthlyData.Overtime = strings.TrimSpace(row.Find(".diff-day-check span").Text())

		// A working day should have expected hours set (data-driven approach)
		monthlyData.IsWorkingDay = monthlyData.ExpectedHours != ""

		// Only add if we have a valid date
		if monthlyData.Date != "" {
			data.MonthlyData = append(data.MonthlyData, monthlyData)
		}
	})

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
	// TODO these dates are not the right format FIXIT
	data.Summary.ReportingDateFrom = doc.Find("#pick_in").AttrOr("value", "")
	data.Summary.ReportingDateTo = doc.Find("#pick_out").AttrOr("value", "")
	data.Summary.WorkedHours = formatTimeFromHMS(strings.TrimSpace(doc.Find("#display_total").Text()))

	// Extract monthly data from timesheet entries
	monthlyRows := doc.Find("#timeSheetTable table tbody tr")
	slog.Info("Found and extracting timesheet rows: ", "count", monthlyRows.Length())

	monthlyRows.Each(func(i int, row *goquery.Selection) {
		monthlyData := KimaiMonthlyData{}

		// Extract date and convert it in format YYYY/MM/DD)
		dateText := strings.TrimSpace(row.Find("td.date").Text())
		monthlyData.Date = convertDateFormat(dateText)

		// Extract in/out times and convert to Xh Ym format
		monthlyData.In = formatTimeFromHMS(strings.TrimSpace(row.Find("td.from").Text()))
		monthlyData.Out = formatTimeFromHMS(strings.TrimSpace(row.Find("td.to").Text()))

		// Extract worked hours (format H:MM:SS) and convert to Xh Ym format
		workedHoursRaw := strings.TrimSpace(row.Find("td.time").Text())
		monthlyData.WorkedHours = formatTimeFromHMS(workedHoursRaw)

		// Extract customer name
		monthlyData.Customer = strings.TrimSpace(row.Find("td.customer").Text())

		// Extract project name (may be inside a link)
		projectCell := row.Find("td.project")
		projectLink := projectCell.Find("a")
		if projectLink.Length() > 0 {
			monthlyData.Project = strings.TrimSpace(projectLink.Text())
		} else {
			monthlyData.Project = strings.TrimSpace(projectCell.Text())
		}

		// Extract activity name (may be inside a link)
		activityCell := row.Find("td.activity")
		activityLink := activityCell.Find("a")
		if activityLink.Length() > 0 {
			monthlyData.Activity = strings.TrimSpace(activityLink.Text())
		} else {
			monthlyData.Activity = strings.TrimSpace(activityCell.Text())
		}

		// Only add if we have a valid date
		if monthlyData.Date != "" && dateText != "" {
			data.MonthlyData = append(data.MonthlyData, monthlyData)
		}
	})

	// Save to JSON file
	filename := fmt.Sprintf("kimai_data_%s.json", time.Now().Format("2006-01-02"))
	err = saveToJSON(data, filename)
	if err != nil {
		return fmt.Errorf("failed to save JSON: %v", err)
	}

	slog.Info("Kimai data saved to " + filename)
	return nil
}

func testKimaiParsing() {

	// Read the dump.html file
	content, err := os.ReadFile("dump.html")
	if err != nil {
		log.Fatal("Error reading dump.html:", err)
	}

	htmlString := string(content)

	// Test the kimai parsing
	fmt.Println("Testing Kimai HTML parsing...")
	err = kimaiParse(&htmlString)
	if err != nil {
		log.Fatal("Error parsing Kimai data:", err)
	}

	fmt.Println("Kimai parsing completed successfully! Check the JSON file in temp directory.")
}

func testTimenetParsing() {

	// Read the dump.html file
	content, err := os.ReadFile("dump.html")
	if err != nil {
		log.Fatal("Error reading dump.html:", err)
	}

	htmlString := string(content)

	// Test the timenet parsing
	fmt.Println("Testing Timenet HTML parsing...")
	err = timenetParse(&htmlString)
	if err != nil {
		log.Fatal("Error parsing Timenet data:", err)
	}

	fmt.Println("Parsing completed successfully! Check the JSON file in temp directory.")
}
