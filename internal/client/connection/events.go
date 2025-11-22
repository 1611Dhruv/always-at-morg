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

// Chat Message event:
type ChatMessageEvent struct{}

func (ChatMessageEvent) isEvent() {}
