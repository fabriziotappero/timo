// scrape.go
package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/chromedp/chromedp"
)

func main() {
	// Create a context
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Set a timeout for the operation
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	url := "https://www.hubspot.com/"

	// Run the ChromeDP tasks
	var htmlContent string
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.OuterHTML("html", &htmlContent),
	)

	if err != nil {
		log.Fatal("Error during scraping:", err)
	}

	// Store the HTML content in a file
	err = os.WriteFile("test.html", []byte(htmlContent), 0644)
	if err != nil {
		log.Fatal("Error writing to file:", err)
	}
}
