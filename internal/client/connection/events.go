package connection

import "github.com/yourusername/always-at-morg/internal/protocol"

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

// GameStateEvent is sent when game state is updated
type GameStateEvent struct {
	State *protocol.GameState
}

func (GameStateEvent) isEvent() {}

// PlayerJoinedEvent is sent when a player joins
type PlayerJoinedEvent struct {
	Player protocol.Player
}

func (PlayerJoinedEvent) isEvent() {}

// PlayerLeftEvent is sent when a player leaves
type PlayerLeftEvent struct {
	PlayerID string
}

func (PlayerLeftEvent) isEvent() {}

// ErrorEvent is sent when an error occurs
type ErrorEvent struct {
	Message string
}

func (ErrorEvent) isEvent() {}

// ChatMessageEvent is sent when a chat message is received
type ChatMessageEvent struct {
	Sender  string
	Message string
}

func (ChatMessageEvent) isEvent() {}
