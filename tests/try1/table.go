// table.go
package main

import (
	"fmt"
	"strings"
)

// ShowTimenetTable reads the latest JSON file and returns a formatted table string
func ShowTimenetTable() (string, error) {
	data, err := readLatestTimenetJSON()
	if err != nil {
		return "", fmt.Errorf("failed to read JSON data: %v", err)
	}

	var result strings.Builder
	result.WriteString("\n========== Timenet Summary ==========\n")
	result.WriteString(fmt.Sprintf("Current Date:      %s\n", data.Date))
	result.WriteString(fmt.Sprintf("Current Time:      %s\n", data.Time))
	result.WriteString(fmt.Sprintf("Reporting Period:  %s\n", data.Summary.MesAno))
	result.WriteString(fmt.Sprintf("Required Hours:    %s\n", data.Summary.HorasPrevistas))
	result.WriteString(fmt.Sprintf("Clocked Hours:     %s\n", data.Summary.HorasTrabajadas))
	result.WriteString(fmt.Sprintf("Total Overtime:    %s\n", data.Summary.AcumuladoAno))
	result.WriteString("=====================================\n")

	return result.String(), nil
}
