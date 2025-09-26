package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var redStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
var boldStyle = lipgloss.NewStyle().Bold(true)

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

func BuildSummary() string {
	timenet_data, err := readLatestJSON[TimenetData]("timenet_data_")
	if err != nil {
		return ""
	}

	kimai_data, err := readLatestJSON[KimaiData]("kimai_data_")
	if err != nil {
		return ""
	}

	var result strings.Builder
	result.WriteString("------------------------ Summary ------------------------\n")
	result.WriteString(fmt.Sprintf(" Last Update:                   %s %s\n", redStyle.Render(timenet_data.Date), redStyle.Render(timenet_data.Time)))
	result.WriteString(fmt.Sprintf(" Reporting Date:                %s\n", timenet_data.Summary.ReportingDate))
	result.WriteString(fmt.Sprintf(" Required Monthly Hours:        %s\n", timenet_data.Summary.ExpectedHoursInMonth))
	result.WriteString(fmt.Sprintf(" Timenet Monthly Clocked Hours: %s\n", timenet_data.Summary.WorkedHoursInMonth))
	result.WriteString(fmt.Sprintf(" Kimai Yearly Clocked Hours:    %s\n", kimai_data.Summary.WorkedHours))
	result.WriteString(fmt.Sprintf(" Yearly Overtime:               %s\n\n", timenet_data.Summary.AccumuletedHoursInYear))
	//result.WriteString("==========================================\n\n")

	// lets plot here a table with daily data
	result.WriteString(" Date          | Overtime | Timenet | Kimai   | Diff  \n")
	result.WriteString("---------------------------------------------------------\n")

	var monthly_diff_min int

	for _, day := range timenet_data.MonthlyData {

		var dayType string
		switch {
		case day.IsHoliday:
			dayType = "🎉"
		case day.IsWorkingDay:
			dayType = "👷🏽‍♀️"
		case day.IsVacation:
			dayType = "🏝️"
		default:
			dayType = "💃"
		}

		// search for timenet_data.MonthlyData.Date inside kimai_data.MonthlyData.Date and
		// if the data match, pick kimai_data.MonthlyData.WorkedHours and add them up
		var kimaiWorkedHours string = ""
		var kimaiWorkedHours_int int = 0
		for _, kimaiDay := range kimai_data.MonthlyData {
			if kimaiDay.Date == day.Date {
				// there might be more than one entry for the same day, so we sum them up
				minutes, err := convertTimeStringToMinutes(kimaiDay.WorkedHours)
				if err == nil {
					kimaiWorkedHours_int += minutes
				}
			}
		}
		kimaiWorkedHours = convertMinutesToTimeString(kimaiWorkedHours_int)

		// work out the difference between timenet and kimai logged hours per each
		// day as well as the monthly total difference
		var diff string = ""
		result1, err := convertTimeStringToMinutes(day.WorkedHours)
		if err == nil {
			result2, err := convertTimeStringToMinutes(kimaiWorkedHours)
			if err == nil {
				diff_num := result2 - result1
				diff = convertMinutesToTimeString(diff_num)
				monthly_diff_min += diff_num
			}
		}

		result.WriteString(fmt.Sprintf(" %-10s %s | %-8s | %-7s | %-7s | %-7s\n",
			day.Date, dayType, day.Overtime, day.WorkedHours, kimaiWorkedHours, diff,
		))
	}
	result.WriteString("---------------------------------------------------------\n")

	// total monthly difference
	monthly_diff := convertMinutesToTimeString(monthly_diff_min)
	result.WriteString(fmt.Sprintf("%65s\n", redStyle.Render(monthly_diff)))

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
	result.WriteString("A time tracking management tool build\n")
	result.WriteString("in Golang with Bubble Tea ❤️\n\n")
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

	result.WriteString(helpStyle.Render("\nb back • esc leave"))

	return result.String()
}
