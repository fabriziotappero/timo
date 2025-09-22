package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// convertTimeStringToMinutes converts a time string like "9h 14m" to total minutes
func convertTimeStringToMinutes(timeStr string) (int, error) {
	// Remove extra spaces and convert to lowercase
	timeStr = strings.TrimSpace(strings.ToLower(timeStr))

	// Initialize total minutes
	totalMinutes := 0

	// Regular expression to match hours and minutes
	hoursRegex := regexp.MustCompile(`(\d+)h`)
	minutesRegex := regexp.MustCompile(`(\d+)m`)

	// Extract hours
	if hoursMatch := hoursRegex.FindStringSubmatch(timeStr); hoursMatch != nil {
		hours, err := strconv.Atoi(hoursMatch[1])
		if err != nil {
			return 0, fmt.Errorf("invalid hours format: %v", err)
		}
		totalMinutes += hours * 60
	}

	// Extract minutes
	if minutesMatch := minutesRegex.FindStringSubmatch(timeStr); minutesMatch != nil {
		minutes, err := strconv.Atoi(minutesMatch[1])
		if err != nil {
			return 0, fmt.Errorf("invalid minutes format: %v", err)
		}
		totalMinutes += minutes
	}

	// If no hours or minutes found, return error
	if totalMinutes == 0 && !hoursRegex.MatchString(timeStr) && !minutesRegex.MatchString(timeStr) {
		return 0, fmt.Errorf("invalid time format: %s (expected format like '9h 14m', '2h', or '30m')", timeStr)
	}

	return totalMinutes, nil
}

// testConvertTimeStringToMinutes tests the convertTimeStringToMinutes function with various inputs
func testConvertTimeStringToMinutes() {
	testCases := []struct {
		input    string
		expected int
		hasError bool
	}{
		{"9h 14m", 554, false},       // 9*60 + 14 = 554
		{"2h", 120, false},           // 2*60 = 120
		{" 2h  ", 120, false},        // 2*60 = 120
		{"30m", 30, false},           // 30
		{" 30m ", 30, false},         // 30
		{"1h 0m", 60, false},         // 1*60 + 0 = 60
		{"0h 45m", 45, false},        // 0*60 + 45 = 45
		{"12H 30M", 750, false},      // Case insensitive: 12*60 + 30 = 750
		{"  3h  15m  ", 195, false},  // With extra spaces: 3*60 + 15 = 195
		{"3h15m", 195, false},        // With extra spaces: 3*60 + 15 = 195
		{"3h15m       ", 195, false}, // With extra spaces: 3*60 + 15 = 195
		{"", 0, true},                // Empty string should error
		{"invalid", 0, true},         // Invalid format should error
		{"1x 2y", 0, true},           // Wrong units should error
	}

	fmt.Println("Testing convertTimeStringToMinutes function:")
	fmt.Println(strings.Repeat("=", 50))

	allPassed := true
	for i, tc := range testCases {
		result, err := convertTimeStringToMinutes(tc.input)

		// Check if error expectation matches
		hasError := err != nil
		if hasError != tc.hasError {
			fmt.Printf("❌ Test %d FAILED: input='%s' - error expectation mismatch\n", i+1, tc.input)
			fmt.Printf("   Expected error: %v, Got error: %v\n", tc.hasError, hasError)
			allPassed = false
			continue
		}

		// If we expect an error, skip result checking
		if tc.hasError {
			fmt.Printf("✅ Test %d PASSED: input='%s' - correctly returned error: %v\n", i+1, tc.input, err)
			continue
		}

		// Check result
		if result != tc.expected {
			fmt.Printf("❌ Test %d FAILED: input='%s'\n", i+1, tc.input)
			fmt.Printf("   Expected: %d minutes, Got: %d minutes\n", tc.expected, result)
			allPassed = false
		} else {
			fmt.Printf("✅ Test %d PASSED: input='%s' -> %d minutes\n", i+1, tc.input, result)
		}
	}

	fmt.Println(strings.Repeat("=", 50))
	if allPassed {
		fmt.Println("🎉 All tests PASSED!")
	} else {
		fmt.Println("❌ Some tests FAILED!")
	}
}
func main() {
	testConvertTimeStringToMinutes()
}
