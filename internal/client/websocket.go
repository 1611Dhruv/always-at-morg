package client

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/yourusername/always-at-morg/internal/protocol"
)

// WSClient manages the WebSocket connection to the server
type WSClient struct {
	conn      *websocket.Conn
	send      chan []byte
	receive   chan *protocol.Message
	connected bool
	mu        sync.RWMutex
}

// NewWSClient creates a new WebSocket client
func NewWSClient(serverURL string) (*WSClient, error) {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(serverURL, nil)
	if err != nil {
		return nil, err
	}

	client := &WSClient{
		conn:      conn,
		send:      make(chan []byte, 256),
		receive:   make(chan *protocol.Message, 256),
		connected: true,
	}

	go client.readPump()
	go client.writePump()

	return client, nil
}

// readPump reads messages from the WebSocket connection
func (c *WSClient) readPump() {
	defer func() {
		c.mu.Lock()
		c.connected = false
		c.mu.Unlock()
		c.conn.Close()
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		msg, err := protocol.DecodeMessage(message)
		if err != nil {
			log.Printf("Error decoding message: %v", err)
			continue
		}

		c.receive <- msg
	}
}

// writePump writes messages to the WebSocket connection
func (c *WSClient) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// SendMessage sends a message to the server
func (c *WSClient) SendMessage(msgType protocol.MessageType, payload interface{}) error {
	msg, err := protocol.EncodeMessage(msgType, payload)
	if err != nil {
		return err
	}

	c.send <- msg
	return nil
}

// JoinRoom sends a join room request
func (c *WSClient) JoinRoom(roomID, playerName string) error {
	return c.SendMessage(protocol.MsgJoinRoom, protocol.JoinRoomPayload{
		RoomID:     roomID,
		PlayerName: playerName,
	})
}

// SendMove sends a player move
func (c *WSClient) SendMove(x, y int, direction string) error {
	return c.SendMessage(protocol.MsgPlayerMove, protocol.PlayerMovePayload{
		X:         x,
		Y:         y,
		Direction: direction,
	})
}

// SendInput sends a player input action
func (c *WSClient) SendInput(action string, data map[string]interface{}) error {
	return c.SendMessage(protocol.MsgPlayerInput, protocol.PlayerInputPayload{
		Action: action,
		Data:   data,
	})
}

// Receive returns the channel for receiving messages
func (c *WSClient) Receive() <-chan *protocol.Message {
	return c.receive
}

// IsConnected checks if the client is connected
func (c *WSClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// Close closes the WebSocket connection
func (c *WSClient) Close() error {
	close(c.send)
	return c.conn.Close()
}

// GameStateReceiver handles game state updates
type GameStateReceiver struct {
	currentState *protocol.GameState
	mu           sync.RWMutex
}

// NewGameStateReceiver creates a new game state receiver
func NewGameStateReceiver() *GameStateReceiver {
	return &GameStateReceiver{
		currentState: &protocol.GameState{
			Players:  make(map[string]protocol.Player),
			Entities: []protocol.Entity{},
		},
	}
}

// HandleMessage processes incoming messages and updates state
func (g *GameStateReceiver) HandleMessage(msg *protocol.Message) {
	switch msg.Type {
	case protocol.MsgGameState:
		var state protocol.GameState
		if err := json.Unmarshal(msg.Payload, &state); err != nil {
			log.Printf("Error unmarshaling game state: %v", err)
			return
		}
		g.UpdateState(&state)

	case protocol.MsgPlayerJoined:
		var payload protocol.PlayerJoinedPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("Error unmarshaling player joined: %v", err)
			return
		}
		g.mu.Lock()
		g.currentState.Players[payload.Player.ID] = payload.Player
		g.mu.Unlock()

	case protocol.MsgPlayerLeft:
		var payload protocol.PlayerLeftPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("Error unmarshaling player left: %v", err)
			return
		}
		g.mu.Lock()
		delete(g.currentState.Players, payload.PlayerID)
		g.mu.Unlock()

	case protocol.MsgRoomJoined:
		var payload protocol.RoomJoinedPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("Error unmarshaling room joined: %v", err)
			return
		}
		g.UpdateState(payload.GameState)
		log.Printf("Joined room %s as player %s", payload.RoomID, payload.PlayerID)

	case protocol.MsgError:
		var payload protocol.ErrorPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("Error unmarshaling error payload: %v", err)
			return
		}
		log.Printf("Server error: %s", payload.Message)
	}
}

// UpdateState updates the current game state
func (g *GameStateReceiver) UpdateState(state *protocol.GameState) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.currentState = state
}

// GetState returns the current game state
func (g *GameStateReceiver) GetState() *protocol.GameState {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.currentState
}
