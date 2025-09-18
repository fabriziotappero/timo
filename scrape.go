// scrape.go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/chromedp/chromedp"
)

var chromiumPath string = ""

// find any Chromium/Chrome in the OS PATH, search as well in the user ~/.config dir
// if nothing is found, download and install a local copy of Chromium in
func setupScraper() {
	if !IsChromiumAvailable() {
		chromiumPath, _ = GetCustomChromiumToPath()
		if chromiumPath == "" {
			DownloadChromium()
			InstallCustomChromium()
		}
	}
}

func scrapeTimenet(password string) (string, error) {

	// Create context with headless browser
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(chromiumPath), // if chromiumPath is empty, it uses any Chromium found in PATH
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
	)

	slog.Info("Using Chrome/Chromium executable:", "path", chromiumPath)

	ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	// Set longer timeout
	ctx, cancel = context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	var responseHTML string

	err := chromedp.Run(ctx,
		// Navigate to Timenet login page
		chromedp.Navigate("https://timenet-cp.gpisoftware.com/check/17f24d33-13d0-4edc-b2e8-fdec9834d639"),

		// Wait for page to load completely
		chromedp.Sleep(3*time.Second),

		// Try to find the password input field with multiple selectors
		chromedp.WaitVisible(`input[name="pin"], input[id="gpi-input-0"], input[type="password"]`, chromedp.ByQuery),

		// Remove readonly attribute from password field (try multiple selectors)
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Try different selectors for the password field
			selectors := []string{`input[type="password"]`}
			for _, selector := range selectors {
				err := chromedp.RemoveAttribute(selector, "readonly", chromedp.ByQuery).Do(ctx)
				if err == nil {
					// Clear and input password
					chromedp.Clear(selector, chromedp.ByQuery).Do(ctx)
					return chromedp.SendKeys(selector, password, chromedp.ByQuery).Do(ctx)
				}
			}
			return fmt.Errorf("could not find password input field")
		}),

		// Wait a moment before submitting
		chromedp.Sleep(2*time.Second),

		// Submit the form (try multiple submit selectors)
		chromedp.ActionFunc(func(ctx context.Context) error {
			submitSelectors := []string{`.enter-button`}
			for _, selector := range submitSelectors {
				err := chromedp.Click(selector, chromedp.ByQuery).Do(ctx)
				if err == nil {
					slog.Info("Successfully clicked submit button with selector: " + selector)
					return nil
				}
			}
			return fmt.Errorf("could not find submit button")
		}),

		chromedp.Sleep(3*time.Second),

		// Click on "Marcajes" button
		chromedp.ActionFunc(func(ctx context.Context) error {
			marcajesSelectors := []string{`button.btn-secondary[aria-label="Marcajes"]`}
			for _, selector := range marcajesSelectors {
				err := chromedp.Click(selector, chromedp.ByQuery).Do(ctx)
				if err == nil {
					slog.Info("Successfully clicked button with selector: " + selector)
					return nil
				}
			}
			return fmt.Errorf("could not find Marcajes button")
		}),

		chromedp.Sleep(3*time.Second),

		// Click on "Ver todos mis marcajes" button
		chromedp.ActionFunc(func(ctx context.Context) error {
			verTodosSelectors := []string{`button.btn.btn-secondary[aria-label="Ver todos mis marcajes"]`}
			for _, selector := range verTodosSelectors {
				err := chromedp.Click(selector, chromedp.ByQuery).Do(ctx)
				if err == nil {
					slog.Info("Successfully clicked button with selector: " + selector)
					return nil
				}
			}
			return fmt.Errorf("could not find Ver todos mis marcajes button")
		}),

		chromedp.Sleep(3*time.Second),

		// Capture the full HTML
		chromedp.OuterHTML(`html`, &responseHTML, chromedp.ByQuery),
	)

	if err != nil {
		return "", fmt.Errorf("failed to scrape Timenet: %v", err)
	}

	// Return the response HTML
	slog.Info("Timenet scrape successful")
	return responseHTML, nil
}

func scrapeKimai(id string, password string) (string, error) {

	// Create context with headless browser
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(chromiumPath), // if chromiumPath is empty, it uses any Chromium found in PATH
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("ignore-certificate-errors", true),
	)

	slog.Info("Using Chrome/Chromium executable:", "path", chromiumPath)

	ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	// Set timeout
	ctx, cancel = context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	var responseHTML string

	err := chromedp.Run(ctx,
		chromedp.Navigate("https://kimai.itk-spain.com/index.php"),
		chromedp.Sleep(3*time.Second),
		chromedp.WaitVisible(`#kimaiusername`, chromedp.ByQuery),
		chromedp.Clear(`#kimaiusername`, chromedp.ByQuery),
		chromedp.SendKeys(`#kimaiusername`, id, chromedp.ByQuery),
		chromedp.Clear(`#kimaipassword`, chromedp.ByQuery),
		chromedp.SendKeys(`#kimaipassword`, password, chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
		chromedp.Click(`#loginButton`, chromedp.ByQuery),
		chromedp.Sleep(3*time.Second),
		chromedp.OuterHTML(`html`, &responseHTML, chromedp.ByQuery),
	)

	if err != nil {
		return "", fmt.Errorf("failed to scrape Kimai: %v", err)
	}

	// Return the response HTML
	slog.Info("Kimai scrape successful")
	return responseHTML, nil
}
