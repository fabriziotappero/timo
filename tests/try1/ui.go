package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	focusedStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	blurredStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	cursorStyle         = focusedStyle
	noStyle             = lipgloss.NewStyle()
	helpStyle           = blurredStyle
	cursorModeHelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	focusedButton     = focusedStyle.Render("[ Retrieve ]")
	blurredButton     = fmt.Sprintf("[ %s ]", blurredStyle.Render("Retrieve"))
	focusedQuitButton = focusedStyle.Render("[ Quit ]")
	blurredQuitButton = fmt.Sprintf("[ %s ]", blurredStyle.Render("Quit"))
)

type processingDoneMsg struct {
	tableData string
	err       error
}

type model struct {
	focusIndex    int
	inputs        []textinput.Model
	cursorMode    cursor.Mode
	submitPressed bool
	spinner       spinner.Model
	showTable     bool
	tableData     string
	errorMsg      string
}

func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	m := model{
		inputs:  make([]textinput.Model, 3),
		spinner: s,
	}

	var t textinput.Model
	for i := range m.inputs {
		t = textinput.New()
		t.Cursor.Style = cursorStyle
		t.CharLimit = 32
		t.Width = 20
		switch i {
		case 0:
			t.Placeholder = "Timenet Password"
			t.Focus()
			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
			t.EchoMode = textinput.EchoPassword
			t.EchoCharacter = '•'
			t.CharLimit = 6
		case 1:
			t.Placeholder = "Kimay ID"
			t.Width = 12
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
	case processingDoneMsg:
		m.submitPressed = false
		if msg.err != nil {
			// Handle error - show existing data if available, or error message
			if msg.tableData != "" {
				m.showTable = true
				m.tableData = msg.tableData
				m.errorMsg = fmt.Sprintf("Warning: %s (showing cached data)", msg.err.Error())
			} else {
				m.errorMsg = fmt.Sprintf("Error: %s", msg.err.Error())
				m.showTable = false
			}
		} else {
			// Success - show fresh data
			m.showTable = true
			m.tableData = msg.tableData
			m.errorMsg = "" // Clear any previous error
		}
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		case "tab", "shift+tab", "enter", "up", "down", "left", "right":
			s := msg.String()
			if s == "enter" && m.focusIndex == len(m.inputs) {
				// Retrieve button pressed - try to show existing table data first
				if existingData, err := ShowTimenetTable(); err == nil {
					m.tableData = existingData
					m.showTable = true
				}
				m.submitPressed = true
				return m, tea.Batch(m.spinner.Tick, processDataCmd(m.inputs[0].Value()))
			}
			if s == "enter" && m.focusIndex == len(m.inputs)+1 {
				// Quit button pressed
				return m, tea.Quit
			}
			if s == "up" || s == "left" || s == "shift+tab" {
				m.focusIndex--
			} else {
				m.focusIndex++
			}
			if m.focusIndex > len(m.inputs)+1 {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs) + 1
			}
			cmds := make([]tea.Cmd, len(m.inputs))
			for i := 0; i <= len(m.inputs)-1; i++ {
				if i == m.focusIndex {
					cmds[i] = m.inputs[i].Focus()
					m.inputs[i].PromptStyle = focusedStyle
					m.inputs[i].TextStyle = focusedStyle
					continue
				}
				m.inputs[i].Blur()
				m.inputs[i].PromptStyle = noStyle
				m.inputs[i].TextStyle = noStyle
			}
			return m, tea.Batch(cmds...)
		}
	default:
		// Handle spinner updates when submit is pressed
		if m.submitPressed {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}
	cmd := m.updateInputs(msg)
	return m, cmd
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

	// If we have table data, display it
	if m.showTable {
		b.WriteString(m.tableData)
	}

	// Show error message if any
	if m.errorMsg != "" {
		if m.showTable {
			b.WriteString(fmt.Sprintf("\n⚠️  %s\n", m.errorMsg))
		} else {
			b.WriteString(fmt.Sprintf("⚠️  %s\n\n", m.errorMsg))
		}
	} else {
		b.WriteString("\n")
	}

	// Always show the form and buttons
	b.WriteString(m.inputs[0].View() + m.inputs[1].View() + m.inputs[2].View() + "   ")

	// Show Fetcfh button
	retrieveButton := &blurredButton
	if m.focusIndex == len(m.inputs) {
		retrieveButton = &focusedButton
	}

	// Show Quit button
	quitButton := &blurredQuitButton
	if m.focusIndex == len(m.inputs)+1 {
		quitButton = &focusedQuitButton
	}

	fmt.Fprintf(&b, "%s  %s\n", *retrieveButton, *quitButton)

	// Show spinner and message below buttons when processing
	if m.submitPressed {
		if m.showTable {
			b.WriteString(fmt.Sprintf("\n%s Updating data...", m.spinner.View()))
		} else {
			b.WriteString(fmt.Sprintf("\n%s Fetching data...", m.spinner.View()))
		}
	}

	return b.String()
}

func processDataCmd(password string) tea.Cmd {
	return func() tea.Msg {
		tableData, err := submitAction(password)
		return processingDoneMsg{tableData: tableData, err: err}
	}
}
