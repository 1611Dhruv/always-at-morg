package connection

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/yourusername/always-at-morg/internal/protocol"
)

// Manager manages the WebSocket connection to the server
type Manager struct {
	serverURL     string
	conn          *websocket.Conn
	state         *State
	eventCallback func(Event)
	connected     bool
	mu            sync.RWMutex
	done          chan struct{}
}

// NewManager creates a new connection manager
func NewManager(serverURL string) *Manager {
	return &Manager{
		serverURL: serverURL,
		state:     NewState(),
		connected: false,
		done:      make(chan struct{}),
	}
}

// OnEvent sets the callback for events
func (m *Manager) OnEvent(callback func(Event)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.eventCallback = callback
}

// Connect establishes a WebSocket connection to the server
func (m *Manager) Connect() error {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(m.serverURL, nil)
	if err != nil {
		m.sendEvent(DisconnectedEvent{Error: err})
		return err
	}

	m.mu.Lock()
	m.conn = conn
	m.connected = true
	// Create a fresh done channel for this connection attempt
	// This allows reconnection to work properly
	m.done = make(chan struct{})
	m.mu.Unlock()

	// Start read/write loops
	go m.readPump()

	m.sendEvent(ConnectedEvent{})
	return nil
}

// Disconnect closes the WebSocket connection
func (m *Manager) Disconnect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Only disconnect if we're currently connected
	if !m.connected {
		return
	}

	m.connected = false

	// Close done channel to signal readPump to stop
	if m.done != nil {
		select {
		case <-m.done:
			// Already closed
		default:
			close(m.done)
		}
	}

	// Close the connection
	if m.conn != nil {
		m.conn.Close()
	}
}

// IsConnected returns whether the manager is connected
func (m *Manager) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connected
}

//// FROM CLIENT -> SERVER MESSAGES ////

// JoinRoom sends a join room request
func (m *Manager) JoinRoom(roomID, userName string) error {
	return m.sendMessage(protocol.MsgJoinRoom, protocol.JoinRoomPayload{
		RoomID:   roomID,
		Username: userName,
	})
}

func (m *Manager) SendOnboardResponse(userName string, avatar []int) error {
	return m.sendMessage(protocol.MsgOnboard, protocol.OnboardPayload{
		Name:   userName,
		Avatar: avatar,
	})
}

// Chat messages
func (m *Manager) SendGlobalChat(userName, message string) error {
	return m.sendMessage(protocol.MsgGlobalChat, protocol.GlobalChatPayload{
		Username:  userName,
		Message:   message,
		Timestamp: time.Now().Unix(),
	})
}

////////////////////////////////////////////

// GetState returns the current game state
func (m *Manager) GetState() *protocol.GameState {
	return m.state.GetState()
}

// sendMessage sends a message to the server
func (m *Manager) sendMessage(msgType protocol.MessageType, payload interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.connected || m.conn == nil {
		return websocket.ErrCloseSent
	}

	msg, err := protocol.EncodeMessage(msgType, payload)
	if err != nil {
		return err
	}

	m.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return m.conn.WriteMessage(websocket.TextMessage, msg)
}

// readPump reads messages from the WebSocket connection
func (m *Manager) readPump() {
	defer func() {
		m.mu.Lock()
		m.connected = false
		if m.conn != nil {
			m.conn.Close()
		}
		m.mu.Unlock()
		m.sendEvent(DisconnectedEvent{})
	}()

	for {
		select {
		case <-m.done:
			return
		default:
			_, message, err := m.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				}
				return
			}

			m.handleMessage(message)
		}
	}
}

// handleMessage processes incoming messages
func (m *Manager) handleMessage(data []byte) {
	msg, err := protocol.DecodeMessage(data)
	if err != nil {
		log.Printf("Error decoding message: %v", err)
		return
	}

	switch msg.Type {
	case protocol.MsgRoomJoined:
		var payload protocol.RoomJoinedPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("Error unmarshaling room joined: %v", err)
			return
		}
		m.state.UpdateState(payload.GameState)
		m.sendEvent(GameStateEvent{})
		log.Printf("Joined room %s as player %s", payload.RoomID, payload.PlayerID)

	case protocol.MsgError:
		var payload protocol.ErrorPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("Error unmarshaling error payload: %v", err)
			return
		}
		m.sendEvent(ErrorEvent{Message: payload.Message})
		log.Printf("Server error: %s", payload.Message)

	case protocol.MsgOnboardRequest:
		m.sendEvent(OnboardRequestEvent{})

	case protocol.MsgGameState:
		var payload protocol.GameState
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("Error unmarshaling game state: %v", err)
			return
		}
		m.state.UpdateState(&payload)
		m.sendEvent(GameStateEvent{})
		// log.Printf("Received game state update (tick: %d)", payload.Tick)

	case protocol.MsgKuluchifiedState:
		// Unified per-tick state update - parse and split into separate events
		var payload protocol.KuluchifiedStatePayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("Error unmarshaling kuluchified state: %v", err)
			return
		}

		// Update game state
		m.state.UpdateState(&payload.GameState)
		m.sendEvent(GameStateEvent{})

		// Send chat messages event
		if len(payload.ChatMessages) > 0 {
			messages := make([]ChatMessage, len(payload.ChatMessages))
			for i, msg := range payload.ChatMessages {
				messages[i] = ChatMessage{
					Username:  msg.Username,
					Message:   msg.Message,
					Timestamp: msg.Timestamp,
				}
			}
			m.sendEvent(GlobalChatMessagesEvent{Messages: messages})
		}

		// TODO: Handle announcements and players when needed

	case protocol.MsgGlobalChatMessages:
		var payload protocol.GlobalChatMessagesPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("Error unmarshaling global chat messages: %v", err)
			return
		}

		// Convert protocol messages to event messages
		messages := make([]ChatMessage, len(payload.Messages))
		for i, msg := range payload.Messages {
			messages[i] = ChatMessage{
				Username:  msg.Username,
				Message:   msg.Message,
				Timestamp: msg.Timestamp,
			}
		}

		m.sendEvent(GlobalChatMessagesEvent{Messages: messages})
		// log.Printf("Received %d global chat messages", len(messages))

	default:
		log.Printf("Unhandled message type: %s", msg.Type)
	}
}

// sendEvent sends an event to the callback if set
func (m *Manager) sendEvent(event Event) {
	m.mu.RLock()
	callback := m.eventCallback
	m.mu.RUnlock()

	if callback != nil {
		callback(event)
	}
}
