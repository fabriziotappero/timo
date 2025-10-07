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
func setDatePickerFilter(dateTarget string, fieldSelector string) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		// Validate fieldSelector
		if fieldSelector != "#ts_in" && fieldSelector != "#ts_out" {
			return fmt.Errorf("invalid fieldSelector '%s': must be '#ts_in' or '#ts_out'", fieldSelector)
		}

		// First, read the current date from the specified field
		var currentDateText string
		chromedp.Text(fieldSelector, &currentDateText, chromedp.ByQuery).Do(ctx)

		slog.Info("Kimai: Current date picker shows", "date", currentDateText, "field", fieldSelector)
		slog.Info("Kimai: We want to set", "date", dateTarget, "field", fieldSelector)

		// Check if we're already at the target date
		if currentDateText == dateTarget {
			slog.Info("Kimai: Already at target date, no navigation needed")
			return nil
		}

		// Calculate how many months to navigate
		monthsDiff := calculateMonthsDifference(currentDateText, dateTarget)

		// Open the date picker by evaluating JS on the hidden input
		chromedp.Sleep(2 * time.Second).Do(ctx)
		chromedp.WaitVisible(fieldSelector, chromedp.ByQuery).Do(ctx)
		slog.Info("Kimai: Triggering datepicker via JS on hidden input")
		var inputSelector string
		if fieldSelector == "#ts_in" {
			inputSelector = "#pick_in"
		} else if fieldSelector == "#ts_out" {
			inputSelector = "#pick_out"
		} else {
			return fmt.Errorf("invalid fieldSelector '%s': must be '#ts_in' or '#ts_out'", fieldSelector)
		}
		errEval := chromedp.EvaluateAsDevTools(fmt.Sprintf("$('#%s').datepicker('show')", inputSelector[1:]), nil).Do(ctx)
		if errEval != nil {
			slog.Error("failed to trigger datepicker via JS", "error", errEval)
		}

		// let's wait a bit for the date picker to be visible
		chromedp.Sleep(2 * time.Second).Do(ctx)
		chromedp.WaitVisible(`.ui-datepicker-calendar`, chromedp.ByQuery).Do(ctx)

		// Click prev/next based on calculated difference
		slog.Info("Kimai: Trying to set MONTH in date picker moving of", "monthsDiff", monthsDiff)
		if monthsDiff > 0 {
			// Go forwards (next)
			for i := 0; i < monthsDiff; i++ {
				chromedp.WaitVisible(`.ui-datepicker-next`, chromedp.ByQuery).Do(ctx)
				chromedp.Click(`.ui-datepicker-next`, chromedp.ByQuery).Do(ctx)
				chromedp.Sleep(1 * time.Millisecond).Do(ctx)
				//chromedp.WaitVisible(`.ui-datepicker-title`, chromedp.ByQuery).Do(ctx)
			}
		} else if monthsDiff < 0 {
			// Go backwards (prev)
			for i := 0; i < -monthsDiff; i++ {
				chromedp.WaitVisible(`.ui-datepicker-prev`, chromedp.ByQuery).Do(ctx)
				chromedp.Click(`.ui-datepicker-prev`, chromedp.ByQuery).Do(ctx)
				chromedp.Sleep(1 * time.Millisecond).Do(ctx)
				//chromedp.WaitVisible(`.ui-datepicker-title`, chromedp.ByQuery).Do(ctx)
			}
		} else {
			slog.Info("Kimai: No month navigation is needed")
		}
		chromedp.Sleep(1 * time.Second).Do(ctx)

		// Extract day from dateTarget and click on it using the HTML select
		slog.Info("Kimai: Trying to set DAY in date picker", "date", dateTarget)

		chromedp.WaitVisible(`.ui-datepicker-calendar`, chromedp.ByQuery).Do(ctx)
		var targetDay, targetMonth, targetYear int
		fmt.Sscanf(dateTarget, "%d/%d/%d", &targetDay, &targetMonth, &targetYear)
		daySelector := fmt.Sprintf(`//a[text()="%d"]`, targetDay)
		slog.Info("Kimai: Trying to click on day in date picker", "day", targetDay)
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
		// Force desktop viewport size to avoid mobile layout
		chromedp.Flag("window-size", "1920,1080"),

		// Disable password save prompts and notifications
		chromedp.Flag("disable-password-generation", true),
		chromedp.Flag("disable-save-password-bubble", true),
		chromedp.Flag("disable-password-manager-reauthentication", true),
		chromedp.Flag("disable-notifications", true),
		chromedp.Flag("disable-desktop-notifications", true),
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("disable-infobars", true),
		chromedp.Flag("disable-translate", true),
		chromedp.Flag("disable-popup-blocking", true),
		// Additional Windows headless stability flags
		//chromedp.Flag("disable-features", "VizDisplayCompositor"),
		//chromedp.Flag("disable-background-timer-throttling", true),
		chromedp.Flag("disable-backgrounding-occluded-windows", true),
		//chromedp.Flag("disable-renderer-backgrounding", true),
		//chromedp.Flag("disable-field-trial-config", true),
		//chromedp.Flag("disable-ipc-flooding-protection", true),
		//chromedp.Flag("single-process", true), // This can help with Windows headless issues
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

