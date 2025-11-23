package connection

// Event represents events from the connection manager
type Event interface {
	isEvent()
}

// ConnectedEvent is sent when connection is established
type ConnectedEvent struct{}

func (ConnectedEvent) isEvent() {}

// DisconnectedEvent is sent when connection is lost
type DisconnectedEvent struct {
	Error error
}

func (DisconnectedEvent) isEvent() {}

// ErrorEvent is sent when an error occurs
type ErrorEvent struct {
	Message string
}

func (ErrorEvent) isEvent() {}

// The onboarding event
type OnboardRequestEvent struct{}

func (OnboardRequestEvent) isEvent() {}

// Game State event:
type GameStateEvent struct{}

func (GameStateEvent) isEvent() {}

// Global chat messages event
type GlobalChatMessagesEvent struct {
	Messages []ChatMessage
}

func (GlobalChatMessagesEvent) isEvent() {}

// ChatMessage represents a single chat message
type ChatMessage struct {
	Username  string
	Message   string
	Timestamp int64
}

// PrivateChatMessageEvent is sent when a private message is received
type PrivateChatMessageEvent struct {
	FromUsername string
	ToUsername   string
	Message      string
	Timestamp    int64
}

func (PrivateChatMessageEvent) isEvent() {}
