package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// convertTimeStringToMinutes converts a time string like "9h 14m" or "-2h 30m" to total minutes
func convertTimeStringToMinutes(timeStr string) (int, error) {
	// Remove extra spaces and convert to lowercase
	timeStr = strings.TrimSpace(strings.ToLower(timeStr))

	// Check for negative sign
	isNegative := false
	if strings.HasPrefix(timeStr, "-") {
		isNegative = true
		timeStr = strings.TrimPrefix(timeStr, "-")
		timeStr = strings.TrimSpace(timeStr) // Remove any spaces after the minus sign
	}

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

	// Apply negative sign if needed
	if isNegative {
		totalMinutes = -totalMinutes
	}

	return totalMinutes, nil
}

// convertMinutesToTimeString converts total minutes to a time string like "1h 13m" or "-2h 30m"
func convertMinutesToTimeString(totalMinutes int) string {
	if totalMinutes == 0 {
		return "0m"
	}

	// Handle negative values
	isNegative := totalMinutes < 0
	if isNegative {
		totalMinutes = -totalMinutes // Work with positive value
	}

	hours := totalMinutes / 60
	minutes := totalMinutes % 60

	var result string
	if hours == 0 {
		result = fmt.Sprintf("%dm", minutes)
	} else if minutes == 0 {
		result = fmt.Sprintf("%dh", hours)
	} else {
		result = fmt.Sprintf("%dh %dm", hours, minutes)
	}

	// Add negative sign if needed
	if isNegative {
		result = "-" + result
	}

	return result
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
		// Negative test cases
		{"-2h 30m", -150, false},  // -(2*60 + 30) = -150
		{"-1h", -60, false},       // -60
		{"-45m", -45, false},      // -45
		{"- 2h 15m", -135, false}, // Space after minus: -(2*60 + 15) = -135
		{"-0h 30m", -30, false},   // -30
		// Error cases
		{"", 0, true},        // Empty string should error
		{"invalid", 0, true}, // Invalid format should error
		{"1x 2y", 0, true},   // Wrong units should error
		{"-", 0, true},       // Just minus sign should error
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

// testConvertMinutesToTimeString tests the convertMinutesToTimeString function with various inputs
func testConvertMinutesToTimeString() {
	testCases := []struct {
		input    int
		expected string
	}{
		{0, "0m"},
		{30, "30m"},
		{60, "1h"},
		{73, "1h 13m"},
		{120, "2h"},
		{195, "3h 15m"},
		{554, "9h 14m"},
		{750, "12h 30m"},
		{1440, "24h"},
		{1441, "24h 1m"},
		// Negative test cases
		{-30, "-30m"},
		{-60, "-1h"},
		{-73, "-1h 13m"},
		{-150, "-2h 30m"},
		{-195, "-3h 15m"},
		{-1441, "-24h 1m"},
	}

	fmt.Println("\nTesting convertMinutesToTimeString function:")
	fmt.Println(strings.Repeat("=", 50))

	allPassed := true
	for i, tc := range testCases {
		result := convertMinutesToTimeString(tc.input)

		if result != tc.expected {
			fmt.Printf("❌ Test %d FAILED: input=%d minutes\n", i+1, tc.input)
			fmt.Printf("   Expected: '%s', Got: '%s'\n", tc.expected, result)
			allPassed = false
		} else {
			fmt.Printf("✅ Test %d PASSED: %d minutes -> '%s'\n", i+1, tc.input, result)
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
	testConvertMinutesToTimeString()

	// Example usage:
	fmt.Println("\nExample conversions:")
	fmt.Println(strings.Repeat("-", 30))

	// Convert string to minutes
	if minutes, err := convertTimeStringToMinutes("2h 45m"); err == nil {
		fmt.Printf("'2h 45m' = %d minutes\n", minutes)

		// Convert back to string
		timeStr := convertMinutesToTimeString(minutes)
		fmt.Printf("%d minutes = '%s'\n", minutes, timeStr)
	}

	// Test the specific example from your request
	result := convertMinutesToTimeString(73)
	fmt.Printf("73 minutes = '%s'\n", result)
}
