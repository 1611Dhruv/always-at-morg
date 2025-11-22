package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// updateUsernameEntry handles username entry screen
func (m Model) updateUsernameEntry(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		return m, tea.Quit

	case "enter":
		if len(m.usernameInput) > 0 {
			m.userName = m.usernameInput

			// Joiin the room
			if m.connMgr != nil && m.connMgr.IsConnected() {
				err := m.connMgr.JoinRoom(m.roomID, m.userName)
				if err != nil {
					m.err = err
					return m, nil
				}
				// Server will respond with an event, which will trigger screen transition
			}
		}
		return m, nil

	case "backspace":
		if len(m.usernameInput) > 0 {
			m.usernameInput = m.usernameInput[:len(m.usernameInput)-1]
		}

	default:
		// Add character to username (limit to 20 chars)
		if len(msg.String()) == 1 && len(m.usernameInput) < 20 {
			m.usernameInput += msg.String()
		}
	}

	return m, nil
}

// viewUsernameEntry renders the username entry screen
func (m Model) viewUsernameEntry() string {
	// Title
	title := titleStyle.Render("🎮 WELCOME TO ALWAYS AT MORG")
	subtitle := subtitleStyle.Render("A Multiplayer Terminal Adventure")

	// Prompt
	promptText := lipgloss.NewStyle().
		Foreground(secondaryColor).
		Margin(2, 0).
		Render("Enter your username:")

	// Input field with cursor
	inputText := m.usernameInput
	if len(inputText) == 0 {
		inputText = mutedStyle.Render("type here...")
	} else {
		inputText = highlightStyle.Render(inputText) + cursorStyle.Render("▊")
	}
	inputField := inputBoxStyle.Render(inputText)

	// Main content (title + input)
	mainContent := lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		subtitle,
		"\n\n\n",
		promptText,
		inputField,
	)

	// Instructions at the bottom
	instructions := instructionStyle.Render(
		"Press " + highlightStyle.Render("ENTER") + " to continue  •  " +
			mutedStyle.Render("ESC to quit"))

	// Calculate positions - main content in center, instructions at bottom
	centeredMain := lipgloss.Place(m.width, m.height-5, lipgloss.Center, lipgloss.Center, mainContent)
	bottomInstructions := lipgloss.Place(m.width, 3, lipgloss.Center, lipgloss.Bottom, instructions)

	// Combine
	return centeredMain + "\n" + bottomInstructions
}
