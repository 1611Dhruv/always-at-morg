# Always at Morg - Multiplayer Terminal Game

A real-time multiplayer terminal game connecting UW Madison students. Hang out in virtual Morgridge Hall from anywhere!

Built with Go (backend/client) and React (website), featuring WebSocket communication and terminal UI using Bubble Tea.

## Quick Start for Players

Install and play in one command:

**Unix/Mac/Linux:**
```bash
curl -fsSL https://always-at-morg.bid/install.sh | bash
morg
```

**Windows PowerShell:**
```powershell
iwr -useb https://always-at-morg.bid/install.ps1 | iex
morg
```

Visit the website: **https://always-at-morg.bid**

## Project Structure

```
always-at-morg/
├── cmd/                    # Go application entry points
│   ├── client/main.go     # Client application
│   └── server/main.go     # Server application
├── internal/              # Go internal packages
│   ├── client/           # Client logic & UI
│   ├── server/           # Server logic & game state
│   └── protocol/         # Shared WebSocket protocol
├── website/              # React website
│   ├── src/             # React source code
│   ├── public/          # Static assets + installers
│   │   ├── install.sh   # Unix/Mac/Linux installer
│   │   ├── install.ps1  # Windows PowerShell installer
│   │   └── releases/    # Binary releases (auto-generated)
│   └── build/           # Production build (generated)
├── build-releases.sh    # Build binaries for all platforms
└── install.sh          # Installer script
```

## For Developers

### Prerequisites
- Go 1.21+
- Node.js 18+ (for website)

### Running Locally

**1. Start the Server:**
```bash
go run cmd/server/main.go
# Server runs on ws://localhost:8080/ws
```

**2. Run the Client:**
```bash
go build -o client cmd/client/main.go
./client
# Auto-connects to ws://always-at-morg.bid:8080/ws by default
# Or specify custom server: ./client ws://localhost:8080/ws
```

**3. Run the Website (optional):**
```bash
cd website
npm install
npm start
# Visit http://localhost:3000
```

### Game Controls

- `W A S D` or Arrow Keys - Move around
- `Enter` - Start chatting
- `G` - Global chat mode
- `O` - Room chat mode
- `P` - Private chat mode
- `Esc` - Exit chat
- `Ctrl+C` - Quit game

## Deployment Guide

### 1. Build All Platform Binaries

```bash
./build-releases.sh
```

This creates binaries in `website/public/releases/`:
- `always-at-morg-darwin_amd64` (macOS Intel)
- `always-at-morg-darwin_arm64` (macOS Apple Silicon)
- `always-at-morg-linux_amd64` (Linux x86_64)
- `always-at-morg-linux_arm64` (Linux ARM64)
- `always-at-morg-linux_arm` (Linux ARM)
- `always-at-morg-windows_amd64.exe` (Windows)

### 2. Build the Website

```bash
cd website
npm run build
```

Production build created in `website/build/`

### 3. Deploy to always-at-morg.bid

Upload `website/build/` to your web server:

```
always-at-morg.bid/
├── index.html          # React website
├── install.sh          # Unix installer
├── install.ps1         # Windows installer
└── releases/           # Binary downloads
    ├── always-at-morg-darwin_amd64
    ├── always-at-morg-darwin_arm64
    └── ...
```

### 4. Deploy the Game Server

```bash
go build -o server cmd/server/main.go
./server
```

Make accessible at `ws://always-at-morg.bid:8080/ws`

## How It Works

### Installation Flow

1. User runs: `curl -fsSL https://always-at-morg.bid/install.sh | bash`
2. Script detects OS/architecture
3. Downloads binary from `https://always-at-morg.bid/releases/`
4. Installs to `~/.local/bin/morg` (Unix) or `%LOCALAPPDATA%\Programs\morg` (Windows)
5. User runs: `morg`
6. Client auto-connects to production server

### Architecture

**Backend:**
- Go WebSocket server (`gorilla/websocket`)
- Room-based game sessions
- Real-time state sync at 20 ticks/second
- Chat system (global, room, private)
- Treasure hunt mini-game

**Client:**
- Bubble Tea terminal UI
- WebSocket client
- Avatar customization (3x3 pixels)
- Real-time multiplayer rendering

**Website:**
- React single-page app
- Font Awesome icons
- Minimal, clean design
- UW Madison theme (#c5050c)

**Protocol:**
- JSON-based WebSocket messages
- Shared types in `internal/protocol/`
- Kuluchified state updates (unified tick)

## Features

- 🏛️ **Explore Morgridge Hall** - Virtual recreation with multiple rooms
- 💬 **Chat System** - Global, room, and private messaging
- 🎮 **Treasure Hunt** - Interactive mini-game
- 👥 **Multiplayer** - See other players move in real-time
- 🎨 **Custom Avatars** - Create your 3x3 pixel character
- ⚡ **Lightning Fast** - Runs entirely in your terminal

## Protocol Messages

### Key Message Types

**Client → Server:**
- `join_room` - Join game room
- `player_move` - Movement update
- `global_chat_message` - Global chat
- `room_chat_message` - Room chat
- `chat_message` - Private message
- `treasure_hunt_guess` - Submit answer

**Server → Client:**
- `room_joined` - Join confirmation
- `kuluchified_state` - Unified game state (tick)
- `global_chat_messages` - Chat history
- `room_chat_messages` - Room chat history
- `treasure_hunt_state` - Treasure hunt updates

## Tech Stack

- **Backend**: Go 1.21+, gorilla/websocket
- **Client TUI**: Bubble Tea, Lipgloss
- **Frontend**: React 18
- **Icons**: Font Awesome 6
- **Styling**: Pure CSS3
- **Tools**: UUID generation, JSON protocol

## Contributing

This is a student project for UW Madison. Feel free to fork and customize!

### Adding Features

1. Update protocol: `internal/protocol/messages.go`
2. Add server logic: `internal/server/`
3. Add client logic: `internal/client/`
4. Update UI: `internal/client/ui/`

## License

MIT

---

Made with ❤️ for **UW Madison** students.

**On, Wisconsin!** 🦡
