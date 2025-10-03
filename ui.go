package main

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	focusedStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	blurredStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	cursorStyle        = focusedStyle
	noStyle            = lipgloss.NewStyle()
	helpStyle          = blurredStyle
	statusMessageStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true)
)

type mainContentMsg struct {
	output string
}

type fetchMsg struct {
	success bool
	message string
}

type model struct {
	focusIndex     int
	inputs         []textinput.Model
	cursorMode     cursor.Mode
	loginSubmitted bool
	showAbout      bool

	// main UI areas
	mainContent   string // holds the main content for data coming from JSON files
	statusMessage string // holds status messages like "fetching data..."

	timenetPassword string
	kimaiID         string
	kimaiPassword   string

	spinner    spinner.Model
	isLoading  bool
	monthIndex int // tracks which month to display (0=current, 1=previous, etc.)
}

func newModel() model {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	m := model{
		inputs:  make([]textinput.Model, 3),
		spinner: s,
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
	case mainContentMsg:
		m.mainContent = msg.output
		m.isLoading = false
		m.statusMessage = ""
		return m, nil
	case fetchMsg:
		slog.Info("Fetch completed", "success", msg.success, "message", msg.message)
		m.isLoading = false
		m.statusMessage = msg.message

		// Auto-load main UI content after successful fetch
		if msg.success {
			return m, func() tea.Msg {
				time.Sleep(1 * time.Second) // Let the success message show briefly
				summary := BuildSummary(m.monthIndex)
				return mainContentMsg{output: summary}
			}
		}

		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

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
			// Only handle about screen if we're logged in (not typing passwords)
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
				m.monthIndex = 0 // Reset to last month
				m.isLoading = true
				m.statusMessage = "Loading last month..."
				return m, tea.Batch(
					m.spinner.Tick,
					func() tea.Msg {
						time.Sleep(200 * time.Millisecond)
						summary := BuildSummary(m.monthIndex)
						return mainContentMsg{output: summary}
					},
				)
			}
		case "left":
			if m.loginSubmitted && !m.showAbout {

				// simple hack to prevent going past available months in the current year
				// this number should depend on the data available in the timenet JSON file
				// for simplicity, we just calculate based on the current month
				monthsAvailable := int(time.Now().Month()) - 1

				if m.monthIndex < monthsAvailable { // 0-11 for 12 months
					m.monthIndex++
				}
				m.isLoading = true
				m.statusMessage = fmt.Sprintf("Loading month %d back...", m.monthIndex)
				return m, tea.Batch(
					m.spinner.Tick,
					func() tea.Msg {
						time.Sleep(200 * time.Millisecond)
						summary := BuildSummary(m.monthIndex)
						return mainContentMsg{output: summary}
					},
				)
			}
		case "right":
			if m.loginSubmitted && !m.showAbout {
				if m.monthIndex > 0 {
					m.monthIndex--
				}
				m.isLoading = true
				m.statusMessage = fmt.Sprintf("Loading month %d back...", m.monthIndex)
				return m, tea.Batch(
					m.spinner.Tick,
					func() tea.Msg {
						time.Sleep(200 * time.Millisecond)
						summary := BuildSummary(m.monthIndex)
						return mainContentMsg{output: summary}
					},
				)
			}

		case "f":
			if m.loginSubmitted && !m.showAbout {
				if m.timenetPassword == "" {
					m.statusMessage = "Timenet password is blank, use valid password."
					return m, nil
				}
				if m.kimaiID == "" || m.kimaiPassword == "" {
					m.statusMessage = "Kimai credentials are blank, use valid credentials."
					return m, nil
				}
				m.isLoading = true
				m.statusMessage = "Fetching remote data..."
				slog.Info("Fetching remote data...")
				return m, tea.Batch(
					m.spinner.Tick,
					func() tea.Msg {
						err := fetchTimenet(m.timenetPassword)
						if err != nil {
							return fetchMsg{success: false, message: "Timenet fetch failed: " + err.Error()}
						}
						time.Sleep(2 * time.Second) // Keep message visible
						return fetchMsg{success: true, message: "Timenet fetch completed successfully"}
					},
					func() tea.Msg {
						err := fetchKimai(m.kimaiID, m.kimaiPassword)
						if err != nil {
							return fetchMsg{success: false, message: "Kimai fetch failed: " + err.Error()}
						}
						time.Sleep(2 * time.Second)
						return fetchMsg{success: true, message: "Kimai fetch completed successfully"}
					},
				)
			}

		case "c":
			// Clear UI main content when logged in
			if m.loginSubmitted && !m.showAbout {
				m.mainContent = ""
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
		b.WriteString(BuildAboutMessage())

	} else if m.loginSubmitted {
		if m.mainContent != "" {

			// Show UI main content output
			b.WriteString(m.mainContent + "\n")
		} else {
			// we should not use the main content area for status messages
			b.WriteString("ready\n\n")
		}

		// Show status message with spinner if loading
		if m.isLoading && m.statusMessage != "" {
			b.WriteString(fmt.Sprintf("%s %s\n", m.spinner.View(), statusMessageStyle.Render(m.statusMessage)))
		} else if m.statusMessage != "" {
			b.WriteString(fmt.Sprintf("%s\n", statusMessageStyle.Render(m.statusMessage)))
		}

		b.WriteString(helpStyle.Render("f fetch • l load • ← → prev/next • c clear • x logout • a about"))

	} else {
		// Show the input form
		for i := range m.inputs {
			b.WriteString(m.inputs[i].View())
			if i < len(m.inputs)-1 {
				b.WriteRune('\n')
			}
		}
		b.WriteString(helpStyle.Render("\n\nenter submit • esc leave"))

	}

	return b.String()
}
