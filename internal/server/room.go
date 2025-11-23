package server

import (
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/always-at-morg/internal/protocol"
)

// Room represents a game room/session
type Room struct {
	ID          string
	Clients     map[string]*Client
	GameState   *protocol.GameState
	chatManager *ChatManager

	mu         sync.RWMutex
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	tickRate   time.Duration
}

// NewRoom creates a new game room
func NewRoom(id string, chatManager *ChatManager) *Room {
	return &Room{
		ID:          id,
		Clients:     make(map[string]*Client),
		GameState:   &protocol.GameState{Tick: 0},
		chatManager: chatManager,

		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		tickRate:   time.Millisecond * 50, // 20 ticks per second
	}
}

// Run starts the room's main loop
func (r *Room) Run() {
	ticker := time.NewTicker(r.tickRate)
	defer ticker.Stop()

	for {
		select {
		case client := <-r.register:
			r.handleRegister(client)

		case client := <-r.unregister:
			r.handleUnregister(client)

		case message := <-r.broadcast:
			r.handleBroadcast(message)

		case <-ticker.C:
			r.update(r.chatManager)
		}
	}
}

func (r *Room) handleRegister(client *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Clients[client.ID] = client

	log.Printf("Player %s joined room %s", client.Name, r.ID)

	// Send room joined message to the new client
	msg, _ := protocol.EncodeMessage(protocol.MsgRoomJoined, protocol.RoomJoinedPayload{
		RoomID:    r.ID,
		PlayerID:  client.ID,
		GameState: r.GameState,
	})
	client.send <- msg

	// Broadcast player joined to others
}

func (r *Room) handleUnregister(client *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.Clients[client.ID]; ok {
		delete(r.Clients, client.ID)
		close(client.send)

		log.Printf("Player %s left room %s", client.Name, r.ID)

	}
}

func (r *Room) handleBroadcast(message []byte) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, client := range r.Clients {
		select {
		case client.send <- message:
		default:
			close(client.send)
			delete(r.Clients, client.ID)
		}
	}
}

// update runs the game logic and broadcasts state
func (r *Room) update(chatManager *ChatManager) {
	r.mu.Lock()
	r.GameState.Tick++

	// Add game logic here (e.g., entity movement, collision detection)

	r.mu.Unlock()

	// Broadcast announcements
	announcements := chatManager.GetAnnouncements()
	for _, announcement := range announcements {
		payload := protocol.AnnouncementPayload{
			Message:   announcement.Message,
			Timestamp: announcement.Timestamp,
		}
		announcementMsg, _ := protocol.EncodeMessage(protocol.MsgAnnouncement, payload)
		r.broadcast <- announcementMsg
	}

	// Broadcast global chat messages
	messages := chatManager.GetGlobalMessages(r)
	if len(messages) > 0 {
		payload := protocol.GlobalChatMessagesPayload{
			Messages: messages,
		}
		chatMsg, _ := protocol.EncodeMessage(protocol.MsgGlobalChatMessages, payload)
		r.broadcast <- chatMsg
	}

	// Broadcast game state to all clients
	msg, _ := protocol.EncodeMessage(protocol.MsgGameState, r.GameState)
	r.broadcast <- msg
}

// UpdatePlayerPosition updates a player's position
func (r *Room) UpdatePlayerPosition(playerID string, x, y int) {
	r.mu.Lock()
	defer r.mu.Unlock()

}

// RoomManager manages all game rooms
type RoomManager struct {
	rooms       map[string]*Room
	chatManager *ChatManager
	mu          sync.RWMutex
}

// NewRoomManager creates a new room manager
func NewRoomManager(chatManager *ChatManager) *RoomManager {
	return &RoomManager{
		rooms:       make(map[string]*Room),
		chatManager: chatManager,
	}
}

// GetOrCreateRoom gets an existing room or creates a new one
func (rm *RoomManager) GetOrCreateRoom(roomID string) *Room {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if room, ok := rm.rooms[roomID]; ok {
		return room
	}

	// Create new room
	if roomID == "" {
		roomID = uuid.New().String()
	}

	room := NewRoom(roomID, rm.chatManager)
	rm.rooms[roomID] = room

	go room.Run()

	log.Printf("Created new room: %s", roomID)
	return room
}

// GetRoom gets an existing room
func (rm *RoomManager) GetRoom(roomID string) *Room {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.rooms[roomID]
}
