package server

import (
	"log" //logs messages
	"sync"
	"time"


	"github.com/google/uuid"
	"github.com/yourusername/always-at-morg/internal/protocol"
)

var startingPositions = []string{
	"52:200",
	"18:150",
	"18:200",
	"23:100",
}


// Room represents a game room/session
type Room struct {
	ID          string
	Clients     map[string]*Client
	GameState   *protocol.GameState
	chatManager *ChatManager

	mu        sync.RWMutex
	broadcast chan []byte  //this is private to room only, used to send messages to all clients in the room
	register  chan *Client //clients register to room, used when a new client joins

	unregister chan *Client
	tickRate   time.Duration
}

// NewRoom creates a new game room
func NewRoom(id string, chatManager *ChatManager) *Room {
	roomMap, err := fillRoomMap()
	if err != nil {
		log.Printf("Warning: failed to load room map: %v", err)
		roomMap = [250][400]int{} // Use empty map as fallback
	}

	return &Room{
		ID:      id,
		Clients: make(map[string]*Client),
		GameState: &protocol.GameState{
			Tick:          0,
			Players:       make(map[string]protocol.Player),
			PosToUsername: make(map[string]string),
			Map:           roomMap,
		},
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

	// Assign starting position based on current number of clients (cycle through positions)
	posStr := startingPositions[len(r.Clients)%len(startingPositions)]
	client.Pos = posStr

	r.Clients[client.ID] = client

	// Update GameState.Players map
	r.GameState.Players[client.Username] = protocol.Player{
		Username: client.Username,
		Pos:      posStr,
		Avatar:   client.Avatar,
	}

	// Update GameState.PosToUsername map to track occupied positions
	r.GameState.PosToUsername[posStr] = client.Username

	log.Printf("Player %s joined room %s at position %s", client.Name, r.ID, client.Pos)

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

	// Build unified Kuluchified state containing everything
	announcements := chatManager.GetAnnouncements()
	announcementPayloads := make([]protocol.AnnouncementPayload, len(announcements))
	for i, announcement := range announcements {
		announcementPayloads[i] = protocol.AnnouncementPayload{
			Message:   announcement.Message,
			Timestamp: announcement.Timestamp,
		}
	}

	chatMessages := chatManager.GetGlobalMessages(r)

	// Build players map
	r.mu.RLock()
	players := make(map[string]protocol.Player)
	for id, client := range r.Clients {
		players[id] = protocol.Player{
			Pos:      client.Pos,
			Avatar:   client.Avatar,
			Username: client.Username,
			// Add position and other player data here when available
		}
	}
	r.mu.RUnlock()

	// Create unified state payload
	kuluchifiedState := protocol.KuluchifiedStatePayload{
		GameState:     *r.GameState,
		ChatMessages:  chatMessages.Messages,
		Announcements: announcementPayloads,
		Players:       players,
	}

	// Send ONE broadcast with everything
	msg, _ := protocol.EncodeMessage(protocol.MsgKuluchifiedState, kuluchifiedState)
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
