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
	globalMessages []ChatMessage              // Global chat history
	dmMessages     map[string][]ChatMessage   // key: "playerID1:playerID2" (sorted) -> messages
	announcements  []ChatMessage              // Announcement history
	activeDMs      map[string]map[string]bool // fromClientID -> toClientID -> isActive
	mu             sync.RWMutex
}

// NewChatManager creates a new chat manager
func NewChatManager() *ChatManager {
	return &ChatManager{
		globalMessages: make([]ChatMessage, 0),
		dmMessages:     make(map[string][]ChatMessage),
		announcements:  make([]ChatMessage, 0),
		activeDMs:      make(map[string]map[string]bool),
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

// HandleChatRequest processes a chat request from one player to another
func (cm *ChatManager) HandleChatRequest(fromClient *Client, toPlayerID string, message string, room *Room) {
	cm.mu.Lock()

	// Initialize the map if it doesn't exist
	if cm.activeDMs[fromClient.ID] == nil {
		cm.activeDMs[fromClient.ID] = make(map[string]bool)
	}

	// Mark as pending (will be activated on response)
	cm.activeDMs[fromClient.ID][toPlayerID] = false
	cm.mu.Unlock()

	payload := protocol.ChatReqestPayload{
		FromPlayerID: fromClient.ID,
		ToPlayerID:   toPlayerID,
		Message:      message,
	}

	msg, err := protocol.EncodeMessage(protocol.MsgChatRequest, payload)
	if err != nil {
		return
	}

	// Send chat request to the target player
	room.mu.RLock()
	defer room.mu.RUnlock()

	if targetClient, ok := room.Clients[toPlayerID]; ok {
		targetClient.send <- msg
	}
}

// HandleChatResponse processes accept/decline of a chat request
func (cm *ChatManager) HandleChatResponse(fromPlayerID string, toPlayerID string, accepted bool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if accepted {
		// Activate the DM session
		if cm.activeDMs[fromPlayerID] == nil {
			cm.activeDMs[fromPlayerID] = make(map[string]bool)
		}
		if cm.activeDMs[toPlayerID] == nil {
			cm.activeDMs[toPlayerID] = make(map[string]bool)
		}

		cm.activeDMs[fromPlayerID][toPlayerID] = true
		cm.activeDMs[toPlayerID][fromPlayerID] = true
	} else {
		// Remove the pending request
		if cm.activeDMs[fromPlayerID] != nil {
			delete(cm.activeDMs[fromPlayerID], toPlayerID)
		}
	}
}

// HandleDirectMessage sends a 1:1 message between two players
func (cm *ChatManager) HandleDirectMessage(fromClient *Client, toPlayerID string, message string, room *Room) {
	cm.mu.Lock()

	// Check if DM session is active
	if cm.activeDMs[fromClient.ID] == nil || !cm.activeDMs[fromClient.ID][toPlayerID] {
		cm.mu.Unlock()
		// Send error - no active DM session
		errMsg, _ := protocol.EncodeMessage(protocol.MsgError, protocol.ErrorPayload{
			Message: "No active chat session with this player",
		})
		fromClient.send <- errMsg
		return
	}

	// Store the DM
	chatMsg := ChatMessage{
		ID:           uuid.New().String(),
		FromPlayerID: fromClient.ID,
		ToPlayerID:   toPlayerID,
		Message:      message,
		Timestamp:    time.Now().Unix(),
		Type:         "dm",
	}

	// Get or create DM history key (sorted player IDs for consistent key)
	dmKey := getDMKey(fromClient.ID, toPlayerID)
	cm.dmMessages[dmKey] = append(cm.dmMessages[dmKey], chatMsg)
	cm.mu.Unlock()

	payload := protocol.ChatMessagePayload{
		FromPlayerID: fromClient.ID,
		ToPlayerID:   toPlayerID,
		Message:      message,
		Timestamp:    chatMsg.Timestamp,
	}

	msg, err := protocol.EncodeMessage(protocol.MsgChatMessage, payload)
	if err != nil {
		return
	}

	// Send message to the target player
	room.mu.RLock()
	defer room.mu.RUnlock()

	if targetClient, ok := room.Clients[toPlayerID]; ok {
		targetClient.send <- msg
	}
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

// IsActiveDM checks if there's an active DM session between two players
func (cm *ChatManager) IsActiveDM(fromPlayerID, toPlayerID string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.activeDMs[fromPlayerID] == nil {
		return false
	}

	return cm.activeDMs[fromPlayerID][toPlayerID]
}

// CloseDM closes a DM session between two players
func (cm *ChatManager) CloseDM(fromPlayerID, toPlayerID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.activeDMs[fromPlayerID] != nil {
		delete(cm.activeDMs[fromPlayerID], toPlayerID)
	}
	if cm.activeDMs[toPlayerID] != nil {
		delete(cm.activeDMs[toPlayerID], fromPlayerID)
	}
}

// Helper function to generate consistent DM keys
func getDMKey(playerID1, playerID2 string) string {
	if playerID1 < playerID2 {
		return playerID1 + ":" + playerID2
	}
	return playerID2 + ":" + playerID1
}
