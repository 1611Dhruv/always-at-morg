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
	ID       string
	Clients  map[string]*Client
	GameState *protocol.GameState

	mu          sync.RWMutex
	broadcast   chan []byte
	register    chan *Client
	unregister  chan *Client
	tickRate    time.Duration
}

// NewRoom creates a new game room
func NewRoom(id string) *Room {
	return &Room{
		ID:       id,
		Clients:  make(map[string]*Client),
		GameState: &protocol.GameState{
			Players:  make(map[string]protocol.Player),
			Entities: []protocol.Entity{},
			Tick:     0,
		},
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
			r.update()
		}
	}
}

func (r *Room) handleRegister(client *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Clients[client.ID] = client

	// Create player
	player := protocol.Player{
		ID:    client.ID,
		Name:  client.Name,
		X:     10 + len(r.GameState.Players)*5,
		Y:     10,
		Color: r.getRandomColor(),
		Score: 0,
	}
	r.GameState.Players[client.ID] = player

	log.Printf("Player %s joined room %s", client.Name, r.ID)

	// Send room joined message to the new client
	msg, _ := protocol.EncodeMessage(protocol.MsgRoomJoined, protocol.RoomJoinedPayload{
		RoomID:    r.ID,
		PlayerID:  client.ID,
		GameState: r.GameState,
	})
	client.send <- msg

	// Broadcast player joined to others
	broadcastMsg, _ := protocol.EncodeMessage(protocol.MsgPlayerJoined, protocol.PlayerJoinedPayload{
		Player: player,
	})
	r.broadcastToOthers(client.ID, broadcastMsg)
}

func (r *Room) handleUnregister(client *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.Clients[client.ID]; ok {
		delete(r.Clients, client.ID)
		delete(r.GameState.Players, client.ID)
		close(client.send)

		log.Printf("Player %s left room %s", client.Name, r.ID)

		// Broadcast player left
		msg, _ := protocol.EncodeMessage(protocol.MsgPlayerLeft, protocol.PlayerLeftPayload{
			PlayerID: client.ID,
		})
		r.broadcast <- msg
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

func (r *Room) broadcastToOthers(excludeID string, message []byte) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for id, client := range r.Clients {
		if id != excludeID {
			select {
			case client.send <- message:
			default:
				close(client.send)
				delete(r.Clients, id)
			}
		}
	}
}

// update runs the game logic and broadcasts state
func (r *Room) update() {
	r.mu.Lock()
	r.GameState.Tick++

	// Add game logic here (e.g., entity movement, collision detection)

	r.mu.Unlock()

	// Broadcast game state to all clients
	msg, _ := protocol.EncodeMessage(protocol.MsgGameState, r.GameState)
	r.broadcast <- msg
}

// UpdatePlayerPosition updates a player's position
func (r *Room) UpdatePlayerPosition(playerID string, x, y int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if player, ok := r.GameState.Players[playerID]; ok {
		player.X = x
		player.Y = y
		r.GameState.Players[playerID] = player
	}
}

func (r *Room) getRandomColor() string {
	colors := []string{"red", "green", "blue", "yellow", "magenta", "cyan"}
	return colors[len(r.GameState.Players)%len(colors)]
}

// RoomManager manages all game rooms
type RoomManager struct {
	rooms map[string]*Room
	mu    sync.RWMutex
}

// NewRoomManager creates a new room manager
func NewRoomManager() *RoomManager {
	return &RoomManager{
		rooms: make(map[string]*Room),
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

	room := NewRoom(roomID)
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
