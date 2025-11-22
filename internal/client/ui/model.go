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

// ChatMode represents the current chat mode
type ChatMode int

const (
	ChatModeGlobal ChatMode = iota
	ChatModePrivate
)

// Model is the main Bubble Tea model
type Model struct {
	viewState ViewState
	connMgr   *connection.Manager   // Single connection manager, reused throughout session
	eventChan chan connection.Event // Channel for connection events

	usernameInput string
	avatar        Avatar
	avatarCursor  int
	width         int
	height        int
	err           error

	GameWorldHeight int        // Height of the game world
	GameWorldWidth  int        // Width of the game world
	GameWorldGrid   [][]string // 2D grid representing the game world
	RoomsGrid       [][]string // 2D grid of RoomCells

	// Loading screen
	loadingDots      int
	serverURL        string
	roomID           string // Room to join
	userName         string
	reconnectAttempt int    // Current reconnection attempt (0-5)
	maxReconnects    int    // Maximum reconnection attempts
	waitingToRetry   bool   // True when waiting for retry delay

	// Chat system
	chatMode        ChatMode
	chatTarget      string   // Username for private chat
	announcements   []string // Server-wide announcements
	chatMessages    []string // Chat messages (global or private)
	chatInput       string   // Current chat input
	chatInputActive bool     // True when typing in chat
}

// NewModel creates a new Bubble Tea model with a connection manager
func NewModel(serverURL string) Model {
	// Create ONE connection manager that will be reused for the entire session
	connMgr := connection.NewManager(serverURL)

	// Create event channel for connection events
	eventChan := make(chan connection.Event, 10)

	// Set up event callback - when server sends events, push to channel
	connMgr.OnEvent(func(event connection.Event) {
		eventChan <- event
	})

	return Model{
		viewState:        ViewLoading,
		connMgr:          connMgr,
		eventChan:        eventChan,
		usernameInput:    "",
		avatar:           NewAvatar(),
		avatarCursor:     0,
		width:            80,
		height:           24,
		serverURL:        serverURL,
		roomID:           "default-room", // Default room
		loadingDots:      0,
		reconnectAttempt: 0,
		maxReconnects:    5,
		chatMode:         ChatModeGlobal,
		chatTarget:       "",
		announcements:    []string{"Welcome to Always at Morg!"},
		chatMessages:     []string{},
		chatInput:        "",
		chatInputActive:  false,
	}
}

// NewModelWithView creates a model starting at a specific view (for testing)
func NewModelWithView(view ViewState) Model {
	m := NewModel("ws://localhost:8080/ws")
	m.viewState = view
	// Set some defaults for testing
	if view == ViewMainGame {
		m.userName = "TestUser"
	}
	return m
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	// Start connection attempt on loading screen using the existing connection manager
	if m.viewState == ViewLoading && m.connMgr != nil {
		return tea.Batch(
			connectCmd(m.connMgr), // Connect to server
			tickCmd(),             // Tick for animations
			listenForEventsCmd(m.connMgr, m.eventChan), // Listen for server events
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

		// Changes dynamically the game world size based on terminal size
		m.GameWorldWidth = int(0.8 * float64(msg.Width)) // 80% of terminal width because of chat panel
		m.GameWorldHeight = msg.Height

		m.GameWorldGrid = make([][]string, m.GameWorldHeight)
		for i := range m.GameWorldGrid {
			m.GameWorldGrid[i] = make([]string, m.GameWorldWidth) // populated with tiles and entities later
		}

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
		m.reconnectAttempt = 0 // Reset retry counter
		m.waitingToRetry = false
		m.err = nil
		m.viewState = ViewUsernameEntry
		return m, nil

	case connectionErrorMsg:
		// Connection failed
		m.err = msg.err
		m.reconnectAttempt++

		// Retry if we haven't exceeded max attempts
		if m.reconnectAttempt < m.maxReconnects {
			// Wait before retrying (exponential backoff)
			m.waitingToRetry = true
			return m, tea.Batch(
				tickCmd(),
				retryConnectCmd(m.reconnectAttempt),
			)
		}

		// Max retries exceeded, stay on loading screen with error
		m.waitingToRetry = false
		return m, nil

	case retryMsg:
		// Time to retry connection after delay
		if m.viewState == ViewLoading && m.reconnectAttempt < m.maxReconnects {
			m.waitingToRetry = false
			return m, connectCmd(m.connMgr)
		}
		return m, nil

	case connectionEventMsg:
		// Server sent an event - handle it and decide which screen to show
		return m.handleConnectionEvent(msg.event)

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

// Disconnect safely disconnects the connection manager
func (m *Model) Disconnect() {
	if m.connMgr != nil {
		m.connMgr.Disconnect()
	}
}

// Add new event handlers below when you add new event types in connection/events.go
func (m Model) handleConnectionEvent(event connection.Event) (tea.Model, tea.Cmd) {
	switch e := event.(type) {

	case connection.ConnectedEvent:
		// Server connected - we already handle this in connectionSuccessMsg
		return m, listenForEventsCmd(m.connMgr, m.eventChan)

	case connection.DisconnectedEvent:
		// Lost connection - go back to loading screen
		m.viewState = ViewLoading
		m.err = e.Error
		return m, nil

	case connection.ErrorEvent:
		// Server sent error - show it but stay on current screen
		return m, tea.Batch(
			tea.Println("Server error:", e.Message),
			listenForEventsCmd(m.connMgr, m.eventChan),
		)

	// ============================================
	// GAME STATE EVENTS
	// ============================================
	case connection.GameStateEvent:
		// Server sent game state - move to avatar customization
		// (This means server accepted our username)
		m.viewState = ViewMainGame
		return m, listenForEventsCmd(m.connMgr, m.eventChan)

	// ============================================
	// CHAT EVENTS
	// ============================================
	case connection.ChatMessageEvent:
		// Received a chat message
		// TODO: Add to chat panel
		return m, listenForEventsCmd(m.connMgr, m.eventChan)

	case connection.OnboardRequestEvent:
		// Server requests onboarding - transition to avatar customization screen
		m.viewState = ViewAvatarCustomization
		return m, listenForEventsCmd(m.connMgr, m.eventChan)

	default:
		// Unknown event type - just keep listening
		return m, listenForEventsCmd(m.connMgr, m.eventChan)
	}
}
