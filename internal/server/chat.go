package server

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/always-at-morg/internal/protocol"
)

// ChatMessage represents a stored chat message
type ChatMessage struct {
	ID           string
	FromPlayerID string
	ToPlayerID   string // Empty for global chat
	Message      string
	Timestamp    int64
	Type         string // "global", "dm", "announcement"
}

// ChatManager manages all chat functionality
type ChatManager struct {
	// Message storage
	globalMessages []ChatMessage            // Global chat history
	dmMessages     map[string][]ChatMessage // key: "playerID1:playerID2" (sorted) -> messages
	roomMessages   map[string][]ChatMessage // key: room number -> messages
	announcements  []ChatMessage            // Announcement history
	mu             sync.RWMutex
}

// NewChatManager creates a new chat manager
func NewChatManager() *ChatManager {
	return &ChatManager{
		globalMessages: make([]ChatMessage, 0),
		dmMessages:     make(map[string][]ChatMessage),
		roomMessages:   make(map[string][]ChatMessage),
		announcements:  make([]ChatMessage, 0),
	}
}

// HandleGlobalChat stores a new global chat message
func (cm *ChatManager) HandleGlobalChat(client *Client, message string, room *Room) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Store the new message
	chatMsg := ChatMessage{
		ID:           uuid.New().String(),
		FromPlayerID: client.ID,
		Message:      message,
		Timestamp:    time.Now().Unix(),
		Type:         "global",
	}
	cm.globalMessages = append(cm.globalMessages, chatMsg)

	// Build payload with all global chat messages
	messages := make([]protocol.GlobalChatPayload, len(cm.globalMessages))
	for i, msg := range cm.globalMessages {
		// Get username from client ID (need to look up from room)
		username := ""
		room.mu.RLock()
		if c, ok := room.Clients[msg.FromPlayerID]; ok {
			username = c.Username
		}
		room.mu.RUnlock()

		messages[i] = protocol.GlobalChatPayload{
			Username:  username,
			Message:   msg.Message,
			Timestamp: msg.Timestamp,
		}
	}

	// Broadcast ALL messages to all clients
	payload := protocol.GlobalChatMessagesPayload{
		Messages: messages,
	}

	msg, err := protocol.EncodeMessage(protocol.MsgGlobalChatMessages, payload)
	if err != nil {
		return
	}

	room.broadcast <- msg
}

// HandleAnnouncement stores a new announcement
func (cm *ChatManager) HandleAnnouncement(message string, room *Room) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Store the announcement
	chatMsg := ChatMessage{
		ID:        uuid.New().String(),
		Message:   message,
		Timestamp: time.Now().Unix(),
		Type:      "announcement",
	}
	cm.announcements = append(cm.announcements, chatMsg)
}

// HandleDirectMessage sends a 1:1 message between two players
// toUsername is the target player's username (not client ID)
func (cm *ChatManager) HandleDirectMessage(fromClient *Client, toUsername string, message string, room *Room) {
	// Find target client by username
	room.mu.RLock()
	var targetClient *Client
	for _, client := range room.Clients {
		if client.Username == toUsername {
			targetClient = client
			break
		}
	}
	room.mu.RUnlock()

	if targetClient == nil {
		// Target player not found in room
		return
	}

	cm.mu.Lock()
	// Store the DM
	chatMsg := ChatMessage{
		ID:           uuid.New().String(),
		FromPlayerID: fromClient.ID,
		ToPlayerID:   targetClient.ID,
		Message:      message,
		Timestamp:    time.Now().Unix(),
		Type:         "dm",
	}

	// Get or create DM history key (sorted player IDs for consistent key)
	dmKey := getDMKey(fromClient.ID, targetClient.ID)
	cm.dmMessages[dmKey] = append(cm.dmMessages[dmKey], chatMsg)
	cm.mu.Unlock()

	// Send usernames in the payload (not IDs) so client can display them
	payload := protocol.ChatMessagePayload{
		FromPlayerID: fromClient.Username,
		ToPlayerID:   targetClient.Username,
		Message:      message,
		Timestamp:    chatMsg.Timestamp,
	}

	msg, err := protocol.EncodeMessage(protocol.MsgChatMessage, payload)
	if err != nil {
		return
	}

	// Send message to both sender and receiver
	targetClient.send <- msg
	fromClient.send <- msg
}

// GetGlobalMessages returns all global chat messages as GlobalChatPayload format
func (cm *ChatManager) GetGlobalMessages(room *Room) protocol.GlobalChatMessagesPayload {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Build payload with all global chat messages
	messages := make([]protocol.GlobalChatPayload, len(cm.globalMessages))
	for i, msg := range cm.globalMessages {
		// Get username from client ID (need to look up from room)
		username := ""
		room.mu.RLock()
		if c, ok := room.Clients[msg.FromPlayerID]; ok {
			username = c.Username
		}
		room.mu.RUnlock()

		messages[i] = protocol.GlobalChatPayload{
			Username:  username,
			Message:   msg.Message,
			Timestamp: msg.Timestamp,
		}
	}

	return protocol.GlobalChatMessagesPayload{
		Messages: messages,
	}
}

