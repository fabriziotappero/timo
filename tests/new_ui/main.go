package main

import (
	"fmt"
	"math/rand"
	"os"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render

func createProgressBar(width int, color string) progress.Model {
	return progress.New(progress.WithSolidFill(color), progress.WithoutPercentage(), progress.WithWidth(width))
}

func main() {
	progress1 := createProgressBar(1, "#e40a1cff")

	initialBars := []progress.Model{progress1}
	initialColors := []string{"#e40a1cff"}

	p := tea.NewProgram(model{progressBars: initialBars, colors: initialColors}, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Oh no!", err)
		os.Exit(1)
	}
}

type model struct {
	progressBars []progress.Model
	colors       []string
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "a":
			// Add a new progress bar when 'a', 'enter', or 'space' is pressed
			colors := []string{"#FF0000", "#00FF00", "#0000FF", "#FFFF00", "#FF00FF", "#00FFFF"}
			randomColor := colors[rand.Intn(len(colors))]
			randomWidth := 1 + rand.Intn(10) // Random width between 1-10

			newBar := createProgressBar(randomWidth, randomColor)
			m.progressBars = append(m.progressBars, newBar)
			m.colors = append(m.colors, randomColor)

			return m, nil
		}

	default:
		return m, nil
	}
	return m, nil
}

func (m model) View() string {
	var result string

	// Display all progress bars at 100%
	for _, bar := range m.progressBars {
		result += bar.ViewAs(1.0) + " "
	}
	result += " 8h:34m\n"

	result += helpStyle("'a': add a new progress bar, 'q': quit\n")

	return result
}
