# Client Architecture

This directory contains the client-side code for Always At Morg, organized into clean, modular components.

## 📁 Directory Structure

```
internal/client/
├── connection/          # Connection management (WebSocket)
│   ├── manager.go      # WebSocket connection manager
│   ├── state.go        # Game state synchronization
│   └── events.go       # Event types (Connected, Disconnected, etc.)
│
├── ui/                  # Bubble Tea UI (menus, chat, overlays)
│   ├── model.go        # Main Bubble Tea model & routing
│   ├── styles.go       # Lipgloss styles & color palette
│   ├── messages.go     # Bubble Tea messages & commands
│   ├── screen_loading.go   # Loading/connection screen
│   ├── screen_username.go  # Username entry screen
│   ├── screen_avatar.go    # Avatar customization screen
│   ├── screen_main.go      # Main game container (70% game + 30% chat)
│   ├── chat_panel.go       # Chat panel rendering
│   └── avatar.go           # Avatar types & presets
│
├── game/                # Game logic (NOT rendering)
│   └── (placeholder - for game state, physics, etc.)
│
├── renderer/            # Game rendering helpers
│   └── (placeholder - for ASCII art rendering)
│
└── (legacy files)
    ├── bubbletea.go    # ⚠️ DEPRECATED - use ui/ instead
    ├── websocket.go    # ⚠️ DEPRECATED - use connection/ instead
    └── termloop.go     # Will be refactored into game/renderer
```

## 🔄 Data Flow

```
User Input → UI Model → Connection Manager → Server
              ↑              ↓
              └── Events ────┘
                     ↓
               Update UI
```

## 🎨 UI Screens

### Screen Flow
```
ViewLoading (connecting...)
    ↓ [success]
ViewUsernameEntry
    ↓ [ENTER]
ViewAvatarCustomization
    ↓ [ENTER]
ViewMainGame (70% game + 30% chat)
```

### Screen Responsibilities

- **screen_loading.go**: Connection screen with animated spinner
- **screen_username.go**: Username input with validation
- **screen_avatar.go**: 3x3 avatar customization (head/torso/legs)
- **screen_main.go**: Split-screen game + chat layout

## 🔌 Connection Manager

The `connection.Manager` handles all WebSocket communication:

```go
// Create manager
mgr := connection.NewManager("ws://localhost:8080/ws")

// Set event callback
mgr.OnEvent(func(event connection.Event) {
    // Handle events: ConnectedEvent, GameStateEvent, etc.
})

// Connect
mgr.Connect()

// Send messages
mgr.JoinRoom("room-id", "player-name")
mgr.SendMove(x, y, "direction")
```

**Events:**
- `ConnectedEvent` - Connection established
- `DisconnectedEvent` - Connection lost
- `GameStateEvent` - Game state updated
- `PlayerJoinedEvent` - Player joined room
- `PlayerLeftEvent` - Player left room
- `ErrorEvent` - Error occurred
- `ChatMessageEvent` - Chat message received

## 🎮 Integrating Game Rendering

**For your teammate's termloop code:**

1. **Move game logic** to `game/` (player entities, physics, etc.)
2. **Move rendering** to `renderer/` (convert game state to ASCII art)
3. **Integration point** is in `ui/screen_main.go`:

```go
// In renderGamePanel()
func (m Model) renderGamePanel(width, height int) string {
    // Replace this placeholder with actual game rendering
    gameContent := renderer.RenderWorld(m.connMgr.GetState(), width, height)
    return gameContent
}
```

## 🚀 Running the Client

```bash
# Build
go build -o bin/client ./cmd/client

# Normal flow (starts at loading screen)
./bin/client

# Connect to custom server
./bin/client --server ws://example.com:8080/ws

# Test specific screens
./bin/client --screen loading
./bin/client --screen username
./bin/client --screen avatar
./bin/client --screen game
```

## 🎨 Styling

All colors and styles are in `ui/styles.go`:

**Color Palette (Earthy):**
- Light Beige (#E8C4A0) - Primary
- Light Green (#7EBB81) - Secondary
- Sage Green (#A8C9A4) - Accents
- Bright Sage (#B5D99C) - Success/Selected
- Light Taupe (#B8A890) - Muted
- Warm White (#F5F3ED) - Text

## 📝 TODO

- [ ] Integrate termloop game rendering
- [ ] Implement chat message handling
- [ ] Add proximity-based chat requests
- [ ] Implement game entity rendering
- [ ] Add map loading/rendering
- [ ] Implement collision detection
- [ ] Add sound effects (if desired)

## 🔧 Development Tips

- **Add new screens:** Create `ui/screen_*.go` and add to ViewState enum
- **Add new events:** Add to `connection/events.go` and handle in manager
- **Modify colors:** Edit `ui/styles.go`
- **Debug:** Use `--screen` flag to test individual screens
