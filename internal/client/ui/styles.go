package ui

import "github.com/charmbracelet/lipgloss"

// Color palette - Earthy tones (lighter for dark backgrounds)
var (
	primaryColor   = lipgloss.Color("#E8C4A0") // Light warm beige
	secondaryColor = lipgloss.Color("#7EBB81") // Light forest green
	accentColor    = lipgloss.Color("#A8C9A4") // Soft sage green
	successColor   = lipgloss.Color("#B5D99C") // Bright sage
	mutedColor     = lipgloss.Color("#B8A890") // Light taupe
	fgColor        = lipgloss.Color("#F5F3ED") // Warm white
	highlightColor = lipgloss.Color("#F0DEB4") // Cream highlight
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			Padding(1, 2).
			Align(lipgloss.Center)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Italic(true).
			Align(lipgloss.Center)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2).
			Margin(1, 0)

	inputBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(accentColor).
			Padding(0, 1).
			Width(30)

	highlightStyle = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true)

	mutedStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	instructionStyle = lipgloss.NewStyle().
				Foreground(mutedColor).
				Italic(true).
				Margin(1, 0)

	avatarBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(secondaryColor).
			Padding(1, 3).
			Align(lipgloss.Center)

	optionStyle = lipgloss.NewStyle().
			Foreground(fgColor).
			Padding(0, 1)

	selectedOptionStyle = lipgloss.NewStyle().
				Foreground(successColor).
				Bold(true).
				Padding(0, 1)

	cursorStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	gameBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1)

	chatBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accentColor).
			Padding(1)

	centerStyle = lipgloss.NewStyle().
			Align(lipgloss.Center).
			Foreground(mutedColor).
			Italic(true)

	spinnerStyle = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E07B7B")).
			Bold(true)
)
