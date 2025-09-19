
Practical Golang utility to manage you logged hours using Timenet and Kimai.

<div align="center"><img src="img/main.jpeg" alt="" width="400"></div>

<p>&nbsp;</p>
<p align="center">
  <a href="https://github.com/fabriziotappero/timo/releases/"><b>DOWNLOAD LASTEST VERSION</b></a>
</p>

## Basic Principles
- Terminal-style application usable as double-click Windows app too.
- Handles login and dynamically scraping.
- Run Kimai and Timenet scraping in parallel in background.
- All scraped data is stored locally in JSON files.
- Simple and intuitive report generation from JSON files.
- Automatically check for new version comparing version from github main branch.

## How Timo Works

Timo is a Terminal program written in Go and compiled for both Linux and Windows. It uses The Charm TUI library for visualization and chromedp 
for dynamic scraping the Kimai and the Timenet URLs.

Once remote HTML information scraped, DOM parsing is done using the `github.com/PuerkitoBio/goquery` library.

local JSON data is processed on request (by the user) and presented to the TUI in a concise manner.

Logged data and all scraped information is stored in JSON files located in the OS 
temporary folder ``~/tmp/`` in Linux or `~\AppData\Local\Temp` in Windows.

## Some Help With Golang

https://go.dev/doc/effective_go

Run the app while developing: 

```
go mod init timo
go mod tidy
go run . --debug
```
Build for Windows and Linux:

```
# 64-bit Windows
go env -w GOOS=windows GOARCH=amd64; go build -o build/timo.exe .

# Linux
go env -w GOOS=linux GOARCH=amd64; go build -o build/timo .
```