// append the HTML content of the specified selector to the target string
func appendHTML(selector string, target *string) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {

		err := chromedp.WaitVisible(`div.card`, chromedp.ByQuery).Do(ctx)
		if err != nil {
			return err
		}
		chromedp.Sleep(1 * time.Second).Do(ctx)

		var htmlContent string
		err = chromedp.OuterHTML(selector, &htmlContent, chromedp.ByQuery).Do(ctx)
		if err != nil {
			return err
		}
		*target += htmlContent
		return nil
	})
}

// scrape timenet website content from january first of curent year
func scrapeTimenet(password string) (string, error) {

	ctx, cancel := newChromeContext(
	//chromedp.Flag("headless", false),
	)
	defer cancel()

	monthsToGoBack := int(time.Now().Month() - time.January)
	slog.Info("Timenet: Scraping months from January to current month",
		"monthsToScrape", monthsToGoBack+1, "currentMonth", time.Now().Month().String())

	// scrape last 12 months
	//monthsToGoBack := 11 // Always go back 11 months to get 12 months total (current + 11 previous)
	//slog.Info("Timenet. Scraping 12 months of data")

	var responseHTML string

	err := chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			slog.Info("Timenet: Navigating to Timenet login page")
			return chromedp.Navigate("https://timenet-wcp.gpisoftware.com/login/28b27216-c0c8-469c-816b-c65d0a11c7dd").Do(ctx)
		}),
		chromedp.Sleep(1*time.Second),

		// login
		chromedp.ActionFunc(func(ctx context.Context) error {
			slog.Info("Timenet: Waiting for login input to be visible")
			return chromedp.WaitVisible(`#gpi-input-0`, chromedp.ByQuery).Do(ctx)
		}),
		chromedp.Clear(`#gpi-input-0`, chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			slog.Info("Timenet: Entering password and submitting")
			return chromedp.SendKeys(`#gpi-input-0`, password+"\n", chromedp.ByQuery).Do(ctx)
		}),

		chromedp.Sleep(2*time.Second),

		// go to checks page
		chromedp.ActionFunc(func(ctx context.Context) error {
			slog.Info("Timenet: Waiting for checks navigation link to be clickable")
			// First wait for the link to be visible
			err := chromedp.WaitVisible(`a.nav-link[href="/checks"]`, chromedp.ByQuery).Do(ctx)
			if err != nil {
				slog.Error("Timenet: CHECKS LINK not visible", "error", err)
				return err
			}
			// Add extra wait for Windows headless mode
			chromedp.Sleep(2 * time.Second).Do(ctx)
			slog.Info("Timenet: Clicking checks navigation link")
			return chromedp.Click(`a.nav-link[href="/checks"]`, chromedp.ByQuery).Do(ctx)
		}),
		chromedp.Sleep(2*time.Second), // Give more time for navigation

		// Verify we're on the checks page by waiting for a checks-specific element
		chromedp.ActionFunc(func(ctx context.Context) error {
			slog.Info("Timenet: Waiting for checks page to load")
			err := chromedp.WaitVisible(`div.container-mes-checks`, chromedp.ByQuery).Do(ctx)
			if err != nil {
				slog.Error("Timenet: Checks page container not found", "error", err)
				return err
			}
			slog.Info("Timenet: Successfully navigated to checks page")
			return nil
		}),

		// loop monthsToGoBack times to go back to January of the same year
		chromedp.ActionFunc(func(ctx context.Context) error {
			slog.Info("Timenet: Starting month iteration loop", "totalMonths", monthsToGoBack+1)
			for i := 0; i < monthsToGoBack+1; i++ {
				slog.Info("Timenet: Processing month iteration", "iteration", i+1, "of", monthsToGoBack+1)

				var err error

				// append current month HTML to responseHTML
				err = appendHTML("div.card", &responseHTML).Do(ctx)
				if err != nil {
					slog.Error("Timenet: Failed to append HTML", "iteration", i+1, "error", err)
					return err
				}

				chromedp.Sleep(1 * time.Second).Do(ctx)

				// click back button to go to previous month
				err = chromedp.Click(`div.container-mes-checks button:first-child`, chromedp.ByQuery).Do(ctx)
				if err != nil {
					return err
				}
			}
			slog.Info("Timenet: Completed all month iterations successfully")
			return nil
		}),
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

