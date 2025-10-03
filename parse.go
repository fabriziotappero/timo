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

// KIMAI JSON DATA STRUCTURE
type KimaiData struct {
	FetchDate   string             `json:"fetch_date"`
	FetchTime   string             `json:"fetch_time"`
	Summary     KimaiSummary       `json:"summary"`
	MonthlyData []KimaiMonthlyData `json:"monthly_data"`
}

type KimaiSummary struct {
	ReportingDateFrom string `json:"reporting_date_from"`
	ReportingDateTo   string `json:"reporting_date_to"`
	LoggedinUser      string `json:"loggedin_user"`
	WorkedTime        string `json:"worked_time"`
}

type KimaiMonthlyData struct {
	Date       string `json:"date"`
	In         string `json:"in"`
	Out        string `json:"out"`
	WorkedTime string `json:"worked_time"`
	Customer   string `json:"customer"`
	Project    string `json:"project"`
	Activity   string `json:"activity"`
	Username   string `json:"username"`
}

// TIMENET JSON DATA STRUCTURE
type TimenetData struct {
	FetchDate                string               `json:"fetch_date"`
	FetchTime                string               `json:"fetch_time"`
	Year                     string               `json:"year"`
	ExpectedWorkedTimeInYear string               `json:"expected_worked_time_in_year"`
	WorkedTimeInYear         string               `json:"worked_time_in_year"`
	OvertimeInYear           string               `json:"overtime_in_year"`
	MonthlyData              []TimenetMonthlyData `json:"monthly_data"`
}

type TimenetMonthlyData struct {
	Month                     string             `json:"month"`
	ExpectedWorkedTimeInMonth string             `json:"expected_worked_time_in_month"`
	WorkedTimeInMonth         string             `json:"worked_time_in_month"`
	OvertimeInMonth           string             `json:"overtime_in_month"`
	DailyData                 []TimenetDailyData `json:"daily_data"`
}

type TimenetDailyData struct {
	Date                    string `json:"date"`
	ExpectedWorkedTimeInDay string `json:"expected_worked_time_in_day"`
	WorkedTimeInDay         string `json:"worked_time_in_day"`
	OvertimeInDay           string `json:"overtime_in_day"`
	IsWorkDay               bool   `json:"is_work_day"`
	IsHoliday               bool   `json:"is_holiday"`
	IsVacation              bool   `json:"is_vacation"`
}

