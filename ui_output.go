package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var redStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
var boldStyle = lipgloss.NewStyle().Bold(true)
var yellowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("226"))

// readLatestJSON finds and reads the most recent JSON file with the given prefix
func readLatestJSON[T any](prefix string) (*T, error) {
	tempDir := os.TempDir()
	pattern := filepath.Join(tempDir, prefix+"*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to search for JSON files: %v", err)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no JSON files found in %s with prefix %s", tempDir, prefix)
	}
	sort.Slice(matches, func(i, j int) bool {
		infoI, errI := os.Stat(matches[i])
		infoJ, errJ := os.Stat(matches[j])
		if errI != nil || errJ != nil {
			return false
		}
		return infoI.ModTime().After(infoJ.ModTime())
	})
	latestFile := matches[0]
	slog.Info("Loading latest JSON file: " + latestFile)
	jsonData, err := os.ReadFile(latestFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %v", latestFile, err)
	}
	var data T
	err = json.Unmarshal(jsonData, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON from %s: %v", latestFile, err)
	}
	return &data, nil
}

// returns a summary string combining data from both Timenet and Kimai JSON files
// to be directed to the main content area of the UI
func BuildSummary(whatMonth int) string {
	timenet_data, err := readLatestJSON[TimenetData]("timenet_data_")
	if err != nil {
		return ""
	}

	kimai_data, err := readLatestJSON[KimaiData]("kimai_data_")
	if err != nil {
		return ""
	}

	// limit month navigation to what is available in the timenet JSON file
	monthCount := len(timenet_data.MonthlyData)
	whatMonth = max(0, min(whatMonth, monthCount-1))

	var result strings.Builder
	result.WriteString("------------------------- Summary -------------------------\n")

	result.WriteString(fmt.Sprintf(" %-38s%s %s\n",
		"Last Remote Fetch:", redStyle.Render(timenet_data.FetchDate),
		redStyle.Render(timenet_data.FetchTime)))

	result.WriteString(fmt.Sprintf(" %-38s%s %s\n", "Reporting Date:",
		timenet_data.MonthlyData[whatMonth].Month,
		timenet_data.Year))

	result.WriteString(fmt.Sprintf(" %-38s%s of %s\n",
		"Timenet Monthly Worked Hours:",
		timenet_data.MonthlyData[whatMonth].WorkedTimeInMonth,
		timenet_data.MonthlyData[whatMonth].ExpectedWorkedTimeInMonth))

	result.WriteString(fmt.Sprintf(" %-38s%s\n",
		"Kimai Yearly Worked Hours:",
		kimai_data.Summary.WorkedTime))

	result.WriteString(fmt.Sprintf(" %-38s%s\n\n",
		"This Year Overtime:",
		timenet_data.OvertimeInYear))

	// lets plot here a table with daily data
	result.WriteString(" Date          | Overtime | Timenet | Kimai   | Diff  \n")
	result.WriteString("-----------------------------------------------------------\n")

	var monthly_diff int = 0
	var monthly_overtime int = 0
	var monthly_timenet int = 0
	var monthly_kimai int = 0

	for _, day := range timenet_data.MonthlyData[whatMonth].DailyData {

		var kimai_worked_time int = 0

		var dayType string
		switch {
		case day.IsHoliday:
			dayType = "🎉"
		case day.IsWorkDay:
			dayType = "🧑‍💼" //🔨🔧💼🧰
		case day.IsVacation:
			dayType = "🏝️"
		default:
			dayType = "🌙" //💃🌙😎
		}

		// search for timenet_data.MonthlyData.Date inside kimai_data.MonthlyData.Date and
		// if the data match, pick kimai_data.MonthlyData.WorkedTime and add them up
		for _, kimaiDay := range kimai_data.MonthlyData {
			if kimaiDay.Date == day.Date {

				//slog.Info("Working on date: " + kimaiDay.Date + " and " + day.Date)

				kimai_minutes, err := convertTimeStringToMinutes(kimaiDay.WorkedTime)

				//slog.Info(fmt.Sprintf("Worked Time: %d", kimai_minutes))

				// there might be more than one entry for the same day, so we sum them up
				if err == nil {
					// we add up only work time, therefore only time  for days where:
					// "project" is not "Break" or
					// Activity does not contain "vacation" or
					// Activity does not contain "holiday" or
					// Activity does not contain "free time"
					if strings.ToLower(kimaiDay.Project) != "break" &&
						!strings.Contains(strings.ToLower(kimaiDay.Activity), "vacation") &&
						!strings.Contains(strings.ToLower(kimaiDay.Activity), "holiday") &&
						!strings.Contains(strings.ToLower(kimaiDay.Activity), "free time") {
						kimai_worked_time += kimai_minutes
					} else {
						slog.Info("Skipping Break entry from Kimai data for date " + kimaiDay.Date)
					}
				}
			}
		}

		// accumulate overtime over the month
		overtime, err := convertTimeStringToMinutes(day.OvertimeInDay)
		if err == nil {
			monthly_overtime += overtime
		}

		// Add to timenet total
		timenet_worked_time, err := convertTimeStringToMinutes(day.WorkedTimeInDay)
		if err == nil {
			monthly_timenet += timenet_worked_time
		}

		// accumulate kimai worked time
		monthly_kimai += kimai_worked_time

		// Calculate diff and accumulate to monthly_diff
		// TODO. Currently flexitime hours are added too, need to exclude them
		// for this we need to parse the
		daily_diff := kimai_worked_time - timenet_worked_time
		monthly_diff += daily_diff

		// add warning icon if absolute difference is > 59min
		warning := " "
		if math.Abs(float64(daily_diff)) > 59 {
			warning = yellowStyle.Render("⚡")
		}

		// TODO is this correct when it is a Flexitime day?
		kimaiWorkedTime := strings.TrimPrefix(convertMinutesToTimeString(kimai_worked_time), "+")
		diff := convertMinutesToTimeString(daily_diff)

		result.WriteString(fmt.Sprintf(" %-10s %s | %-8s | %-7s | %-7s | %-7s %s\n",
			day.Date, dayType, day.OvertimeInDay, day.WorkedTimeInDay, kimaiWorkedTime, diff, warning,
		))

	}
	result.WriteString("-----------------------------------------------------------\n")

	// Display monthly totals for each column
	result.WriteString(
		fmt.Sprintf(" %-10s %s   %-10s %-9s %-9s %-9s\n",
			"", "🎲",
			convertMinutesToTimeString(monthly_overtime),
			strings.TrimPrefix(convertMinutesToTimeString(monthly_timenet), "+"),
			strings.TrimPrefix(convertMinutesToTimeString(monthly_kimai), "+"),
			redStyle.Render(convertMinutesToTimeString(monthly_diff)+" (WTO)"),
		))

	return result.String()
}

func BuildAboutMessage() string {

	var result strings.Builder

	localMajor, localMinor, localPatch, err := ReadLocalVersion()
	var version string = ""
	if err == nil {
		version = fmt.Sprintf("%d.%d.%d", localMajor, localMinor, localPatch)
	}

	result.WriteString(fmt.Sprintf("%s v%s\n\n", boldStyle.Render("TIMO"), version))
	result.WriteString("A time tracking management tool build in\n")
	result.WriteString("Golang with the Bubble Tea ❤️ library.\n\n")
	result.WriteString("checking...\n")

	// get version from env variable
	res, err := NewVersionAvailable()
	if err != nil {
		result.WriteString("Error checking for new version.\n")
	} else if res {
		result.WriteString("🚀 new version available at: https://github.com/fabriziotappero/timo/releases\n")
	} else {
		result.WriteString("👍 you are using the latest version.\n")
	}

	result.WriteString("\nDo you want to contribute? Open an issue on GitHub.\n")

	result.WriteString(helpStyle.Render("\nb back • esc leave"))

	return result.String()
}
