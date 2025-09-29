// scrape.go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/yosssi/gohtml"
)

var chromiumPath string = ""

// calculateMonthsDifference calculates how many months to navigate from current to target date
// currentDate and targetDate should be in format "dd/mm/yyyy"
// Returns positive number for going forwards, negative for going backwards
func calculateMonthsDifference(currentDate, targetDate string) int {
	// Check for empty dates
	if currentDate == "" || targetDate == "" {
		slog.Warn("Empty date provided", "current", currentDate, "target", targetDate)
		return 0
	}

	// Parse current date (e.g., "01/09/2025")
	var currentDay, currentMonth, currentYear int
	fmt.Sscanf(currentDate, "%d/%d/%d", &currentDay, &currentMonth, &currentYear)

	// Parse target date (e.g., "01/01/2025")
	var targetDay, targetMonth, targetYear int
	fmt.Sscanf(targetDate, "%d/%d/%d", &targetDay, &targetMonth, &targetYear)

	// Calculate months difference (target - current)
	// Positive = go forward, Negative = go backward
	monthsDiff := (targetYear-currentYear)*12 + (targetMonth - currentMonth)

	slog.Info("Date calculation",
		"current", currentDate, "target", targetDate,
		"currentMonth", currentMonth, "targetMonth", targetMonth,
		"monthsDifference", monthsDiff)

	return monthsDiff
}

// set the date in the Kimai date picker for both from and to date
func setDatePickerFilter(dateStr string, fieldSelector string) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		// Validate fieldSelector
		if fieldSelector != "#ts_in" && fieldSelector != "#ts_out" {
			return fmt.Errorf("invalid fieldSelector '%s': must be '#ts_in' or '#ts_out'", fieldSelector)
		}

		// First, read the current date from the specified field
		var currentDateText string
		chromedp.Text(fieldSelector, &currentDateText, chromedp.ByQuery).Do(ctx)

		slog.Info("Current date picker shows", "date", currentDateText, "field", fieldSelector)
		slog.Info("We want to set", "date", dateStr, "field", fieldSelector)

		// Check if we're already at the target date
		if currentDateText == dateStr {
			slog.Info("Already at target date, no navigation needed")
			return nil
		}

		// Calculate how many months to navigate
		monthsDiff := calculateMonthsDifference(currentDateText, dateStr)

		// Open the date picker
		chromedp.WaitVisible(fieldSelector, chromedp.ByQuery).Do(ctx)
		chromedp.Click(fieldSelector, chromedp.ByQuery).Do(ctx)
		chromedp.Sleep(2 * time.Second).Do(ctx)
		chromedp.WaitVisible(`#ui-datepicker-div`, chromedp.ByQuery).Do(ctx)

		// Click prev/next based on calculated difference
		if monthsDiff > 0 {
			// Go forwards (next)
			for i := 0; i < monthsDiff; i++ {
				chromedp.Click(`.ui-datepicker-next`, chromedp.ByQuery).Do(ctx)
				chromedp.Sleep(500 * time.Millisecond).Do(ctx)
			}
		} else if monthsDiff < 0 {
			// Go backwards (prev)
			for i := 0; i < -monthsDiff; i++ {
				chromedp.Click(`.ui-datepicker-prev`, chromedp.ByQuery).Do(ctx)
				chromedp.Sleep(500 * time.Millisecond).Do(ctx)
			}
		}

		// Extract day from dateStr and click on it using the HTML select
		var targetDay, targetMonth, targetYear int
		fmt.Sscanf(dateStr, "%d/%d/%d", &targetDay, &targetMonth, &targetYear)
		daySelector := fmt.Sprintf(`//a[text()="%d"]`, targetDay)
		chromedp.Click(daySelector, chromedp.BySearch).Do(ctx)
		chromedp.Sleep(2 * time.Second).Do(ctx)

		return nil
	})
}

// find any Chromium/Chrome in the OS PATH, common locations, or user config dir
// if nothing is found, download and install a local copy of Chromium
func setupScraper() {
	// First try to find Chrome/Chromium in PATH or oher common locations
	chromiumPath = FindChromiumExecutable()

	if chromiumPath == "" {
		// Try to get custom chromium from user config directory
		customPath, err := GetCustomChromiumToPath()
		if err == nil {
			chromiumPath = customPath
		} else {
			// Last resort: download and install chromium
			fmt.Println("Setting up external tools...")
			DownloadChromium()
			InstallCustomChromium()
			// Try to get the path again after installation
			chromiumPath, _ = GetCustomChromiumToPath()
		}
	}
}

// creates a chromedp context with common options and timeout
func newChromeContext(extraOpts ...chromedp.ExecAllocatorOption) (context.Context, context.CancelFunc) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(chromiumPath),
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
	)
	opts = append(opts, extraOpts...)
	slog.Info("Using Chrome/Chromium executable:", "path", chromiumPath)
	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, ctxCancel := chromedp.NewContext(allocCtx)
	// Set timeout
	ctx, timeoutCancel := context.WithTimeout(ctx, 35*time.Second)
	// Compose all cancels into one
	cancel := func() {
		timeoutCancel()
		ctxCancel()
		allocCancel()
	}
	return ctx, cancel
}

