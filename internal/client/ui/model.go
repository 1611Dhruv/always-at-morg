package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yourusername/always-at-morg/internal/client/connection"
)

// ViewState represents the current view in the TUI
type ViewState int

const (
	ViewLoading ViewState = iota
	ViewUsernameEntry
	ViewAvatarCustomization
	ViewMainGame
)

// Model is the main Bubble Tea model
type Model struct {
	viewState     ViewState
	connMgr       *connection.Manager
	playerName    string
	usernameInput string
	avatar        Avatar
	avatarCursor  int
	width         int
	height        int
	err           error

	// Loading screen
	loadingDots int
	serverURL   string
}

// NewModel creates a new Bubble Tea model
func NewModel(serverURL string) Model {
	return Model{
		viewState:     ViewLoading,
		usernameInput: "",
		avatar:        NewAvatar(),
		avatarCursor:  0,
		width:         80,
		height:        24,
		serverURL:     serverURL,
		loadingDots:   0,
	}
}

// NewModelWithView creates a model starting at a specific view (for testing)
func NewModelWithView(view ViewState) Model {
	m := NewModel("ws://localhost:8080/ws")
	m.viewState = view
	// Set some defaults for testing
	if view == ViewMainGame {
		m.playerName = "TestUser"
	}
	return m
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	// Start connection attempt on loading screen
	if m.viewState == ViewLoading {
		return tea.Batch(
			connectCmd(m.serverURL),
			tickCmd(),
		)
	}
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
		// Route to appropriate screen update handler
		switch m.viewState {
		case ViewLoading:
			return m.updateLoading(msg)
		case ViewUsernameEntry:
			return m.updateUsernameEntry(msg)
		case ViewAvatarCustomization:
			return m.updateAvatarCustomization(msg)
		case ViewMainGame:
			return m.updateMainGame(msg)
		}

	case connectionSuccessMsg:
		// Connection successful, move to username entry
		m.viewState = ViewUsernameEntry
		return m, nil

	case connectionErrorMsg:
		// Connection failed, stay on loading screen with error
		m.err = msg.err
		return m, nil

	case tickMsg:
		// Update loading animation
		if m.viewState == ViewLoading {
			m.loadingDots = (m.loadingDots + 1) % 4
			return m, tickCmd()
		}
		return m, nil
	}

	return m, nil
}

// View renders the current view
func (m Model) View() string {
	switch m.viewState {
	case ViewLoading:
		return m.viewLoading()
	case ViewUsernameEntry:
		return m.viewUsernameEntry()
	case ViewAvatarCustomization:
		return m.viewAvatarCustomization()
	case ViewMainGame:
		return m.viewMainGame()
	}
	return ""
}
