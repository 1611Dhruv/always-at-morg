package client

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yourusername/always-at-morg/internal/protocol"
)

// ViewState represents the current view in the TUI
type ViewState int

const (
	ViewMenu ViewState = iota
	ViewLobby
	ViewGame
)

// Model is the Bubble Tea model for the game client
type Model struct {
	viewState      ViewState
	wsClient       *WSClient
	stateReceiver  *GameStateReceiver
	playerName     string
	roomID         string
	cursor         int
	choices        []string
	err            error
	gameStarted    bool
	statusMessage  string
}

// NewModel creates a new Bubble Tea model
func NewModel() Model {
	return Model{
		viewState: ViewMenu,
		cursor:    0,
		choices:   []string{"Join Room", "Create Room", "Quit"},
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.viewState {
		case ViewMenu:
			return m.updateMenu(msg)
		case ViewLobby:
			return m.updateLobby(msg)
		case ViewGame:
			return m.updateGame(msg)
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

// updateMenu handles menu view updates
func (m Model) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.choices)-1 {
			m.cursor++
		}

	case "enter":
		switch m.cursor {
		case 0: // Join Room
			m.viewState = ViewLobby
			m.roomID = "default-room"
			m.playerName = "Player1"
			return m, connectToServer("ws://localhost:8080/ws", m.roomID, m.playerName)

		case 1: // Create Room
			m.viewState = ViewLobby
			m.roomID = "new-room"
			m.playerName = "Host"
			return m, connectToServer("ws://localhost:8080/ws", m.roomID, m.playerName)

		case 2: // Quit
			return m, tea.Quit
		}
	}

	return m, nil
}

// updateLobby handles lobby view updates
func (m Model) updateLobby(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		if m.wsClient != nil {
			m.wsClient.Close()
		}
		return m, tea.Quit

	case "g":
		// Start game (transition to termloop)
		m.viewState = ViewGame
		m.gameStarted = true
		m.statusMessage = "Game started! Use arrow keys to move, 'q' to quit"
		return m, nil

	case "b":
		// Back to menu
		if m.wsClient != nil {
			m.wsClient.Close()
		}
		m.viewState = ViewMenu
		m.wsClient = nil
		return m, nil
	}

	return m, nil
}

// updateGame handles game view updates
func (m Model) updateGame(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		if m.wsClient != nil {
			m.wsClient.Close()
		}
		return m, tea.Quit

	case "b":
		// Back to lobby
		m.viewState = ViewLobby
		m.gameStarted = false
		return m, nil

	// Arrow keys for movement
	case "up", "w":
		if m.wsClient != nil {
			state := m.stateReceiver.GetState()
			for _, player := range state.Players {
				m.wsClient.SendMove(player.X, player.Y-1, "up")
				break
			}
		}

	case "down", "s":
		if m.wsClient != nil {
			state := m.stateReceiver.GetState()
			for _, player := range state.Players {
				m.wsClient.SendMove(player.X, player.Y+1, "down")
				break
			}
		}

	case "left", "a":
		if m.wsClient != nil {
			state := m.stateReceiver.GetState()
			for _, player := range state.Players {
				m.wsClient.SendMove(player.X-1, player.Y, "left")
				break
			}
		}

	case "right", "d":
		if m.wsClient != nil {
			state := m.stateReceiver.GetState()
			for _, player := range state.Players {
				m.wsClient.SendMove(player.X+1, player.Y, "right")
				break
			}
		}
	}

	return m, nil
}

// View renders the current view
func (m Model) View() string {
	switch m.viewState {
	case ViewMenu:
		return m.viewMenu()
	case ViewLobby:
		return m.viewLobby()
	case ViewGame:
		return m.viewGame()
	}
	return ""
}

// viewMenu renders the menu view
func (m Model) viewMenu() string {
	s := "Welcome to the Game!\n\n"

	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		s += fmt.Sprintf("%s %s\n", cursor, choice)
	}

	s += "\nUse arrow keys to navigate, Enter to select\n"

	if m.err != nil {
		s += fmt.Sprintf("\nError: %v\n", m.err)
	}

	return s
}

// viewLobby renders the lobby view
func (m Model) viewLobby() string {
	var s strings.Builder

	s.WriteString(fmt.Sprintf("Lobby: %s\n", m.roomID))
	s.WriteString(fmt.Sprintf("Player: %s\n\n", m.playerName))

	if m.statusMessage != "" {
		s.WriteString(fmt.Sprintf("Status: %s\n\n", m.statusMessage))
	}

	if m.stateReceiver != nil {
		state := m.stateReceiver.GetState()
		s.WriteString("Players in room:\n")
		for _, player := range state.Players {
			s.WriteString(fmt.Sprintf("  - %s (Score: %d)\n", player.Name, player.Score))
		}
	}

	s.WriteString("\nPress 'g' to start game, 'b' to go back, 'q' to quit\n")

	return s.String()
}

// viewGame renders the game view
func (m Model) viewGame() string {
	var s strings.Builder

	s.WriteString("Game View\n")
	s.WriteString("==========\n\n")

	if m.stateReceiver != nil {
		state := m.stateReceiver.GetState()

		// Simple ASCII representation
		s.WriteString(fmt.Sprintf("Tick: %d\n\n", state.Tick))

		s.WriteString("Players:\n")
		for _, player := range state.Players {
			s.WriteString(fmt.Sprintf("  %s @ (%d, %d) - Color: %s, Score: %d\n",
				player.Name, player.X, player.Y, player.Color, player.Score))
		}
	}

	s.WriteString("\nUse WASD or arrow keys to move, 'b' to return to lobby, 'q' to quit\n")
	s.WriteString("\nNote: For full game rendering, this view should be replaced with Termloop\n")

	return s.String()
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
