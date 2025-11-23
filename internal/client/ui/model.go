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
	GameWorldGrid   [][]string // 2D grid representing the game world (rendered from room map)

	// Loading screen
	loadingDots      int
	serverURL        string
	roomID           string // Room to join
	userName         string
	reconnectAttempt int  // Current reconnection attempt (0-5)
	maxReconnects    int  // Maximum reconnection attempts
	waitingToRetry   bool // True when waiting for retry delay

	// Chat system
	chatMode           ChatMode
	chatTarget         string              // Username for private chat
	announcements      []string            // Server-wide announcements
	globalChatMessages []string            // Global chat messages
	privateChatHistory map[string][]string // Private chat messages per user (key: username)
	chatInput          string              // Current chat input
	chatInputActive    bool                // True when typing in chat

	// Treasure Hunt
	currentClue string
	playerSelectActive bool                // True when selecting a player for private chat
	nearbyPlayers      []string            // List of nearby players for selection
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
		maxReconnects:      5,
		chatMode:           ChatModeGlobal,
		chatTarget:         "",
		announcements:      []string{"Welcome to Always at Morg!"},
		globalChatMessages: []string{},
		privateChatHistory: make(map[string][]string),
		chatInput:          "",
		chatInputActive:    false,
		currentClue:      "Loading clue...",
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
		// Cap viewport to reasonable maximums for performance
		maxWidth := 120  // Maximum viewport width
		maxHeight := 60  // Maximum viewport height

		gameWidth := int(0.8 * float64(msg.Width)) // 80% of terminal width because of chat panel
		if gameWidth > maxWidth {
			gameWidth = maxWidth
		}
		m.GameWorldWidth = gameWidth

		gameHeight := msg.Height
		if gameHeight > maxHeight {
			gameHeight = maxHeight
		}
		m.GameWorldHeight = gameHeight

		// Populate grids from game world and room map data
		m.populateGrids()

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
		// Server sent game state update - recalculate viewport and re-render
		m.viewState = ViewMainGame
		m.populateGrids() // Recalculate viewport based on current player position
		return m, listenForEventsCmd(m.connMgr, m.eventChan)

	case connection.GlobalChatMessagesEvent:
		// Receive all global chat messages from server (replace, don't append)
		m.globalChatMessages = make([]string, 0, len(e.Messages))
		for _, msg := range e.Messages {
			// Format: [Username] Message
			formattedMsg := highlightStyle.Render("["+msg.Username+"]") + " " + msg.Message
			m.globalChatMessages = append(m.globalChatMessages, formattedMsg)
		}
		return m, listenForEventsCmd(m.connMgr, m.eventChan)

	case connection.PrivateChatMessageEvent:
		// Received a private message - append to private chat history for the relevant user
		// Determine which user's chat history to update (the other person, not ourselves)
		var otherUser string
		var formattedMsg string

		if e.FromUsername == m.userName {
			// Sent by me to someone else
			otherUser = e.ToUsername
			formattedMsg = highlightStyle.Render("[You]") + " " + e.Message
		} else {
			// Received from someone else
			otherUser = e.FromUsername
			formattedMsg = highlightStyle.Render("["+e.FromUsername+"]") + " " + e.Message
		}

		// Append to this user's private chat history
		if m.privateChatHistory[otherUser] == nil {
			m.privateChatHistory[otherUser] = []string{}
		}
		m.privateChatHistory[otherUser] = append(m.privateChatHistory[otherUser], formattedMsg)
		return m, listenForEventsCmd(m.connMgr, m.eventChan)

	case connection.OnboardRequestEvent:
		// Server requests onboarding - transition to avatar customization screen
		m.viewState = ViewAvatarCustomization
		return m, listenForEventsCmd(m.connMgr, m.eventChan)

	case connection.TreasureHuntStateEvent:
		m.currentClue = e.ClueText
		return m, listenForEventsCmd(m.connMgr, m.eventChan)

	default:
		// Unknown event type - just keep listening
		return m, listenForEventsCmd(m.connMgr, m.eventChan)
	}
}