// scrape kimai website content from january first of curent year.
// Once logged into the kimai site store current view filter then
// sets it to January 1st of current year and once finished scraping
// re-sets to its original state.
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
		chromedp.Sleep(1*time.Second),
		chromedp.WaitVisible(`#kimaiusername`, chromedp.ByQuery),
		chromedp.Clear(`#kimaiusername`, chromedp.ByQuery),
		chromedp.SendKeys(`#kimaiusername`, id, chromedp.ByQuery),
		chromedp.Sleep(300*time.Millisecond),
		chromedp.Clear(`#kimaipassword`, chromedp.ByQuery),
		chromedp.SendKeys(`#kimaipassword`, password, chromedp.ByQuery),
		chromedp.Sleep(300*time.Millisecond),
		chromedp.Click(`#loginButton`, chromedp.ByQuery),
		chromedp.Sleep(1*time.Second),

		// in Kimai preference set 920 entries per page
		chromedp.Evaluate(`floaterShow("floaters.php","prefs",0,0,450);`, nil), // open preferences floating panel
		chromedp.Sleep(1*time.Second),
		chromedp.WaitVisible(`#floater`, chromedp.ByQuery),
		chromedp.Sleep(1*time.Second),
		chromedp.Click(`#floater .menu.tabSelection li:nth-child(3)`, chromedp.ByQuery),
		chromedp.Sleep(300*time.Millisecond),
		chromedp.WaitVisible(`#rowlimit`, chromedp.ByQuery),
		chromedp.Sleep(300*time.Millisecond),
		chromedp.Clear(`#rowlimit`, chromedp.ByQuery),
		chromedp.SendKeys(`#rowlimit`, "920\n", chromedp.ByQuery),
		chromedp.Sleep(300*time.Millisecond),

		// wait for date picker elements to be visible/loaded
		chromedp.WaitVisible(`#dates`, chromedp.ByQuery),
		chromedp.Sleep(300*time.Millisecond),
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

		// store locally current view filter
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
	slog.Info("Kimai: Restored original view filter", "start", viewFilterOriginalStartDate, "end", viewFilterOriginalEndDate)

	if err1 != nil {
		return "", fmt.Errorf("failed to reset Kimai date picker date: %v", err1)
	}

	// Return the response HTML
	slog.Info("Kimai scraping was successful")
	return responseHTML, nil
}
