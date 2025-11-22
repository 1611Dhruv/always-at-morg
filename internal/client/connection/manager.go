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

	if m.conn != nil {
		close(m.done)
		m.conn.Close()
		m.connected = false
	}
}

// IsConnected returns whether the manager is connected
func (m *Manager) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connected
}

// JoinRoom sends a join room request
func (m *Manager) JoinRoom(roomID, playerName string) error {
	return m.sendMessage(protocol.MsgJoinRoom, protocol.JoinRoomPayload{
		RoomID:     roomID,
		PlayerName: playerName,
	})
}

// SendMove sends a player move
func (m *Manager) SendMove(x, y int, direction string) error {
	return m.sendMessage(protocol.MsgPlayerMove, protocol.PlayerMovePayload{
		X:         x,
		Y:         y,
		Direction: direction,
	})
}

// SendInput sends a player input action
func (m *Manager) SendInput(action string, data map[string]interface{}) error {
	return m.sendMessage(protocol.MsgPlayerInput, protocol.PlayerInputPayload{
		Action: action,
		Data:   data,
	})
}

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
	case protocol.MsgGameState:
		var state protocol.GameState
		if err := json.Unmarshal(msg.Payload, &state); err != nil {
			log.Printf("Error unmarshaling game state: %v", err)
			return
		}
		m.state.UpdateState(&state)
		m.sendEvent(GameStateEvent{State: &state})

	case protocol.MsgPlayerJoined:
		var payload protocol.PlayerJoinedPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("Error unmarshaling player joined: %v", err)
			return
		}
		m.state.AddPlayer(payload.Player)
		m.sendEvent(PlayerJoinedEvent{Player: payload.Player})

	case protocol.MsgPlayerLeft:
		var payload protocol.PlayerLeftPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("Error unmarshaling player left: %v", err)
			return
		}
		m.state.RemovePlayer(payload.PlayerID)
		m.sendEvent(PlayerLeftEvent{PlayerID: payload.PlayerID})

	case protocol.MsgRoomJoined:
		var payload protocol.RoomJoinedPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("Error unmarshaling room joined: %v", err)
			return
		}
		m.state.UpdateState(payload.GameState)
		m.sendEvent(GameStateEvent{State: payload.GameState})
		log.Printf("Joined room %s as player %s", payload.RoomID, payload.PlayerID)

	case protocol.MsgError:
		var payload protocol.ErrorPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("Error unmarshaling error payload: %v", err)
			return
		}
		m.sendEvent(ErrorEvent{Message: payload.Message})
		log.Printf("Server error: %s", payload.Message)
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
