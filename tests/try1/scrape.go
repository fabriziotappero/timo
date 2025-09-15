// scrape.go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/chromedp/chromedp"
)

// scrapeTimenet automates login to Timenet
func scrapeTimenet(password string) (string, error) {
	// Create context with headless browser
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true), // Set to true for headless mode
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
	)
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
	slog.Info("Timenet login successful, response received")
	return responseHTML, nil
}
