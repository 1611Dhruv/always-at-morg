package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// updateLoading handles loading screen updates
func (m Model) updateLoading(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc", "q":
		return m, tea.Quit
	}
	return m, nil
}

// viewLoading renders the loading/connection screen
func (m Model) viewLoading() string {
	// Title
	title := titleStyle.Render("🌲 ALWAYS AT MORG")
	subtitle := subtitleStyle.Render("Connecting to the hall...")

	// Animated loading dots
	dots := strings.Repeat(".", m.loadingDots)
	spinner := spinnerStyle.Render("◐◓◑◒"[m.loadingDots%4 : m.loadingDots%4+1])

	loadingText := lipgloss.NewStyle().
		Foreground(mutedColor).
		Render("Establishing connection" + dots)

	status := lipgloss.JoinVertical(
		lipgloss.Center,
		spinner+" "+loadingText,
	)

	// Error message if connection failed
	var errorMsg string
	if m.err != nil {
		errorMsg = errorStyle.Render("\n\n✗ Connection failed: " + m.err.Error())
		retryMsg := mutedStyle.Render("\nPress ESC to quit")
		errorMsg = errorMsg + retryMsg
	}

	// Main content
	mainContent := lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		subtitle,
		"\n\n",
		status,
		errorMsg,
	)

	// Instructions at bottom
	instructions := instructionStyle.Render(
		mutedStyle.Render("Connecting to ") + highlightStyle.Render(m.serverURL) + "  •  " +
			mutedStyle.Render("ESC to quit"))

	// Layout
	centeredMain := lipgloss.Place(m.width, m.height-5, lipgloss.Center, lipgloss.Center, mainContent)
	bottomInstructions := lipgloss.Place(m.width, 3, lipgloss.Center, lipgloss.Bottom, instructions)

	return centeredMain + "\n" + bottomInstructions
}
