package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// updateAvatarCustomization handles avatar customization screen
func (m Model) updateAvatarCustomization(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		return m, tea.Quit

	case "up", "k":
		if m.avatarCursor > 0 {
			m.avatarCursor--
		}

	case "down", "j":
		if m.avatarCursor < 2 {
			m.avatarCursor++
		}

	case "left", "h":
		// Cycle through options for current row
		switch m.avatarCursor {
		case 0: // Head
			m.avatar.HeadIndex--
			if m.avatar.HeadIndex < 0 {
				m.avatar.HeadIndex = len(HeadOptions) - 1
			}
		case 1: // Torso
			m.avatar.TorsoIndex--
			if m.avatar.TorsoIndex < 0 {
				m.avatar.TorsoIndex = len(TorsoOptions) - 1
			}
		case 2: // Legs
			m.avatar.LegsIndex--
			if m.avatar.LegsIndex < 0 {
				m.avatar.LegsIndex = len(LegOptions) - 1
			}
		}

	case "right", "l":
		// Cycle through options for current row
		switch m.avatarCursor {
		case 0: // Head
			m.avatar.HeadIndex = (m.avatar.HeadIndex + 1) % len(HeadOptions)
		case 1: // Torso
			m.avatar.TorsoIndex = (m.avatar.TorsoIndex + 1) % len(TorsoOptions)
		case 2: // Legs
			m.avatar.LegsIndex = (m.avatar.LegsIndex + 1) % len(LegOptions)
		}

	case "enter":
		// Confirm avatar and go to main game
		m.viewState = ViewMainGame
		return m, nil
	}

	return m, nil
}

// viewAvatarCustomization renders the avatar customization screen
func (m Model) viewAvatarCustomization() string {
	// Title
	title := titleStyle.Render(fmt.Sprintf("✨ CUSTOMIZE YOUR AVATAR, %s", strings.ToUpper(m.playerName)))

	// Avatar preview with cursor indicators
	var avatarLines []string
	avatarParts := strings.Split(m.avatar.Render(), "\n")
	rowLabels := []string{"HEAD", "TORSO", "LEGS"}

	for i, part := range avatarParts {
		cursor := "  "
		if m.avatarCursor == i {
			cursor = cursorStyle.Render("▶ ")
		} else {
			cursor = mutedStyle.Render("  ")
		}

		label := lipgloss.NewStyle().
			Foreground(accentColor).
			Width(8).
			Align(lipgloss.Right).
			Render(rowLabels[i] + ":")

		avatarPart := highlightStyle.Render(part)
		if m.avatarCursor != i {
			avatarPart = optionStyle.Render(part)
		}

		avatarLines = append(avatarLines, cursor+label+"  "+avatarPart)
	}

	preview := avatarBoxStyle.Render(strings.Join(avatarLines, "\n"))

	// Options for current part
	var optionsDisplay string
	var currentOptions []string
	var currentIndex int

	switch m.avatarCursor {
	case 0:
		currentOptions = HeadOptions
		currentIndex = m.avatar.HeadIndex
	case 1:
		currentOptions = TorsoOptions
		currentIndex = m.avatar.TorsoIndex
	case 2:
		currentOptions = LegOptions
		currentIndex = m.avatar.LegsIndex
	}

	var optionParts []string
	for i, opt := range currentOptions {
		if i == currentIndex {
			optionParts = append(optionParts, selectedOptionStyle.Render("「"+opt+"」"))
		} else {
			optionParts = append(optionParts, optionStyle.Render(" "+opt+" "))
		}
	}
	optionsDisplay = strings.Join(optionParts, " ")

	optionsBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(mutedColor).
		Padding(1, 2).
		Width(60).
		Align(lipgloss.Center).
		Render(optionsDisplay)

	// Main content
	mainContent := lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		"\n\n",
		preview,
		"\n\n",
		optionsBox,
	)

	// Instructions at the bottom
	instructions := instructionStyle.Render(
		highlightStyle.Render("↑↓") + " Select  " +
			highlightStyle.Render("←→") + " Change  " +
			highlightStyle.Render("ENTER") + " Confirm  •  " +
			mutedStyle.Render("ESC Quit"))

	// Calculate positions
	centeredMain := lipgloss.Place(m.width, m.height-5, lipgloss.Center, lipgloss.Center, mainContent)
	bottomInstructions := lipgloss.Place(m.width, 3, lipgloss.Center, lipgloss.Bottom, instructions)

	return centeredMain + "\n" + bottomInstructions
}
