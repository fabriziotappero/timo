package main

import (
	"math/rand"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render

func newModel() model {
	s := spinner.New()
	s.Spinner = spinner.MiniDot

	progress1 := createProgressBar(1, "#e40a1cff")

	initialBars := []progress.Model{progress1}
	initialColors := []string{"#e40a1cff"}

	return model{
		spinner:      s,
		progressBars: initialBars,
		colors:       initialColors,
		loggedIn:     false,
	}
}

func createProgressBar(width int, color string) progress.Model {
	return progress.New(progress.WithSolidFill(color), progress.WithoutPercentage(), progress.WithWidth(width))
}

type model struct {
	spinner      spinner.Model
	progressBars []progress.Model
	colors       []string
	loggedIn     bool
	quitting     bool
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			m.quitting = true
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

	if !m.loggedIn {
		result = "Please login to Timenet/Kimai to fetch data.\n\n"
	}

	// for _, bar := range m.progressBars {
	// 	result += bar.ViewAs(1.0) + " "
	// }
	// result += " 8h:34m\n"

	result += helpStyle(" l login • q quit")

	return result
}
