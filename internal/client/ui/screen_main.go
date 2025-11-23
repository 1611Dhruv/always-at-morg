package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// updateMainGame handles main game screen
func (m Model) updateMainGame(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle chat input if active
	if m.chatInputActive {
		switch msg.String() {
		case "esc":
			// Cancel chat input
			m.chatInputActive = false
			m.chatInput = ""
			return m, nil

		case "enter":
			// Send message
			if len(m.chatInput) > 0 {
				if m.connMgr != nil && m.connMgr.IsConnected() {
					if m.chatMode == ChatModeGlobal {
						// Send global chat message
						m.connMgr.SendGlobalChat(m.userName, m.chatInput)
					}
				}
				// Clear input but stay in chat mode
				m.chatInput = ""
			}
			return m, nil

		case "backspace":
			if len(m.chatInput) > 0 {
				m.chatInput = m.chatInput[:len(m.chatInput)-1]
			}
			return m, nil

		case " ":
			// Handle space explicitly
			if len(m.chatInput) < 100 {
				m.chatInput += " "
			}
			return m, nil

		default:
			// Add character to input (limit to 100 chars)
			// Use Runes to properly handle shift+letter for capitals
			if msg.Type == tea.KeyRunes && len(m.chatInput) < 100 {
				for _, r := range msg.Runes {
					m.chatInput += string(r)
				}
			}
			return m, nil
		}
	}

	// Normal game controls
	switch msg.String() {
	case "ctrl+c":
		if m.connMgr != nil {
			m.connMgr.Disconnect()
		}
		return m, tea.Quit

	// Chat controls
	case "t", "T":
		// Start typing in chat
		m.chatInputActive = true
		m.chatInput = ""
		return m, nil

	case "g", "G":
		// Switch to global chat
		m.chatMode = ChatModeGlobal
		m.chatTarget = ""
		return m, nil

	case "p", "P":
		// Switch to private chat (for now, set to placeholder)
		m.chatMode = ChatModePrivate
		// TODO: Implement player selection UI
		m.chatTarget = "Player2" // Placeholder
		return m, nil

	// Movement keys (placeholder for now - will be handled by game renderer)
	case "up", "w":
		// TODO: Send movement to server via connection manager
		// if m.connMgr != nil && m.connMgr.IsConnected() {
		//     m.connMgr.SendMove(x, y-1, "up")
		// }
	case "down", "s":
		// TODO: Send movement to server
	case "left", "a":
		// TODO: Send movement to server
	case "right", "d":
		// TODO: Send movement to server
	}

	return m, nil
}

// viewMainGame renders the split-screen main game view
func (m Model) viewMainGame() string {
	// Calculate dimensions (70% game, 30% chat)
	gameWidth := int(float64(m.width) * 0.7)
	chatWidth := m.width - gameWidth - 10 // Account for borders and margins
	contentHeight := m.height - 10        // Leave more room for spacing

	if contentHeight < 10 {
		contentHeight = 10
	}

	// Chat section (right 30%)
	chatPanelHeight := contentHeight - 4 // Leave room for input box below
	chatContent := m.renderChatPanel(chatWidth, chatPanelHeight)
	chatBox := chatBoxStyle.
		Width(chatWidth).
		Height(chatPanelHeight).
		Render(chatContent)

	// Chat input box (below chat panel, adapts to chat width)
	chatInputBox := m.renderChatInputBox(chatWidth)

	// Combine chat panel and input box vertically
	chatSection := lipgloss.JoinVertical(
		lipgloss.Left,
		chatBox,
		chatInputBox,
	)

	// Game section (left 70%) - extends to match full chat section height
	gameContent := m.renderGamePanel(gameWidth, contentHeight)
	gameBox := gameBoxStyle.
		Width(gameWidth).
		Height(contentHeight + 3).
		Render(gameContent)

	// Join game and chat sections horizontally
	mainContent := lipgloss.JoinHorizontal(
		lipgloss.Top,
		gameBox,
		chatSection,
	)

	// Status bar at the bottom
	statusBar := m.renderStatusBar()

	// Calculate positions
	centeredMain := lipgloss.Place(m.width, m.height-4, lipgloss.Center, lipgloss.Top, mainContent)
	bottomStatus := lipgloss.Place(m.width, 4, lipgloss.Center, lipgloss.Bottom, statusBar)

	return centeredMain + bottomStatus
}

// renderGamePanel renders the game world panel (left 70%)
func (m Model) renderGamePanel(width, height int) string {
	gameTitle := lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true).
		Width(width).
		Align(lipgloss.Center).
		Render("GAME WORLD")

	// Calculate actual viewport dimensions (accounting for borders and padding)
	viewportWidth := width - 4
	viewportHeight := height - 2

	// Render the actual game grid
	gameGrid := m.renderGameWorld(viewportWidth, viewportHeight)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		gameTitle,
		gameGrid,
	)
}

