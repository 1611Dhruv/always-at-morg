package server

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/yourusername/always-at-morg/internal/protocol"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now
	},
}

// Client represents a WebSocket client
type Client struct {
	ID   string
	Name string
	Room *Room
	conn *websocket.Conn
	send chan []byte
	Username string
	Avatar string
	inGame bool
}

// Server represents the WebSocket server
type Server struct {
	roomManager *RoomManager
	userManager *UserManager
}

// NewServer creates a new WebSocket server
func NewServer() *Server {
	return &Server{
		roomManager: NewRoomManager(),
		userManager: NewUserManager(),
	}
}

// HandleWebSocket handles WebSocket connections
func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}

	client := &Client{
		ID:   uuid.New().String(),
		conn: conn,
		send: make(chan []byte, 256),
	}

	go client.writePump()
	go client.readPump(s)
}

// readPump pumps messages from the WebSocket connection to the room
func (c *Client) readPump(s *Server) {
	defer func() {
		if c.Room != nil {
			c.Room.unregister <- c
		}
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		c.handleMessage(s, message)
	}
}

// writePump pumps messages from the room to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current WebSocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage handles incoming messages from the client
func (c *Client) handleMessage(s *Server, data []byte) {
	msg, err := protocol.DecodeMessage(data)
	if err != nil {
		log.Printf("Error decoding message: %v", err)
		return
	}

	switch msg.Type {
	case protocol.MsgOnboard:
		var payload protocol.OnboardPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("Error unmarshaling onboard payload: %v", err)
			return
		}

		// Username should already be set from MsgJoinRoom
		if c.Username == "" {
			errMsg, _ := protocol.EncodeMessage(protocol.MsgError, protocol.ErrorPayload{
				Message: "Invalid onboarding flow - username not set",
			})
			c.send <- errMsg
			return
		}

		// Create user in UserManager with username and avatar
		user, _ := s.userManager.GetOrCreateUserByUsername(c.Username, payload.Avatar)

		// Set client fields
		c.Avatar = user.Avatar
		c.Name = payload.Name

		// Auto-join default room
		room := s.roomManager.GetOrCreateRoom("0")
		c.Room = room
		c.inGame = true
		room.register <- c

		log.Printf("New user %s onboarded with avatar %s", c.Username, c.Avatar)


	case protocol.MsgJoinRoom:
		var payload protocol.JoinRoomPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("Error unmarshaling join room payload: %v", err)
			return
		}

		// Set default room ID if not specified
		if payload.RoomID == "" {
			payload.RoomID = "0"
		}

		// Check if username exists in UserManager
		if s.userManager.DoesUserExist(payload.Username) {
			// Returning user - get their profile
			user, _ := s.userManager.GetOrCreateUserByUsername(payload.Username, "")

			// Set client fields from existing user
			c.Username = user.Username
			c.Avatar = user.Avatar
			c.Name = user.Username

			// Join room
			room := s.roomManager.GetOrCreateRoom(payload.RoomID)
			c.Room = room
			c.inGame = true
			room.register <- c

			log.Printf("Returning user %s joined", user.Username)
			return
		}

		// New user - store username and request onboarding for avatar selection
		c.Username = payload.Username
		onboardRequest, _ := protocol.EncodeMessage(protocol.MsgOnboardRequest, nil)
		c.send <- onboardRequest

	case protocol.MsgLeaveRoom:
		if c.Room != nil {
			c.Room.unregister <- c
			c.Room = nil
		}

	case protocol.MsgPlayerMove:
		if c.Room == nil {
			return
		}
		var payload protocol.PlayerMovePayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("Error unmarshaling player move payload: %v", err)
			return
		}
		c.Room.UpdatePlayerPosition(c.ID, payload.X, payload.Y)

	case protocol.MsgPlayerInput:
		if c.Room == nil {
			return
		}
		var payload protocol.PlayerInputPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("Error unmarshaling player input payload: %v", err)
			return
		}
		// Handle custom input actions here
		log.Printf("Player %s action: %s", c.Name, payload.Action)
	}
}
