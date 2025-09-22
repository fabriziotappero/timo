package main

// A simple example demonstrating the use of multiple text input components
// from the Bubbles component library.

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	blurredStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	cursorStyle  = focusedStyle
	noStyle      = lipgloss.NewStyle()
	helpStyle    = blurredStyle
)

const version = "v1.2.3" // Your app version

type tableMsg struct {
	output string
}

type fetchMsg struct {
	success bool
	message string
}

type model struct {
	focusIndex      int
	inputs          []textinput.Model
	cursorMode      cursor.Mode
	loginSubmitted  bool
	showAbout       bool
	tableOutput     string
	timenetPassword string
	kimaiID         string
	kimaiPassword   string
}

func newModel() model {
	m := model{
		inputs: make([]textinput.Model, 3),
	}

	var t textinput.Model
	for i := range m.inputs {
		t = textinput.New()
		t.Cursor.Style = cursorStyle
		t.CharLimit = 64
		t.Width = 30

		switch i {
		case 0:
			t.Placeholder = "Timenet Password"
			t.Focus()
			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
			t.EchoMode = textinput.EchoPassword
			t.EchoCharacter = '•'
		case 1:
			t.Placeholder = "Kimai ID"
			t.CharLimit = 64
		case 2:
			t.Placeholder = "Kimai Password"
			t.EchoMode = textinput.EchoPassword
			t.EchoCharacter = '•'
		}

		m.inputs[i] = t
	}

	return m
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tableMsg:
		m.tableOutput = msg.output
		return m, nil
	case fetchMsg:
		slog.Info("Fetch completed", "success", msg.success, "message", msg.message)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		case "x":
			// Only handle logout if we're logged in (not typing in inputs)
			if m.loginSubmitted {
				m.loginSubmitted = false
				m.focusIndex = 0
				// Clear all input values
				for i := range m.inputs {
					m.inputs[i].SetValue("")
					m.inputs[i].Blur()
				}
				// Focus the first input
				m.inputs[0].Focus()
				m.inputs[0].PromptStyle = focusedStyle
				m.inputs[0].TextStyle = focusedStyle
				return m, nil
			}

		case "a":
			// Only handle about screen if we're logged in (not typing in inputs)
			if m.loginSubmitted {
				m.showAbout = true
				return m, nil
			}

		case "b":
			// Handle back from about screen
			if m.showAbout {
				m.showAbout = false
				return m, nil
			}

		case "l":
			if m.loginSubmitted && !m.showAbout {
				slog.Info("Loading local data...")
				return m, func() tea.Msg {
					tableOutput := BuildSummaryTable()
					//slog.Info(tableOutput)
					return tableMsg{output: tableOutput}
				}
			}

		case "f":
			if m.loginSubmitted && !m.showAbout && m.timenetPassword != "" {
				slog.Info("Fetching Timenet data...")
				return m, tea.Batch(
					func() tea.Msg {
						err := fetchTimenet(m.timenetPassword)
						if err != nil {
							return fetchMsg{success: false, message: "Timenet fetch failed: " + err.Error()}
						}
						return fetchMsg{success: true, message: "Timenet fetch completed successfully"}
					},
				)
			}

		case "c":
			// Clear table output when logged in
			if m.loginSubmitted && !m.showAbout {
				m.tableOutput = ""
				return m, nil
			}

		case "ctrl+r":
			m.cursorMode++
			if m.cursorMode > cursor.CursorHide {
				m.cursorMode = cursor.CursorBlink
			}
			cmds := make([]tea.Cmd, len(m.inputs))
			for i := range m.inputs {
				cmds[i] = m.inputs[i].Cursor.SetMode(m.cursorMode)
			}
			return m, tea.Batch(cmds...)

		// Set focus to next input
		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()

			// Submit when Enter is pressed on the last field
			if s == "enter" && m.focusIndex == len(m.inputs)-1 {
				// Store credentials
				m.timenetPassword = m.inputs[0].Value()
				m.kimaiID = m.inputs[1].Value()
				m.kimaiPassword = m.inputs[2].Value()
				m.loginSubmitted = true
				return m, nil // Don't quit, just change state
			}

			// Cycle indexes
			if s == "up" || s == "shift+tab" {
				m.focusIndex--
			} else {
				m.focusIndex++
			}

			if m.focusIndex > len(m.inputs)-1 {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs) - 1
			}

			cmds := make([]tea.Cmd, len(m.inputs))
			for i := 0; i <= len(m.inputs)-1; i++ {
				if i == m.focusIndex {
					// Set focused state
					cmds[i] = m.inputs[i].Focus()
					m.inputs[i].PromptStyle = focusedStyle
					m.inputs[i].TextStyle = focusedStyle
					continue
				}
				// Remove focused state
				m.inputs[i].Blur()
				m.inputs[i].PromptStyle = noStyle
				m.inputs[i].TextStyle = noStyle
			}

			return m, tea.Batch(cmds...)
		}
	}

	// Always update inputs when we're in login form state or about screen is not shown
	if !m.loginSubmitted || !m.showAbout {
		cmd := m.updateInputs(msg)
		return m, cmd
	}

	return m, nil
}

func (m *model) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}

	return tea.Batch(cmds...)
}

func (m model) View() string {
	var b strings.Builder

	if m.showAbout {
		// Show about screen (available from any state)
		b.WriteString(fmt.Sprintf("TIMO %s\n\n", version))
		b.WriteString("A time tracking management tool\n")
		b.WriteString("built in Golang with Bubble Tea ❤️\n\n")
		b.WriteString("Checking for new version...\n")
		b.WriteString("New Version available at: https://github.com/fabriziotappero/timo/releases\n\n")

		b.WriteString(helpStyle.Render("b back • esc leave"))

	} else if m.loginSubmitted {
		if m.tableOutput != "" {
			// Show summary table output
			b.WriteString(m.tableOutput)
			b.WriteString("\n")
			b.WriteString(helpStyle.Render("\nf fetch • c clear • esc leave • x logout • a about"))
		} else {
			b.WriteString("ready\n")
			b.WriteString(helpStyle.Render("\nf fetch • l load • esc leave • x logout • a about"))
		}

	} else {
		// Show the input form
		for i := range m.inputs {
			b.WriteString(m.inputs[i].View())
			if i < len(m.inputs)-1 {
				b.WriteRune('\n')
			}
		}
		b.WriteString(helpStyle.Render("\n\nesc leave • enter submit"))
	}

	return b.String()
}
