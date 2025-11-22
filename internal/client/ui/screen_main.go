package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// updateMainGame handles main game screen
func (m Model) updateMainGame(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		if m.connMgr != nil {
			m.connMgr.Disconnect()
		}
		return m, tea.Quit

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

	// Game section (left 70%)
	gameContent := m.renderGamePanel(gameWidth, contentHeight)
	gameBox := gameBoxStyle.
		Width(gameWidth).
		Height(contentHeight + 3).
		Render(gameContent)

	// Chat section (right 30%)
	chatContent := m.renderChatPanel(chatWidth, contentHeight)
	chatBox := chatBoxStyle.
		Width(chatWidth).
		Height(contentHeight + 3).
		Render(chatContent)

	// Join game and chat horizontally
	mainContent := lipgloss.JoinHorizontal(
		lipgloss.Top,
		gameBox,
		chatBox,
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
		Render("🎮 GAME WORLD")

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
	chatTitle := lipgloss.NewStyle().
		Foreground(accentColor).
		Bold(true).
		Width(width).
		Align(lipgloss.Center).
		Render("💬 CHAT")

	// TODO: Chat messages will be rendered here
	chatPlaceholder := centerStyle.
		Width(width).
		Height(height-2).
		Align(lipgloss.Center, lipgloss.Center).
		Render("COMING SOON\n\n" + mutedStyle.Render("(Chat messages here)"))

	return lipgloss.JoinVertical(
		lipgloss.Left,
		chatTitle,
		chatPlaceholder,
	)
}

// renderStatusBar renders the bottom status bar
func (m Model) renderStatusBar() string {
	playerInfo := lipgloss.NewStyle().
		Foreground(successColor).
		Bold(true).
		Render("Player: " + m.playerName)

	avatarDisplay := lipgloss.NewStyle().
		Foreground(secondaryColor).
		Render(strings.ReplaceAll(m.avatar.Render(), "\n", " "))

	controls := mutedStyle.Render("WASD/Arrows: Move  •  ESC: Quit")

	return lipgloss.NewStyle().
		Foreground(fgColor).
		Width(m.width).
		Padding(1, 0).
		Align(lipgloss.Center).
		Render(playerInfo + "  " + avatarDisplay + "  •  " + controls)
}