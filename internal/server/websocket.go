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
	pongWait       = 60 * time.Second    //time allowed to read the next pong message from client
	pingPeriod     = (pongWait * 9) / 10 //send pings to client with this period. must be less than pongWait
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{ //upgrade HTTP connections to WebSocket connections
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now
	},
}

// Client represents a WebSocket client
type Client struct {
	ID               string
	Name             string
	Room             *Room
	conn             *websocket.Conn
	send             chan []byte
	Username         string
	Avatar           []int
	inGame           bool
	Pos              string
	CurrentRoomNumber string // Current room the player is in ("1", "2", etc.) or "" if in hallway

	// Treasure Hunt Progress
	TreasureHuntStep int
}

// Server represents the WebSocket server
type Server struct {
	roomManager *RoomManager
	userManager *UserManager
	chatManager *ChatManager
}

// NewServer creates a new WebSocket server
func NewServer() *Server {
	chatManager := NewChatManager()
	s := &Server{
		roomManager: NewRoomManager(chatManager),
		userManager: NewUserManager(),
		chatManager: chatManager,
	}

	// Setup treasure hunt broadcast
	Manager.SetUpdateCallback(func(payload protocol.TreasureHuntStatePayload) {
		// Broadcast to all rooms/clients
		// Since we don't have a direct "BroadcastAll" on RoomManager, we can iterate or 
		// rely on the fact that the next tick will pick it up.
		// Ideally, RoomManager should have a Broadcast method.
		// For now, we rely on the game loop tick in room.go to pick up the state via Manager.GetState()
		// But to be safe, we can try to broadcast if possible.
	})

	// Start the treasure hunt game loop
	go Manager.StartGameLoop()

	return s
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

		log.Printf("New user %s onboarded with avatar %v", c.Username, c.Avatar)

		// Auto-join default room
		room := s.roomManager.GetOrCreateRoom("0")
		c.Room = room
		c.inGame = true
		room.register <- c

		// --- ADDED: Send initial treasure hunt state for new users ---
		// Use global state instead of per-user step
		thMsg, _ := protocol.EncodeMessage(protocol.MsgTreasureHuntState, Manager.GetState())
		c.send <- thMsg
		// ------------------------------------------------------------

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
			user, _ := s.userManager.GetOrCreateUserByUsername(payload.Username, make([]int, 3))

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

			// Send initial treasure hunt state
			thMsg, _ := protocol.EncodeMessage(protocol.MsgTreasureHuntState, Manager.GetState())
			c.send <- thMsg

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
			// TODO: mark user not in game so they're not rendered
		}

	case protocol.MsgGlobalChat:
		var payload protocol.GlobalChatPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("Error unmarshaling global chat payload: %v", err)
			return
		}

		// Handle global chat through ChatManager
		s.chatManager.HandleGlobalChat(c, payload.Message, c.Room)

	case protocol.MsgRoomChat:
		var payload protocol.RoomChatPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("Error unmarshaling room chat payload: %v", err)
			return
		}

		// No validation - trust the client about which room they're in
		// (Server doesn't have flood-filled room map, only client does)

		// Handle room chat through ChatManager
		s.chatManager.HandleRoomChat(c, payload.RoomNumber, payload.Message, c.Room)

	case protocol.MsgAnnouncement:
		var payload protocol.AnnouncementPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("Error unmarshaling announcement payload: %v", err)
			return
		}

		// Handle global chat through ChatManager
		s.chatManager.HandleAnnouncement(payload.Message, c.Room)

	case protocol.MsgChatMessage:
		var payload protocol.ChatMessagePayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("Error unmarshaling chat message payload: %v", err)
			return
		}

		// payload.ToPlayerID is actually a username from the client
		s.chatManager.HandleDirectMessage(c, payload.ToPlayerID, payload.Message, c.Room)

	case protocol.MsgGlobalChatMessages:
		// Client requesting global chat history
		if c.Room == nil {
			return
		}

		payload := s.chatManager.GetGlobalMessages(c.Room)

		msg, err := protocol.EncodeMessage(protocol.MsgGlobalChatMessages, payload)
		if err != nil {
			return
		}

		c.send <- msg

	case protocol.MsgTreasureHuntGuess:
		var payload protocol.TreasureHuntGuessPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return
		}

		// Check answer using Username (Global Game)
		CheckTreasureHuntAnswer(c.Username, payload.Guess)

		// Send updated state
		resp, _ := protocol.EncodeMessage(protocol.MsgTreasureHuntState, Manager.GetState())
		c.send <- resp

	case protocol.MsgPlayerMove:
		var payload protocol.PlayerMovePayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("Error unmarshaling player move payload: %v", err)
			return
		}

		// Update player position in room
		if c.Room != nil {
			c.Room.UpdatePlayerPosition(c.Username, payload.NewX, payload.NewY)
		}
	}
}