// renderGameWorld creates a grid representation of the game world
func (m Model) renderGameWorld(width, height int) string {
	var builder strings.Builder

	// Initialize GameWorldGrid if not already done
	if len(m.GameWorldGrid) == 0 || len(m.GameWorldGrid) != height {
		// This will be updated in the Update() function on WindowSizeMsg
		// For now, just render empty grid
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				builder.WriteString(".")
			}
			if y < height-1 {
				builder.WriteString("\n")
			}
		}
		return builder.String()
	}

	// Render the actual grid
	for y := 0; y < height && y < len(m.GameWorldGrid); y++ {
		for x := 0; x < width && x < len(m.GameWorldGrid[y]); x++ {
			if m.GameWorldGrid[y][x] == "" {
				builder.WriteString(".")
			} else {
				builder.WriteString(m.GameWorldGrid[y][x])
			}
		}
		if y < height-1 {
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

// renderChatPanel renders the chat panel (right 30%)
func (m Model) renderChatPanel(width, height int) string {
	// Split chat panel vertically: 30% announcements, 70% chat
	announcementHeight := int(float64(height) * 0.3)
	chatHeight := height - announcementHeight - 2 // -2 for borders

	// Render announcement section (top 30%)
	announcementSection := m.renderAnnouncementsSection(width, announcementHeight)

	// Render chat section (bottom 70%)
	chatSection := m.renderChatSection(width, chatHeight)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		announcementSection,
		chatSection,
	)
}

// renderAnnouncementsSection renders the announcements area
func (m Model) renderAnnouncementsSection(width, height int) string {
	// Title
	announcementTitle := lipgloss.NewStyle().
		Foreground(accentColor).
		Bold(true).
		Width(width).
		Align(lipgloss.Center).
		Render("ANNOUNCEMENTS")

	// Announcements content
	var announcementLines []string
	displayCount := height - 3 // Reserve space for title and padding
	if displayCount < 1 {
		displayCount = 1
	}

	// Show most recent announcements
	startIdx := 0
	if len(m.announcements) > displayCount {
		startIdx = len(m.announcements) - displayCount
	}

	for i := startIdx; i < len(m.announcements); i++ {
		announcementLines = append(announcementLines, mutedStyle.Render("• "+m.announcements[i]))
	}

	// If no announcements, show placeholder
	if len(announcementLines) == 0 {
		announcementLines = append(announcementLines, mutedStyle.Render("No announcements"))
	}

	announcementContent := lipgloss.NewStyle().
		Width(width).
		Height(height-1).
		Padding(0, 1).
		Render(strings.Join(announcementLines, "\n"))

	return lipgloss.JoinVertical(
		lipgloss.Left,
		announcementTitle,
		announcementContent,
	)
}

// renderChatSection renders the chat messages area
func (m Model) renderChatSection(width, height int) string {
	// Chat mode indicator
	var modeIndicator string
	if m.chatMode == ChatModeGlobal {
		modeIndicator = highlightStyle.Render("[GLOBAL]") + mutedStyle.Render(" Press 'p' for private")
	} else {
		target := m.chatTarget
		if target == "" {
			target = "none"
		}
		modeIndicator = highlightStyle.Render("[PRIVATE: "+target+"]") + mutedStyle.Render(" Press 'g' for global")
	}

	chatTitle := lipgloss.NewStyle().
		Foreground(accentColor).
		Bold(true).
		Width(width).
		Align(lipgloss.Center).
		Render("CHAT")

	modeBar := lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		Render(modeIndicator)

	// Chat messages
	var messageLines []string
	displayCount := height - 3 // Reserve space for title, mode bar, padding (no input box here)
	if displayCount < 1 {
		displayCount = 1
	}

	// Show most recent messages
	startIdx := 0
	if len(m.chatMessages) > displayCount {
		startIdx = len(m.chatMessages) - displayCount
	}

	for i := startIdx; i < len(m.chatMessages); i++ {
		messageLines = append(messageLines, m.chatMessages[i])
	}

	// If no messages, show placeholder
	if len(messageLines) == 0 {
		messageLines = append(messageLines, mutedStyle.Render("No messages yet. Press 't' to type."))
	}

	messagesContent := lipgloss.NewStyle().
		Width(width).
		Height(displayCount).
		Padding(0, 1).
		Render(strings.Join(messageLines, "\n"))

	return lipgloss.JoinVertical(
		lipgloss.Left,
		chatTitle,
		modeBar,
		messagesContent,
	)
}

// renderChatInputBox renders the chat input box (adapts to width)
func (m Model) renderChatInputBox(width int) string {
	inputPrefix := "> "
	inputText := m.chatInput

	// Always ensure we have content to maintain consistent height
	if m.chatInputActive {
		if inputText == "" {
			inputText = cursorStyle.Render("|")
		} else {
			inputText += cursorStyle.Render("|")
		}
	} else {
		if inputText == "" {
			inputText = mutedStyle.Render("Press 't' to type...")
		}
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(1). // Fixed height to prevent shifting
		Border(lipgloss.RoundedBorder()).
		BorderForeground(mutedColor).
		Padding(0, 1).
		Render(inputPrefix + inputText)
}

// renderStatusBar renders the bottom status bar
func (m Model) renderStatusBar() string {
	playerInfo := lipgloss.NewStyle().
		Foreground(successColor).
		Bold(true).
		Render("Player: " + m.userName)

	avatarDisplay := lipgloss.NewStyle().
		Foreground(secondaryColor).
		Render(strings.ReplaceAll(m.avatar.Render(), "\n", " "))

	var controls string
	if m.chatInputActive {
		controls = mutedStyle.Render("ENTER: Send  •  ESC: Cancel")
	} else {
		controls = mutedStyle.Render("T: Chat  •  G/P: Mode  •  CTRL+C: Quit")
	}

	return lipgloss.NewStyle().
		Foreground(fgColor).
		Width(m.width).
		Padding(1, 0).
		Align(lipgloss.Center).
		Render(playerInfo + "  " + avatarDisplay + "  •  " + controls)
}
