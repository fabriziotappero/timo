
Practical Golang utility to manage you logged hours using Timenet and Kimai.

<div align="center"><img src="img/main.jpeg" alt="" width="400"></div>

## Timo Basic Principles
- Terminal-style application usable as double-click Windows app too.
- Handles login and dynamically scraping.
- Run Kimai and Timenet scraping in parallel in background.
- All scraped data is stored locally in JSON files.
- Simple and intuitive report generation from JSON files.
- Can update itself.


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
