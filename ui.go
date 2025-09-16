package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type LoginData struct {
	TimenetPassword string
	KimayID         string
	KimaiPassword   string
}

var (
	spinnerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Margin(1, 0)
	dotStyle      = helpStyle.UnsetMargins()
	durationStyle = dotStyle
	appStyle      = lipgloss.NewStyle().Margin(1, 2, 0, 2)
	redStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	italicStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true)
)

type resultMsg struct {
	some_num   time.Duration
	some_text  string
	isSpinning bool
}

// Custom message type for update check result
type isNewVersionAvailable bool

// func (r resultMsg) StringWithSpinner(spinner spinner.Model) string {
// 	if r.isSpinning {
// 		return fmt.Sprintf("%s %s", spinner.View(), r.some_text)
// 	}
// 	if r.some_num == 0 {
// 		return fmt.Sprintf("• %s", r.some_text)
// 	}
// 	return fmt.Sprintf("🍔 Ate %s %s", r.some_text,
// 		durationStyle.Render(r.some_num.String()))
// }

func (r resultMsg) String() string {
	if r.isSpinning {
		return fmt.Sprintf("⟳ %s", r.some_text)
	}
	if r.some_num == 0 {
		return fmt.Sprintf("• %s", r.some_text)
	}
	return fmt.Sprintf("🍔 Eating %s %s", r.some_text,
		durationStyle.Render(r.some_num.String()))
}

type model struct {
	spinner               spinner.Model
	results               []resultMsg
	quitting              bool
	loggedIn              bool
	showLogin             bool
	loginData             LoginData
	loginStep             int  // 0: username, 1: password, 2: kimaiID
	isNewVersionAvailable bool // true if a new version is available
}

func newModel() model {
	const numLastResults = 4
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle
	return model{
		spinner:               s,
		results:               make([]resultMsg, numLastResults),
		loggedIn:              false,
		showLogin:             false,
		loginStep:             0,
		isNewVersionAvailable: false,
	}
}

func (m model) Init() tea.Cmd {
	return func() tea.Msg {
		ok, _ := NewVersionAvailable()
		return isNewVersionAvailable(ok)
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case isNewVersionAvailable:
		m.isNewVersionAvailable = bool(msg)
		return m, nil
	case tea.KeyMsg:
		// Handle login mode
		if m.showLogin {
			switch msg.String() {
			case "esc":
				m.showLogin = false
				m.loginStep = 0
				return m, nil
			case "enter":
				m.loginStep++
				if m.loginStep > 2 {
					// Login complete
					m.loggedIn = true
					m.showLogin = false
					m.loginStep = 0
				}
				return m, nil
			default:
				// Handle text input for login fields
				switch m.loginStep {
				case 0: // Timenet password
					if msg.String() == "backspace" {
						if len(m.loginData.TimenetPassword) > 0 {
							m.loginData.TimenetPassword = m.loginData.TimenetPassword[:len(m.loginData.TimenetPassword)-1]
						}
					} else if len(msg.String()) == 1 {
						m.loginData.TimenetPassword += msg.String()
					}
				case 1: // Kimai ID
					if msg.String() == "backspace" {
						if len(m.loginData.KimayID) > 0 {
							m.loginData.KimayID = m.loginData.KimayID[:len(m.loginData.KimayID)-1]
						}
					} else if len(msg.String()) == 1 {
						m.loginData.KimayID += msg.String()
					}
				case 2: // Kimai password
					if msg.String() == "backspace" {
						if len(m.loginData.KimaiPassword) > 0 {
							m.loginData.KimaiPassword = m.loginData.KimaiPassword[:len(m.loginData.KimaiPassword)-1]
						}
					} else if len(msg.String()) == 1 {
						m.loginData.KimaiPassword += msg.String()
					}
				}
				return m, nil
			}
		}

		// Handle main menu
		switch msg.String() {
		case "f":
			if m.loggedIn && len(m.loginData.TimenetPassword) > 0 {
				// First show a spinning message
				m.results = append(m.results[1:], resultMsg{
					some_text:  "Fetching Timenet data...",
					isSpinning: true,
				})

				// Run fetchTimenet in a command
				return m, func() tea.Msg {
					err := fetchTimenet(m.loginData.TimenetPassword)
					if err != nil {
						return resultMsg{some_text: "Timenet data fetch has failed: " + err.Error(), some_num: 0, isSpinning: false}
					}
					return resultMsg{some_text: "Timenet fetch completed successfully.", some_num: 0, isSpinning: false}
				}
			} else if m.loggedIn {
				// User is logged in but no Timenet password
				return m, func() tea.Msg {
					return resultMsg{some_text: "No Timenet password provided", some_num: 0, isSpinning: false}
				}
			}
			// User not logged in - do nothing or could show a message
			return m, nil
		case "l":
			if !m.loggedIn {
				m.showLogin = true
				m.loginStep = 0
			}
			return m, nil
		case "x":
			if m.loggedIn {
				m.loggedIn = false
				m.loginData = LoginData{}
			}
			return m, nil
		case "q", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		default:
			return m, nil
		}

	case resultMsg:
		m.results = append(m.results[1:], msg)
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	default:
		return m, nil
	}
}

func (m model) View() string {
	var s string

	if m.quitting {
		//s += "That's all for today, bye."
	} else if m.showLogin {

		// Login window
		prompts := []string{"Timenet Password:", "Kimai ID:", "Kimai Password:"}
		values := []string{
			strings.Repeat("*", len(m.loginData.TimenetPassword)),
			m.loginData.KimayID,
			strings.Repeat("*", len(m.loginData.KimaiPassword)),
		}

		for i, prompt := range prompts {
			if i == m.loginStep {
				s += fmt.Sprintf("▶ %s %s█\n", prompt, values[i])
			} else if i < m.loginStep {
				s += fmt.Sprintf("✓ %s %s\n", prompt, values[i])
			} else {
				s += fmt.Sprintf("  %s\n", prompt)
			}
		}

		//s += "\nPress Enter to continue, Esc to cancel."
		s += helpStyle.Render("Enter continue • Esc cancel")
	} else if m.loggedIn {

		// this is the main logged-in area

		// load local json data and show it
		tableStr, err := ShowTimenetTable()
		if err != nil {
			s += "Failed to load Timenet data.\n"
		} else {
			s += tableStr
		}

		// for _, res := range m.results {
		// 	s += res.StringWithSpinner(m.spinner) + "\n" // this is incorrect
		// }

	} else {

		if m.isNewVersionAvailable {
			s += "Please login. " + italicStyle.Render("\t\t\t🚀 Update from Github!\n")
		} else {
			s += "Please login.\n"
		}

	}

	//s += "\n"

	if !m.quitting && !m.showLogin {
		if m.loggedIn {
			s += helpStyle.Render("f fetch • x logout • q quit")
		} else {
			s += helpStyle.Render("l login • q quit")
		}
	}

	if m.quitting {
		s += "\n"
	}

	return appStyle.Render(s)
}
