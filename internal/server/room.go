package server

import (
	"fmt"
	"log" //logs messages
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/always-at-morg/internal/protocol"
)

var startingPositions = []string{
	"10:100",
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
		roomMap = [250][400]string{} // Use empty map as fallback
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

// findRandomSpawnPosition finds a random valid spawn position in the room
// A valid position must have all 9 tiles in the 3x3 area as walkable (' ' or '@')
func (r *Room) findRandomSpawnPosition() (string, error) {
	maxAttempts := 1000
	for i := 0; i < maxAttempts; i++ {
		x := rand.Intn(400)
		y := rand.Intn(250)
		posStr := fmt.Sprintf("%d:%d", y, x) // Format: "Y:X" to match client expectation

		// Check if all 9 tiles in the 3x3 area are walkable (' ' or '@')
		valid := true
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				ny := y + dy
				nx := x + dx

				// Check bounds
				if ny < 0 || ny >= 250 || nx < 0 || nx >= 400 {
					valid = false
					break
				}

				// Get value - must be walkable (' ' space or '@' dark brown floor)
				cellValue := r.GameState.Map[ny][nx]
				if cellValue != " " && cellValue != "@" {
					valid = false
					break
				}
			}
			if !valid {
				break
			}
		}

		if !valid {
			continue
		}

		// Check if position is not occupied
		if _, occupied := r.GameState.PosToUsername[posStr]; occupied {
			continue
		}

		return posStr, nil
	}

	return "", fmt.Errorf("failed to find valid spawn position after %d attempts", maxAttempts)
}

func (r *Room) handleRegister(client *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Find a random valid spawn position
	posStr, err := r.findRandomSpawnPosition()
	if err != nil {
		log.Printf("Error finding spawn position for %s: %v", client.Name, err)
		// Fallback to a default position if we can't find a valid one
		posStr = "52:120"
	}
	client.Pos = posStr

	// Parse position and set CurrentRoomNumber
	var x, y int
	fmt.Sscanf(posStr, "%d:%d", &y, &x)
	client.CurrentRoomNumber = r.getRoomNumberFromPosition(x, y)

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
	roomChatMessages := chatManager.GetAllRoomMessages(r)

	// Build players map (keyed by username for easy client lookup)
	r.mu.RLock()
	players := make(map[string]protocol.Player)
	for _, client := range r.Clients {
		players[client.Username] = protocol.Player{
			Pos:      client.Pos,
			Avatar:   client.Avatar,
			Username: client.Username,
		}
	}
	r.mu.RUnlock()

	// Create unified state payload with current players
	kuluchifiedState := protocol.KuluchifiedStatePayload{
		GameState: protocol.GameState{
			Tick:          r.GameState.Tick,
			Players:       players, // Use the players map we just built!
			PosToUsername: r.GameState.PosToUsername,
		},
		ChatMessages:      chatMessages.Messages,
		RoomChatMessages:  roomChatMessages,
		Announcements:     announcementPayloads,
		Players:           players,
		TreasureHuntState: Manager.GetState(), // Broadcast treasure hunt state to all clients
	}

	// Send ONE broadcast with everything
	msg, _ := protocol.EncodeMessage(protocol.MsgKuluchifiedState, kuluchifiedState)
	r.broadcast <- msg
}

// isWalkable checks if a position is walkable according to the room map
func (r *Room) isWalkable(x, y int) bool {
	// Check bounds
	if y < 0 || y >= 250 || x < 0 || x >= 400 {
		return false
	}

	// Get room map value
	value := r.GameState.Map[y][x]

	// Wall characters ("r", "o", "i") are not walkable
	// "e" (entrances), "-1" (hallways), "@" (dark brown), and room numbers ("1", "2", "3", ...) are walkable
	// "c" (couch) is not walkable (furniture)
	return value != "r" && value != "o" && value != "i" && value != "c"
}

// canAvatarFitAt checks if a 3x3 avatar can fit at the given position
// The avatar occupies a 3x3 grid centered on (x, y)
func (r *Room) canAvatarFitAt(x, y int) bool {
	// Check all 9 tiles in the 3x3 footprint - must be walkable
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			checkX := x + dx
			checkY := y + dy

			// Check bounds
			if checkY < 0 || checkY >= 250 || checkX < 0 || checkX >= 400 {
				return false // Out of bounds
			}

			// Check if tile is walkable: ' ' (hallway), 'e' (entrance), "-1" (outside), '@' (dark brown floor), or room numbers ("1", "2", etc.)
			value := r.GameState.Map[checkY][checkX]
			if value == " " || value == "e" || value == "-1" || value == "@" {
				// Explicitly walkable
				continue
			}
			// Check if it's a room number (numeric string)
			if _, err := strconv.Atoi(value); err == nil {
				// It's a room number - walkable
				continue
			}
			// Not walkable (walls, inaccessible areas, furniture T/t/W, etc.)
			return false
		}
	}

	return true // All tiles in 3x3 grid are walkable
}

// UpdatePlayerPosition updates a player's position
func (r *Room) UpdatePlayerPosition(username string, x, y int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Validate that the 3x3 avatar footprint fits at the new position
	if !r.canAvatarFitAt(x, y) {
		// Avatar would collide with wall or go out of bounds, reject movement
		return
	}

	// Check if position is already occupied by another player
	newPos := fmt.Sprintf("%d:%d", y, x) // Format: "Y:X"
	if existingUser, occupied := r.GameState.PosToUsername[newPos]; occupied && existingUser != username {
		// Position is occupied by another player, reject movement
		return
	}

	// Find the client by username
	for clientID, client := range r.Clients {
		if client.Username == username {
			// Update old position in PosToUsername map (remove)
			oldPos := client.Pos
			if oldPos != "" {
				delete(r.GameState.PosToUsername, oldPos)
			}

			// Update client position
			client.Pos = newPos

			// Update current room number based on new position
			client.CurrentRoomNumber = r.getRoomNumberFromPosition(x, y)

			// Update the client in the map (important - we were modifying a copy!)
			r.Clients[clientID] = client

			// Update new position in PosToUsername map
			r.GameState.PosToUsername[newPos] = username

			// Update GameState.Players directly so client sees the change on next state update
			if player, exists := r.GameState.Players[username]; exists {
				player.Pos = newPos
				r.GameState.Players[username] = player
			}

			return
		}
	}
}

// getRoomNumberFromPosition determines which room a position is in
// Returns room number as string ("1", "2", etc.) or "" if in hallway
func (r *Room) getRoomNumberFromPosition(x, y int) string {
	// Check bounds
	if y < 0 || y >= 250 || x < 0 || x >= 400 {
		return ""
	}

	// Get the map value at this position
	value := r.GameState.Map[y][x]

	// Check if it's a room number (numeric string)
	if _, err := strconv.Atoi(value); err == nil {
		return value
	}

	return ""
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
