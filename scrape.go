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
	ctx, timeoutCancel := context.WithTimeout(ctx, 25*time.Second)
	// Compose all cancels into one
	cancel := func() {
		timeoutCancel()
		ctxCancel()
		allocCancel()
	}
	return ctx, cancel
}

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

	// dump HTML to file for debugging
	//os.WriteFile("dump.html", []byte(responseHTML), 0644)

	// Return the response HTML
	slog.Info("Timenet Web scrape successful")
	return responseHTML, nil

}

func scrapeKimai(id string, password string) (string, error) {

	ctx, cancel := newChromeContext(
		chromedp.Flag("ignore-certificate-errors", true),
	)
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
