package ui

// This file will contain chat panel rendering logic
// TODO: Implement chat message rendering, input handling, etc.

// ChatMessage represents a chat message
type ChatMessage struct {
	Sender  string
	Content string
	IsOwn   bool
}

// ChatPanel manages chat state
type ChatPanel struct {
	messages []ChatMessage
	input    string
}

// NewChatPanel creates a new chat panel
func NewChatPanel() *ChatPanel {
	return &ChatPanel{
		messages: []ChatMessage{},
		input:    "",
	}
}

// AddMessage adds a message to the chat
func (c *ChatPanel) AddMessage(sender, content string, isOwn bool) {
	c.messages = append(c.messages, ChatMessage{
		Sender:  sender,
		Content: content,
		IsOwn:   isOwn,
	})

	// Keep only last 50 messages
	if len(c.messages) > 50 {
		c.messages = c.messages[len(c.messages)-50:]
	}
}

// GetMessages returns all messages
func (c *ChatPanel) GetMessages() []ChatMessage {
	return c.messages
}
