package client

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yourusername/always-at-morg/internal/protocol"
)

// Color palette - Earthy tones (lighter for dark backgrounds)
var (
	primaryColor   = lipgloss.Color("#E8C4A0")   // Light warm beige
	secondaryColor = lipgloss.Color("#7EBB81")   // Light forest green
	accentColor    = lipgloss.Color("#A8C9A4")   // Soft sage green
	successColor   = lipgloss.Color("#B5D99C")   // Bright sage
	mutedColor     = lipgloss.Color("#B8A890")   // Light taupe
	fgColor        = lipgloss.Color("#F5F3ED")   // Warm white
	highlightColor = lipgloss.Color("#F0DEB4")   // Cream highlight
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
)

// ViewState represents the current view in the TUI
type ViewState int

const (
	ViewUsernameEntry ViewState = iota
	ViewAvatarCustomization
	ViewMainGame
)

// Avatar presets
var (
	HeadOptions = []string{
		" ō ", // happy face
		" ^ ", // cute face
		" - ", // neutral face
		" ◡ ", // smile
		" ◉ ", // wide eyes
		" ∩ ", // arc
	}

	TorsoOptions = []string{
		"/|\\", // T-pose
		"{+}", // armored
		"<|>", // wide stance
		"[|]", // box body
		"(|)", // rounded
		"\\|/", // Y-pose
	}

	LegOptions = []string{
		"/ \\", // standing
		"| |", // straight legs
		"^ ^", // feet up
		"∧ ∧", // pointed feet
		"⌐ ⌐", // boots
		"◡ ◡", // curved
	}
)

// Avatar represents a 3x3 character avatar
type Avatar struct {
	HeadIndex  int
	TorsoIndex int
	LegsIndex  int
}

// Render returns the 3-line string representation
func (a Avatar) Render() string {
	return fmt.Sprintf("%s\n%s\n%s",
		HeadOptions[a.HeadIndex],
		TorsoOptions[a.TorsoIndex],
		LegOptions[a.LegsIndex])
}

// Model is the Bubble Tea model for the game client
type Model struct {
	viewState      ViewState
	wsClient       *WSClient
	stateReceiver  *GameStateReceiver
	playerName     string
	usernameInput  string
	roomID         string
	cursor         int
	choices        []string
	err            error
	gameStarted    bool
	statusMessage  string

	// Avatar customization
	avatar         Avatar
	avatarCursor   int // which row (0=head, 1=torso, 2=legs)

	// Terminal dimensions
	width          int
	height         int
}

// NewModel creates a new Bubble Tea model
func NewModel() Model {
	return Model{
		viewState:     ViewUsernameEntry,
		cursor:        0,
		usernameInput: "",
		avatar: Avatar{
			HeadIndex:  0,
			TorsoIndex: 0,
			LegsIndex:  0,
		},
		avatarCursor: 0,
		width:        80,
		height:       24,
	}
}

// NewModelWithView creates a model starting at a specific view (for testing)
func NewModelWithView(view ViewState) Model {
	m := NewModel()
	m.viewState = view
	// Set some defaults for testing
	if view == ViewMainGame {
		m.playerName = "TestUser"
	}
	return m
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch m.viewState {
		case ViewUsernameEntry:
			return m.updateUsernameEntry(msg)
		case ViewAvatarCustomization:
			return m.updateAvatarCustomization(msg)
		case ViewMainGame:
			return m.updateMainGame(msg)
		}

	case connectionMsg:
		m.wsClient = msg.client
		m.stateReceiver = msg.receiver
		m.statusMessage = "Connected to server"
		return m, listenForMessages(m.wsClient, m.stateReceiver)

	case gameStateMsg:
		// Game state updated, refresh view
		return m, nil

	case errorMsg:
		m.err = msg.err
		return m, nil
	}

	return m, nil
}

