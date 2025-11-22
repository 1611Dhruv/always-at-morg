package protocol

import "encoding/json"

// MessageType defines the type of WebSocket message
type MessageType string

const (
	// Client -> Server
	MsgJoinRoom   MessageType = "join_room"
	MsgLeaveRoom  MessageType = "leave_room"
	MsgPlayerMove MessageType = "player_move"
	MsgPlayerInput MessageType = "player_input"

	// Server -> Client
	MsgRoomJoined  MessageType = "room_joined"
	MsgRoomLeft    MessageType = "room_left"
	MsgGameState   MessageType = "game_state"
	MsgPlayerJoined MessageType = "player_joined"
	MsgPlayerLeft   MessageType = "player_left"
	MsgError       MessageType = "error"
)

// Message is the wrapper for all WebSocket messages
type Message struct {
	Type    MessageType     `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// JoinRoomPayload is sent when a player wants to join a room
type JoinRoomPayload struct {
	RoomID     string `json:"room_id"`
	PlayerName string `json:"player_name"`
}

// RoomJoinedPayload is sent when a player successfully joins a room
type RoomJoinedPayload struct {
	RoomID   string      `json:"room_id"`
	PlayerID string      `json:"player_id"`
	GameState *GameState `json:"game_state"`
}

// PlayerJoinedPayload is broadcast when a new player joins
type PlayerJoinedPayload struct {
	Player Player `json:"player"`
}

// PlayerLeftPayload is broadcast when a player leaves
type PlayerLeftPayload struct {
	PlayerID string `json:"player_id"`
}

// PlayerMovePayload contains player movement data
type PlayerMovePayload struct {
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Direction string `json:"direction"`
}

// PlayerInputPayload contains general player input
type PlayerInputPayload struct {
	Action string                 `json:"action"`
	Data   map[string]interface{} `json:"data,omitempty"`
}

// GameState represents the current state of the game
type GameState struct {
	Players map[string]Player `json:"players"`
	Entities []Entity         `json:"entities,omitempty"`
	Tick    int64            `json:"tick"`
}

// Player represents a player in the game
type Player struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	Color    string `json:"color"`
	Score    int    `json:"score"`
}

// Entity represents a game entity (e.g., collectibles, obstacles)
type Entity struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	X    int    `json:"x"`
	Y    int    `json:"y"`
}

// ErrorPayload contains error information
type ErrorPayload struct {
	Message string `json:"message"`
}

// EncodeMessage encodes a message with its payload
func EncodeMessage(msgType MessageType, payload interface{}) ([]byte, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	msg := Message{
		Type:    msgType,
		Payload: payloadBytes,
	}

	return json.Marshal(msg)
}

// DecodeMessage decodes a message
func DecodeMessage(data []byte) (*Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	return &msg, err
}
