package ui

import (
	"fmt"
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
	// Simple title
	title := titleStyle.Render("ALWAYS AT MORG")

	// Animated dots
	dots := strings.Repeat(".", (m.loadingDots % 3) + 1)
	spaces := strings.Repeat(" ", 3 - (m.loadingDots % 3))

	// Connection status
	var statusText string
	if m.err != nil {
		if m.reconnectAttempt < m.maxReconnects {
			if m.waitingToRetry {
				// Calculate retry delay (exponential backoff)
				delay := 1 << uint(m.reconnectAttempt-1)
				if delay > 10 {
					delay = 10
				}
				statusText = lipgloss.NewStyle().
					Foreground(mutedColor).
					Render(fmt.Sprintf("Waiting %ds before retry %d/%d%s%s", delay, m.reconnectAttempt, m.maxReconnects, dots, spaces))
			} else {
				statusText = lipgloss.NewStyle().
					Foreground(mutedColor).
					Render(fmt.Sprintf("Retrying connection (%d/%d)%s%s", m.reconnectAttempt, m.maxReconnects, dots, spaces))
			}
		} else {
			statusText = errorStyle.Render(fmt.Sprintf("Connection failed after %d attempts", m.maxReconnects))
		}
	} else {
		statusText = lipgloss.NewStyle().
			Foreground(mutedColor).
			Render("Connecting" + dots + spaces)
	}

	// Main content - just title and status
	mainContent := lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		"\n\n",
		statusText,
	)

	// Simple instructions
	instructions := mutedStyle.Render("ESC to quit")

	// Layout
	centeredMain := lipgloss.Place(m.width, m.height-3, lipgloss.Center, lipgloss.Center, mainContent)
	bottomInstructions := lipgloss.Place(m.width, 2, lipgloss.Center, lipgloss.Bottom, instructions)

	return centeredMain + "\n" + bottomInstructions
}