// timenetParse extracts data from Timenet HTML and saves to JSON file
func timenetParse(htmlContent *string) error {
	if htmlContent == nil {
		return fmt.Errorf("HTML content is nil")
	}

	data := TimenetData{
		FetchDate: time.Now().Format("2006/01/02"),
		FetchTime: time.Now().Format("15:04"),
	}

	// NewDocumentFromReader takes a io.Reader not a string
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(*htmlContent))
	if err != nil {
		return err
	}

	// REVIEW THESE 3 ITEMS
	data.ExpectedWorkedTimeInYear = strings.TrimSpace(doc.Find("table.table-resum-hores tbody tr").First().Find("td").Eq(2).Text())
	data.OvertimeInYear = strings.TrimSpace(doc.Find("table.table-resum-hores tbody tr").Eq(2).Find("td").Eq(2).Text())
	data.WorkedTimeInYear = strings.TrimSpace(doc.Find("table.table-resum-hores tbody tr").Eq(1).Find("td").Eq(2).Text())

	str := strings.TrimSpace(doc.Find(".container-mes-checks h2").First().Text()) // taken from current month
	data.Year = regexp.MustCompile(`[^0-9]`).ReplaceAllString(str, "")            // get only the year number

	monthlyEntries := doc.Find("div.card")
	slog.Info("Timenet. Number of months to parse", "count", monthlyEntries.Length())

	monthlyEntries.Each(func(i int, s *goquery.Selection) {

		// let's create one month of data
		monthlyData := TimenetMonthlyData{}

		str := strings.TrimSpace(s.Find(".container-mes-checks h2").First().Text())
		monthlyData.Month = GetMonth(str) // convert Spanish string to English month

		monthlyData.ExpectedWorkedTimeInMonth = strings.TrimSpace(s.Find("table.table-resum-hores tbody tr").First().Find("td").Eq(1).Text())
		monthlyData.WorkedTimeInMonth = strings.TrimSpace(s.Find("table.table-resum-hores tbody tr").Eq(1).Find("td").Eq(1).Text())
		monthlyData.OvertimeInMonth = strings.TrimSpace(s.Find("table.table-resum-hores tbody tr").Eq(2).Find("td").Eq(1).Text())

		// let's fill up each day of data in one month
		dailyEntries := s.Find("table.table-checks tbody tr")
		slog.Info("Timenet. Number of days to parse", "count", dailyEntries.Length())

		dailyEntries.Each(func(i int, content *goquery.Selection) {
			dailyData := TimenetDailyData{}

			// store data in format YYYY/MM/DD
			dailyData.Date = convertDateFormat(strings.TrimSpace(content.Find(".day-value").Text()))

			dailyData.ExpectedWorkedTimeInDay = strings.TrimSpace(content.Find(".prevision-day-check").Text())
			dailyData.WorkedTimeInDay = strings.TrimSpace(content.Find(".total-day-check span").Text())
			dailyData.OvertimeInDay = strings.TrimSpace(content.Find(".diff-day-check span").Text())

			dailyData.IsWorkDay = dailyData.ExpectedWorkedTimeInDay != ""

			dayTypeName := strings.TrimSpace(content.Find(".day-type-name").Text())
			dailyData.IsHoliday = strings.Contains(dayTypeName, "Festivo") || strings.Contains(dayTypeName, "Bank Holiday")

			dailyData.IsVacation = strings.Contains(dayTypeName, "Vacation") ||
				strings.Contains(dayTypeName, "Vacaciones") ||
				strings.Contains(dayTypeName, "Ausencia") ||
				(dayTypeName != "" && dayTypeName != "Laborable" && dayTypeName != "non working day" && !dailyData.IsHoliday)

			// Only add if we have a valid date
			if dailyData.Date != "" {
				monthlyData.DailyData = append(monthlyData.DailyData, dailyData)
				//slog.Info("Timenet. Parsed daily data for", "date", dailyData.Date)
			}

		})

		// let's fill up each day of data in one month
		monthlyRows := doc.Find("table.table-checks tbody tr")
		slog.Info("Timenet. Found and extracting daily rows", "count", monthlyRows.Length())

		// let's add the monthly data
		data.MonthlyData = append(data.MonthlyData, monthlyData)
		slog.Info("Timenet. Parsed monthly data for month", "month", monthlyData.Month)

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
		FetchDate: time.Now().Format("2006/01/02"),
		FetchTime: time.Now().Format("15:04"),
	}

	// NewDocumentFromReader takes a io.Reader not a string
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(*htmlContent))
	if err != nil {
		return err
	}

	data.Summary.LoggedinUser = strings.TrimSpace(doc.Find("#top #menu b").First().Text())

	// Extract summary data
	// TODO these dates are not the right format FIXIT
	data.Summary.ReportingDateFrom = doc.Find("#pick_in").AttrOr("value", "")
	data.Summary.ReportingDateTo = doc.Find("#pick_out").AttrOr("value", "")
	data.Summary.WorkedTime = formatTimeFromHMS(strings.TrimSpace(doc.Find("#display_total").Text()))

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

		// Extract worked time (format H:MM:SS) and convert to Xh Ym format
		workedTimeRaw := strings.TrimSpace(row.Find("td.time").Text())
		monthlyData.WorkedTime = formatTimeFromHMS(workedTimeRaw)

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

		// extras username if available
		monthlyData.Username = strings.TrimSpace(row.Find("td.username").Text())

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

// converts Spanish month names to English (case-insensitive)
func GetMonth(input string) string {
	monthMap := map[string]string{
		"enero": "January", "febrero": "February", "marzo": "March", "abril": "April",
		"mayo": "May", "junio": "June", "julio": "July", "agosto": "August",
		"septiembre": "September", "octubre": "October", "noviembre": "November", "diciembre": "December",
	}

	inputLower := strings.ToLower(input)
	for spanish, english := range monthMap {
		if strings.Contains(inputLower, spanish) {
			return english
		}
	}
	return input
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
