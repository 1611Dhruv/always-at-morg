# Always At Morg - Multiplayer Terminal Game

A real-time multiplayer game built with Go, featuring WebSocket communication between a backend server and terminal-based clients using Bubble Tea and Termloop.

## Architecture

### Backend (Server)
- WebSocket server using `gorilla/websocket`
- Room-based game sessions with concurrent player support
- Real-time game state synchronization
- Server-side game loop running at 20 ticks per second

### Frontend (Client)
- **Bubble Tea**: Menu navigation and lobby management
- **Termloop**: Real-time game rendering and controls
- WebSocket client for server communication
- State synchronization system

### Protocol
- JSON-based message protocol
- Message types for joining rooms, player movements, game state updates
- Shared data structures between client and server

## Project Structure

```
.
├── cmd/
│   ├── server/          # Server entry point
│   │   └── main.go
│   └── client/          # Client entry point
│       └── main.go
├── internal/
│   ├── server/          # Server implementation
│   │   ├── websocket.go # WebSocket handling
│   │   └── room.go      # Room & game state management
│   ├── client/          # Client implementation
│   │   ├── websocket.go # WebSocket client
│   │   ├── bubbletea.go # Bubble Tea UI
│   │   └── termloop.go  # Termloop game rendering
│   └── protocol/        # Shared protocol
│       └── messages.go  # Message types & encoding
├── go.mod
└── README.md
```

## Getting Started

### Prerequisites

- Go 1.21 or later

### Installation

1. Clone the repository:
```bash
cd always-at-morg
```

2. Download dependencies:
```bash
go mod download
```

### Running the Server

Start the WebSocket server:

```bash
go run cmd/server/main.go
```

The server will start on `localhost:8080` by default. You can specify a different address:

```bash
go run cmd/server/main.go -addr :9000
```

### Running the Client

#### Menu Mode (Default)

Start with the interactive menu:

```bash
go run cmd/client/main.go
```

Use arrow keys to navigate:
- Join Room
- Create Room
- Quit

#### Direct Connection Modes

Join a specific room directly:

```bash
go run cmd/client/main.go -mode lobby -room my-room -name Player1
```

Start game directly with Termloop:

```bash
go run cmd/client/main.go -mode game -room my-room -name Player1 -termloop
```

Start game with Bubble Tea renderer:

```bash
go run cmd/client/main.go -mode game -room my-room -name Player1
```

### Client Flags

- `-server`: WebSocket server URL (default: `ws://localhost:8080/ws`)
- `-mode`: Start mode - `menu`, `lobby`, or `game` (default: `menu`)
- `-room`: Room ID to join (default: `default-room`)
- `-name`: Player name (default: `Player1`)
- `-termloop`: Use termloop for game rendering (default: false)

## Game Controls

### Bubble Tea Mode

**Menu:**
- Arrow keys / j,k: Navigate
- Enter: Select
- q: Quit

**Lobby:**
- g: Start game
- b: Back to menu
- q: Quit

**Game:**
- WASD / Arrow keys: Move
- b: Back to lobby
- q: Quit

### Termloop Mode

- Arrow keys: Move player
- ESC: Exit game

## Development

### Building

Build the server:
```bash
go build -o bin/server cmd/server/main.go
```

Build the client:
```bash
go build -o bin/client cmd/client/main.go
```

### Running Multiple Clients

You can run multiple clients to test multiplayer functionality:

Terminal 1 (Server):
```bash
go run cmd/server/main.go
```

Terminal 2 (Client 1):
```bash
go run cmd/client/main.go -mode game -room test-room -name Alice
```

Terminal 3 (Client 2):
```bash
go run cmd/client/main.go -mode game -room test-room -name Bob
```

## Protocol Messages

### Client to Server

- `join_room`: Join a game room
- `leave_room`: Leave current room
- `player_move`: Send player movement
- `player_input`: Send custom input action

### Server to Client

- `room_joined`: Confirmation of room join with initial state
- `game_state`: Periodic game state updates
- `player_joined`: Notification when another player joins
- `player_left`: Notification when a player leaves
- `error`: Error messages

## Extending the Game

### Adding Game Logic

Edit `internal/server/room.go` in the `update()` method to add:
- Collision detection
- Collectible items
- Scoring logic
- Win conditions

### Adding Entities

Add new entity types in `internal/protocol/messages.go`:

```go
type Entity struct {
    ID   string `json:"id"`
    Type string `json:"type"` // e.g., "coin", "obstacle"
    X    int    `json:"x"`
    Y    int    `json:"y"`
}
```

Update the rendering in `internal/client/termloop.go` to display new entities.

### Custom Input Actions

Use the `player_input` message type for custom actions:

```go
client.SendInput("jump", map[string]interface{}{
    "height": 5,
})
```

Handle in `internal/server/websocket.go` in the `handleMessage()` method.

## Dependencies

- [gorilla/websocket](https://github.com/gorilla/websocket) - WebSocket implementation
- [charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea) - Terminal UI framework
- [JoelOtter/termloop](https://github.com/JoelOtter/termloop) - Terminal game engine
- [google/uuid](https://github.com/google/uuid) - UUID generation

## License

MIT

## Contributing

Feel free to submit issues and pull requests!