// updateUsernameEntry handles username entry screen
func (m Model) updateUsernameEntry(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		return m, tea.Quit

	case "enter":
		if len(m.usernameInput) > 0 {
			m.playerName = m.usernameInput
			m.viewState = ViewAvatarCustomization
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

// updateMainGame handles main game screen
func (m Model) updateMainGame(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		if m.wsClient != nil {
			m.wsClient.Close()
		}
		return m, tea.Quit

	// Movement keys (placeholder for now)
	case "up", "w":
		// TODO: Send movement to server
	case "down", "s":
		// TODO: Send movement to server
	case "left", "a":
		// TODO: Send movement to server
	case "right", "d":
		// TODO: Send movement to server
	}

	return m, nil
}

// View renders the current view
func (m Model) View() string {
	switch m.viewState {
	case ViewUsernameEntry:
		return m.viewUsernameEntry()
	case ViewAvatarCustomization:
		return m.viewAvatarCustomization()
	case ViewMainGame:
		return m.viewMainGame()
	}
	return ""
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

// viewMainGame renders the split-screen main game view
func (m Model) viewMainGame() string {
	// Calculate dimensions (70% game, 30% chat)
	gameWidth := int(float64(m.width) * 0.7)
	chatWidth := m.width - gameWidth - 10 // Account for borders and margins
	contentHeight := m.height - 10        // Leave more room for spacing

	if contentHeight < 10 {
		contentHeight = 10
	}

	// Game section
	gameTitle := lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true).
		Width(gameWidth).
		Align(lipgloss.Center).
		Render("🎮 GAME WORLD")

	gamePlaceholder := centerStyle.
		Width(gameWidth).
		Height(contentHeight).
		Align(lipgloss.Center, lipgloss.Center).
		Render("COMING SOON\n\n" + mutedStyle.Render("(Termloop will render here)"))

	gameContent := lipgloss.JoinVertical(
		lipgloss.Left,
		gameTitle,
		gamePlaceholder,
	)

	gameBox := gameBoxStyle.
		Width(gameWidth).
		Height(contentHeight + 3).
		Render(gameContent)

	// Chat section
	chatTitle := lipgloss.NewStyle().
		Foreground(accentColor).
		Bold(true).
		Width(chatWidth).
		Align(lipgloss.Center).
		Render("💬 CHAT")

	chatPlaceholder := centerStyle.
		Width(chatWidth).
		Height(contentHeight).
		Align(lipgloss.Center, lipgloss.Center).
		Render("COMING SOON\n\n" + mutedStyle.Render("(Chat messages here)"))

	chatContent := lipgloss.JoinVertical(
		lipgloss.Left,
		chatTitle,
		chatPlaceholder,
	)

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
	playerInfo := lipgloss.NewStyle().
		Foreground(successColor).
		Bold(true).
		Render("Player: " + m.playerName)

	avatarDisplay := lipgloss.NewStyle().
		Foreground(secondaryColor).
		Render(strings.ReplaceAll(m.avatar.Render(), "\n", " "))

	controls := mutedStyle.Render("WASD/Arrows: Move  •  ESC: Quit")

	statusBar := lipgloss.NewStyle().
		Foreground(fgColor).
		Width(m.width).
		Padding(1, 0).
		Align(lipgloss.Center).
		Render(playerInfo + "  " + avatarDisplay + "  •  " + controls)

	// Calculate positions
	centeredMain := lipgloss.Place(m.width, m.height-4, lipgloss.Center, lipgloss.Top, mainContent)
	bottomStatus := lipgloss.Place(m.width, 4, lipgloss.Center, lipgloss.Bottom, statusBar)

	return centeredMain + bottomStatus
}

// Messages for Bubble Tea

type connectionMsg struct {
	client   *WSClient
	receiver *GameStateReceiver
}

type gameStateMsg struct {
	state *protocol.GameState
}

type errorMsg struct {
	err error
}

// Commands for Bubble Tea

func connectToServer(serverURL, roomID, playerName string) tea.Cmd {
	return func() tea.Msg {
		client, err := NewWSClient(serverURL)
		if err != nil {
			return errorMsg{err}
		}

		receiver := NewGameStateReceiver()

		if err := client.JoinRoom(roomID, playerName); err != nil {
			return errorMsg{err}
		}

		return connectionMsg{
			client:   client,
			receiver: receiver,
		}
	}
}

func listenForMessages(client *WSClient, receiver *GameStateReceiver) tea.Cmd {
	return func() tea.Msg {
		for msg := range client.Receive() {
			receiver.HandleMessage(msg)
			return gameStateMsg{state: receiver.GetState()}
		}
		return nil
	}
}