// GetDMMessages returns all DM messages between two players
func (cm *ChatManager) GetDMMessages(playerID1, playerID2 string) []ChatMessage {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	dmKey := getDMKey(playerID1, playerID2)
	messages := cm.dmMessages[dmKey]

	// Return a copy
	result := make([]ChatMessage, len(messages))
	copy(result, messages)
	return result
}

// GetAnnouncements returns all announcements
func (cm *ChatManager) GetAnnouncements() []ChatMessage {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Return a copy
	messages := make([]ChatMessage, len(cm.announcements))
	copy(messages, cm.announcements)
	return messages
}

// HandleRoomChat stores and broadcasts a room chat message
func (cm *ChatManager) HandleRoomChat(client *Client, roomNumber string, message string, room *Room) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Store the new message
	chatMsg := ChatMessage{
		ID:           uuid.New().String(),
		FromPlayerID: client.ID,
		ToPlayerID:   roomNumber, // Store room number in ToPlayerID field
		Message:      message,
		Timestamp:    time.Now().Unix(),
		Type:         "room",
	}

	// Initialize room message array if it doesn't exist
	if cm.roomMessages[roomNumber] == nil {
		cm.roomMessages[roomNumber] = make([]ChatMessage, 0)
	}
	cm.roomMessages[roomNumber] = append(cm.roomMessages[roomNumber], chatMsg)

	// Build payload with all room chat messages for this room
	messages := make([]protocol.RoomChatPayload, len(cm.roomMessages[roomNumber]))
	for i, msg := range cm.roomMessages[roomNumber] {
		// Get username from client ID
		username := ""
		room.mu.RLock()
		if c, ok := room.Clients[msg.FromPlayerID]; ok {
			username = c.Name
		}
		room.mu.RUnlock()

		messages[i] = protocol.RoomChatPayload{
			RoomNumber: roomNumber,
			Username:   username,
			Message:    msg.Message,
			Timestamp:  msg.Timestamp,
		}
	}

	payload := protocol.RoomChatMessagesPayload{
		RoomNumber: roomNumber,
		Messages:   messages,
	}

	// Broadcast to all clients in the same room
	room.mu.RLock()
	for _, c := range room.Clients {
		// Only send to clients in the same room
		if c.CurrentRoomNumber == roomNumber {
			msg, _ := protocol.EncodeMessage(protocol.MsgRoomChatMessages, payload)
			c.send <- msg
		}
	}
	room.mu.RUnlock()
}

// GetRoomMessages returns all chat messages for a specific room
func (cm *ChatManager) GetRoomMessages(roomNumber string, room *Room) protocol.RoomChatMessagesPayload {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Get messages for this room
	roomMsgs := cm.roomMessages[roomNumber]
	if roomMsgs == nil {
		return protocol.RoomChatMessagesPayload{
			RoomNumber: roomNumber,
			Messages:   []protocol.RoomChatPayload{},
		}
	}

	// Convert to protocol format
	messages := make([]protocol.RoomChatPayload, len(roomMsgs))
	for i, msg := range roomMsgs {
		// Get username from client ID
		username := ""
		room.mu.RLock()
		if c, ok := room.Clients[msg.FromPlayerID]; ok {
			username = c.Name
		}
		room.mu.RUnlock()

		messages[i] = protocol.RoomChatPayload{
			RoomNumber: roomNumber,
			Username:   username,
			Message:    msg.Message,
			Timestamp:  msg.Timestamp,
		}
	}

	return protocol.RoomChatMessagesPayload{
		RoomNumber: roomNumber,
		Messages:   messages,
	}
}

// GetAllRoomMessages returns all room chat messages for all rooms
func (cm *ChatManager) GetAllRoomMessages(room *Room) map[string][]protocol.RoomChatPayload {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	result := make(map[string][]protocol.RoomChatPayload)

	for roomNumber, roomMsgs := range cm.roomMessages {
		messages := make([]protocol.RoomChatPayload, len(roomMsgs))
		for i, msg := range roomMsgs {
			// Get username from client ID
			username := ""
			room.mu.RLock()
			if c, ok := room.Clients[msg.FromPlayerID]; ok {
				username = c.Name
			}
			room.mu.RUnlock()

			messages[i] = protocol.RoomChatPayload{
				RoomNumber: roomNumber,
				Username:   username,
				Message:    msg.Message,
				Timestamp:  msg.Timestamp,
			}
		}
		result[roomNumber] = messages
	}

	return result
}

// Helper function to generate consistent DM keys
func getDMKey(playerID1, playerID2 string) string {
	if playerID1 < playerID2 {
		return playerID1 + ":" + playerID2
	}
	return playerID2 + ":" + playerID1
}
