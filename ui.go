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
	success  bool
	message  string
	duration time.Duration
	source   string // "timenet" or "kimai"
}

type TimedMessage struct {
	text      string
	timestamp time.Time
	duration  time.Duration
}

type clearExpiredMsg struct{}

type model struct {
	focusIndex     int
	inputs         []textinput.Model
	cursorMode     cursor.Mode
	loginSubmitted bool
	showAbout      bool

	// main UI areas
	mainContent  string         // holds the main content for data coming from JSON files
	messageQueue []TimedMessage // queue of timed messages
	maxMessages  int            // maximum number of messages to show

	// fetch tracking
	pendingFetches map[string]bool // tracks which fetches are still running

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
		inputs:         make([]textinput.Model, 3),
		spinner:        s,
		messageQueue:   make([]TimedMessage, 0),
		maxMessages:    3,
		pendingFetches: make(map[string]bool),
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

// Helper methods for message queue
func (m *model) addMessage(text string, duration time.Duration) tea.Cmd {
	msg := TimedMessage{
		text:      text,
		timestamp: time.Now(),
		duration:  duration,
	}

	m.messageQueue = append(m.messageQueue, msg)

	// Keep only maxMessages
	if len(m.messageQueue) > m.maxMessages {
		m.messageQueue = m.messageQueue[1:]
	}

	// Return a command to clear this message after its duration
	return tea.Tick(duration, func(t time.Time) tea.Msg {
		return clearExpiredMsg{}
	})
}

func (m *model) clearExpiredMessages() {
	now := time.Now()
	var active []TimedMessage

	for _, msg := range m.messageQueue {
		if now.Sub(msg.timestamp) < msg.duration {
			active = append(active, msg)
		}
	}

	m.messageQueue = active
}

func (m *model) getCurrentMessage() string {
	if m.isLoading && len(m.messageQueue) > 0 {
		return m.messageQueue[len(m.messageQueue)-1].text
	}
	if len(m.messageQueue) > 0 {
		return m.messageQueue[len(m.messageQueue)-1].text
	}
	return ""
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case mainContentMsg:
		m.mainContent = msg.output
		// Only stop loading if no pending fetches
		if len(m.pendingFetches) == 0 {
			m.isLoading = false
		}
		return m, nil

	case fetchMsg:
		slog.Info("Fetch completed", "source", msg.source, "success", msg.success, "message", msg.message)

		// Remove this fetch from pending
		delete(m.pendingFetches, msg.source)

		cmd := m.addMessage(msg.message, msg.duration)

		// Only trigger summary build if ALL fetches are complete
		if len(m.pendingFetches) == 0 {
			// All fetches done - keep only the last message and load summary
			if len(m.messageQueue) > 1 {
				m.messageQueue = m.messageQueue[len(m.messageQueue)-1:]
			}
			return m, tea.Batch(cmd, func() tea.Msg {
				summary := BuildSummary(m.monthIndex)
				return mainContentMsg{output: summary}
			})
		}

		// Some fetches still pending - just show the message
		return m, cmd

	case clearExpiredMsg:
		m.clearExpiredMessages()
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
				m.messageQueue = nil                     // Clear all messages
				m.pendingFetches = make(map[string]bool) // Clear pending fetches
				m.isLoading = false
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
				m.messageQueue = nil // Clear all messages
				return m, nil
			}

		case "l":
			if m.loginSubmitted && !m.showAbout {
				m.monthIndex = 0 // Reset to last month
				m.isLoading = true
				cmd := m.addMessage("Loaded last fetched data", 3*time.Second)
				return m, tea.Batch(
					m.spinner.Tick,
					cmd,
					func() tea.Msg {
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
				cmd := m.addMessage(fmt.Sprintf("Moved to %d months ago", m.monthIndex), 2*time.Second)
				return m, tea.Batch(
					m.spinner.Tick,
					cmd,
					func() tea.Msg {
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
				cmd := m.addMessage(fmt.Sprintf("Moved to %d months ago", m.monthIndex), 2*time.Second)
				return m, tea.Batch(
					m.spinner.Tick,
					cmd,
					func() tea.Msg {
						summary := BuildSummary(m.monthIndex)
						return mainContentMsg{output: summary}
					},
				)
			}

		case "f":
			if m.loginSubmitted && !m.showAbout {
				if m.timenetPassword == "" {
					cmd := m.addMessage("Timenet password is blank, use valid password", 5*time.Second)
					return m, cmd
				}
				if m.kimaiID == "" || m.kimaiPassword == "" {
					cmd := m.addMessage("Kimai credentials are blank, use valid credentials", 5*time.Second)
					return m, cmd
				}

				// Initialize fetch tracking
				m.pendingFetches = map[string]bool{
					"timenet": true,
					"kimai":   true,
				}
				m.isLoading = true

				cmd := m.addMessage("Fetching remote data...", 60*time.Second)
				slog.Info("Fetching remote data...")

				return m, tea.Batch(
					m.spinner.Tick,
					cmd,
					func() tea.Msg {
						err := fetchTimenet(m.timenetPassword)
						if err != nil {
							return fetchMsg{success: false, message: "Timenet fetch failed: " + err.Error(), duration: 5 * time.Second, source: "timenet"}
						}
						return fetchMsg{success: true, message: "Timenet fetch completed successfully", duration: 5 * time.Second, source: "timenet"}
					},
					func() tea.Msg {
						err := fetchKimai(m.kimaiID, m.kimaiPassword)
						if err != nil {
							return fetchMsg{success: false, message: "Kimai fetch failed: " + err.Error(), duration: 5 * time.Second, source: "kimai"}
						}
						return fetchMsg{success: true, message: "Kimai fetch completed successfully", duration: 5 * time.Second, source: "kimai"}
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
			b.WriteString(m.mainContent + "")
		} else {
			// we should not use the main content area for status messages
			b.WriteString(BuildSplashScreen())
		}

		// Show status message with spinner if loading
		currentMsg := m.getCurrentMessage()
		if m.isLoading && currentMsg != "" {
			b.WriteString(fmt.Sprintf("%s %s\n", m.spinner.View(), statusMessageStyle.Render(currentMsg)))
		} else if currentMsg != "" {
			b.WriteString(fmt.Sprintf("%s\n", statusMessageStyle.Render(currentMsg)))
		} else {
			b.WriteString("\n") // leaves a blank line when there is no status message
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
