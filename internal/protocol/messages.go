package protocol //handles communication protocol between client and server
// WebSocket message types and payloads
import "encoding/json"

// MessageType defines the type of WebSocket message
type MessageType string

const (
	// Client -> Server
	MsgJoinRoom    MessageType = "join_room"
	MsgLeaveRoom   MessageType = "leave_room"
	MsgPlayerMove  MessageType = "player_move"
	MsgPlayerInput MessageType = "player_input"
	MsgOnboard     MessageType = "onboard" //client onboarding message

	MsgChatRequest  MessageType = "chat_request"        // start a one to one?
	MsgChatMessage  MessageType = "chat_message"        // one to one
	MsgGlobalChat   MessageType = "global_chat_message" // me sending u messaeg?
	MsgAnnouncement MessageType = "announcement"

	// Server -> Client
	MsgOnboardRequest     MessageType = "onboard_request" //server requests onboarding for new user
	MsgRoomJoined         MessageType = "room_joined"     //server confirming
	MsgRoomLeft           MessageType = "room_left"
	MsgGameState          MessageType = "game_state"
	MsgPlayerJoined       MessageType = "player_joined"
	MsgPlayerLeft         MessageType = "player_left"
	MsgError              MessageType = "error"
	MsgGlobalChatMessages MessageType = "global_chat_messages"
	MsgKuluchifiedState   MessageType = "kuluchified_state" // Unified per-tick state update

	//chat and interaction
	MsgChatResponse  MessageType = "chat_response"  // accept/decline chat interaction
	MsgNearbyPlayers MessageType = "nearby_players" // take lite

)

// Message is the wrapper for all WebSocket messages
type Message struct {
	Type    MessageType     `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// JoinRoomPayload is sent when a player wants to join a room
type JoinRoomPayload struct {
	Username string `json:"username"` // Always required
	RoomID   string `json:"room_id"`
}

// RoomJoinedPayload is sent when a player successfully joins a room
type RoomJoinedPayload struct {
	RoomID    string     `json:"room_id"`
	PlayerID  string     `json:"player_id"`
	GameState *GameState `json:"game_state"`
}

// type Pos struct {
// 	X int `json:"x"`
// 	Y int `json:"y"`
// }

// GameState represents the current state of the game
type GameState struct {
	Players       map[string]Player `json:"players"`
	PosToUsername map[string]string `json:"pos_to_username"`
	Tick          int64             `json:"tick"`
	Map           [250][400]string     `json:"-"` // Don't send to clients
}

// Player represents a player in the game
type Player struct {
	Username string `json:"username"`
	Pos      string    `json:"pos"`
	Avatar   []int  `json:"avatar"`
}

// PlayerMovePayload is sent when a player wants to move
type PlayerMovePayload struct {
	NewX int `json:"new_x"`
	NewY int `json:"new_y"`
}

// chat request payload for initiating chat interaction
type ChatReqestPayload struct {
	FromPlayerID string `json:"from_player_id"`
	ToPlayerID   string `json:"to_player_id"`
	Message      string `json:"message"`
}

// accept/decline chat interaction
type ChatResponsePayload struct {
	FromPlayerID string `json:"from_player_id"` // Who initiated the request
	ToPlayerID   string `json:"to_player_id"`   // Who is responding
	Accepted     bool   `json:"accepted"`
}

// chat message payload for sending messages between players
type ChatMessagePayload struct {
	FromPlayerID string `json:"from_player_id"`
	ToPlayerID   string `json:"to_player_id"`
	Message      string `json:"message"`
	Timestamp    int64  `json:"timestamp"`
}

// global chat message payload for messages sent to all players
type GlobalChatPayload struct {
	Username  string `json:"username"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

// announcement payload for server-wide messages
type AnnouncementPayload struct {
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

// ErrorPayload contains error information
type ErrorPayload struct {
	Message string `json:"message"`
}

type OnboardPayload struct {
	Name   string `json:"name"`   // Display name
	Avatar []int  `json:"avatar"` // Color for now (username already provided in JoinRoom)
}

type GlobalChatMessagesPayload struct {
	Messages []GlobalChatPayload `json:"messages"`
}

// KuluchifiedStatePayload is the unified per-tick state update containing everything
type KuluchifiedStatePayload struct {
	GameState     GameState             `json:"game_state"`
	ChatMessages  []GlobalChatPayload   `json:"chat_messages"`
	Announcements []AnnouncementPayload `json:"announcements"`
	Players       map[string]Player     `json:"players"`
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
