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

// set the from date in the Kimai date picker
func setDatePickerFromDate(fromDateStr string) chromedp.Action {

	// for the moment we just go back to previous month and select day 1

	return chromedp.Tasks{
		chromedp.Sleep(3 * time.Second),
		chromedp.WaitVisible(`#ts_in`, chromedp.ByQuery),
		chromedp.Click(`#display #dates #ts_in`, chromedp.ByQuery),
		chromedp.Sleep(3 * time.Second),
		chromedp.WaitVisible(`#ui-datepicker-div`, chromedp.ByQuery),
		chromedp.Sleep(3 * time.Second),
		chromedp.Click(`.ui-datepicker-prev`, chromedp.ByQuery),
		chromedp.Sleep(2 * time.Second),
		chromedp.WaitVisible(`td.ui-datepicker-current-day`, chromedp.ByQuery),
		chromedp.Click(`//a[text()="1"]`, chromedp.BySearch),
		chromedp.Sleep(2 * time.Second),
	}
}

// find any Chromium/Chrome in the OS PATH, search as well in the user ~/.config dir
// if nothing is found, download and install a local copy of Chromium in
func setupScraper() {
	if !IsChromiumAvailable() {
		chromiumPath, _ = GetCustomChromiumToPath()
		if chromiumPath == "" {
			fmt.Println("Setting up some external tools...")
			DownloadChromium()
			InstallCustomChromium()
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

	ctx, cancel := newChromeContext()
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
		chromedp.Flag("headless", false),
	)
	defer cancel()

	var responseHTML string
	var viewFilterOriginalStartDate string
	var viewFilterOriginalEndDate string
	var viewFilterStartDate string
	var viewFilterEndDate string

	err := chromedp.Run(ctx,
		// login
		chromedp.Navigate("https://kimai.itk-spain.com/index.php"),
		chromedp.Sleep(2*time.Second),
		chromedp.WaitVisible(`#kimaiusername`, chromedp.ByQuery),
		chromedp.Clear(`#kimaiusername`, chromedp.ByQuery),
		chromedp.SendKeys(`#kimaiusername`, id, chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
		chromedp.Clear(`#kimaipassword`, chromedp.ByQuery),
		chromedp.SendKeys(`#kimaipassword`, password, chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
		chromedp.Click(`#loginButton`, chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),

		// wait for date picker elements to be visible/loaded
		chromedp.WaitVisible(`#dates`, chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),

		// store locally the current view filter
		chromedp.Text(`#ts_in`, &viewFilterOriginalStartDate, chromedp.ByQuery),
		chromedp.Text(`#ts_out`, &viewFilterOriginalEndDate, chromedp.ByQuery),

		// set the from view filter to January 1st of current year
		setDatePickerFromDate("01/01/2025"),

		// wait for date picker elements to be visible/loaded
		chromedp.WaitVisible(`#dates`, chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),

		// store current view filter
		chromedp.Text(`#ts_in`, &viewFilterStartDate, chromedp.ByQuery),
		chromedp.Text(`#ts_out`, &viewFilterEndDate, chromedp.ByQuery),

		// scrape current year data content
		chromedp.OuterHTML(`html`, &responseHTML, chromedp.ByQuery),
		//chromedp.Sleep(5*time.Second),
	)
	slog.Info("Just scraped Kimai content with View filter", "start", viewFilterStartDate, "end", viewFilterEndDate)
	slog.Info("Restored Kimai URL with original View filter", "start", viewFilterOriginalStartDate, "end", viewFilterOriginalEndDate)

	if err != nil {
		return "", fmt.Errorf("failed to scrape Kimai: %v", err)
	}

	// Return the response HTML
	slog.Info("Kimai scrape successful")
	return responseHTML, nil
}