// scrape timenet website content for the current month
func scrapeTimenet(password string) (string, error) {

	ctx, cancel := newChromeContext(
	//chromedp.Flag("headless", false),
	)
	defer cancel()

	var responseHTML string

	err := chromedp.Run(ctx,
		chromedp.Navigate("https://timenet-wcp.gpisoftware.com/login/28b27216-c0c8-469c-816b-c65d0a11c7dd"),
		chromedp.Sleep(1*time.Second),

		chromedp.WaitVisible(`#gpi-input-0`, chromedp.ByQuery),
		chromedp.Clear(`#gpi-input-0`, chromedp.ByQuery),
		chromedp.SendKeys(`#gpi-input-0`, password+"\n", chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),

		chromedp.WaitVisible(`footer`, chromedp.ByQuery),
		chromedp.Click(`a.nav-link[href="/checks"]`, chromedp.ByQuery),

		chromedp.Sleep(2*time.Second),
		chromedp.WaitVisible(`footer`, chromedp.ByQuery),

		chromedp.OuterHTML(`html`, &responseHTML, chromedp.ByQuery),
	)

	if err != nil {
		return "", fmt.Errorf("failed to scrape Timenet Web: %v", err)
	}

	// DEBUG dump HTML to file
	if false {
		cleanHTML(&responseHTML)
		responseHTML = gohtml.Format(responseHTML)
		os.WriteFile("dump.html", []byte(responseHTML), 0644)
		//os.Exit(0)
	}

	// Return the response HTML
	slog.Info("Timenet Web scrape successful")
	return responseHTML, nil

}

// scrape kimai website content for the whole current year
// once logged into the kimai site store current view filter,
// reset the filter and once finished restore the filter.
func scrapeKimai(id string, password string) (string, error) {

	ctx, cancel := newChromeContext(
		chromedp.Flag("ignore-certificate-errors", true),
		//chromedp.Flag("headless", false),
	)
	defer cancel()

	var responseHTML string
	var viewFilterOriginalStartDate string
	var viewFilterOriginalEndDate string
	var viewFilterStartDate string
	var viewFilterEndDate string

	var currentDate string = time.Now().Format("02/01/2006")

	// Calculate January 1st of the year in currentDate
	var day, month, year int
	fmt.Sscanf(currentDate, "%d/%d/%d", &day, &month, &year)
	januaryFirst := fmt.Sprintf("01/01/%d", year)

	// Calculate last day of the month and year in currentDate
	firstOfNextMonth := time.Date(year, time.Month(month+1), 1, 0, 0, 0, 0, time.UTC)
	lastDayOfMonth := firstOfNextMonth.AddDate(0, 0, -1)
	lastDayOfMonthStr := lastDayOfMonth.Format("02/01/2006")

	slog.Info("Kimai URL is going to be scraped", "fromDate", januaryFirst, "toDate", lastDayOfMonthStr)

	err := chromedp.Run(ctx,
		// login
		chromedp.Navigate("https://kimai.itk-spain.com/index.php"),
		chromedp.Sleep(2*time.Second),
		chromedp.WaitVisible(`#kimaiusername`, chromedp.ByQuery),
		chromedp.Clear(`#kimaiusername`, chromedp.ByQuery),
		chromedp.SendKeys(`#kimaiusername`, id, chromedp.ByQuery),
		chromedp.Sleep(1*time.Second),
		chromedp.Clear(`#kimaipassword`, chromedp.ByQuery),
		chromedp.SendKeys(`#kimaipassword`, password, chromedp.ByQuery),
		chromedp.Sleep(1*time.Second),
		chromedp.Click(`#loginButton`, chromedp.ByQuery),
		chromedp.Sleep(1*time.Second),

		// wait for date picker elements to be visible/loaded
		chromedp.WaitVisible(`#dates`, chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
		chromedp.WaitVisible(`#ts_in`, chromedp.ByQuery),
		chromedp.WaitVisible(`#ts_out`, chromedp.ByQuery),

		// store locally the current view filter
		chromedp.Text(`#ts_in`, &viewFilterOriginalStartDate, chromedp.ByQuery),
		chromedp.Text(`#ts_out`, &viewFilterOriginalEndDate, chromedp.ByQuery),

		// set the from view filter to January 1st of current year
		setDatePickerFilter(januaryFirst, "#ts_in"),

		// set the to view filter to last day of current month
		setDatePickerFilter(lastDayOfMonthStr, "#ts_out"),

		// wait for date picker elements to be visible/loaded
		chromedp.WaitVisible(`#dates`, chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),

		// store current view filter
		chromedp.Text(`#ts_in`, &viewFilterStartDate, chromedp.ByQuery),
		chromedp.Text(`#ts_out`, &viewFilterEndDate, chromedp.ByQuery),

		// scrape current year data content
		chromedp.OuterHTML(`html`, &responseHTML, chromedp.ByQuery),
	)
	slog.Info("Just scraped Kimai content with View filter", "start", viewFilterStartDate, "end", viewFilterEndDate)

	if err != nil {
		return "", fmt.Errorf("failed to scrape Kimai: %v", err)
	}

	// restore original date picker view filter
	err1 := chromedp.Run(ctx,
		chromedp.Sleep(1*time.Second),
		setDatePickerFilter(viewFilterOriginalStartDate, "#ts_in"),
		chromedp.Sleep(1*time.Second),
		setDatePickerFilter(viewFilterOriginalEndDate, "#ts_out"),
	)
	slog.Info("Restored original Kimai view filter", "start", viewFilterOriginalStartDate, "end", viewFilterOriginalEndDate)

	if err1 != nil {
		return "", fmt.Errorf("failed to reset Kimai date picker date: %v", err1)
	}

	// Return the response HTML
	slog.Info("Kimai scraping was successful")
	return responseHTML, nil
}
