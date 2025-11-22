package client

import (
	"fmt"
	"log"

	tl "github.com/JoelOtter/termloop"
	"github.com/yourusername/always-at-morg/internal/protocol"
)

// TermloopGame manages the termloop game instance
type TermloopGame struct {
	game          *tl.Game
	level         *tl.BaseLevel
	wsClient      *WSClient
	stateReceiver *GameStateReceiver
	players       map[string]*PlayerEntity
	localPlayerID string
}

// NewTermloopGame creates a new termloop game instance
func NewTermloopGame(wsClient *WSClient, stateReceiver *GameStateReceiver) *TermloopGame {
	game := tl.NewGame()
	level := tl.NewBaseLevel(tl.Cell{
		Bg: tl.ColorBlack,
		Fg: tl.ColorWhite,
		Ch: ' ',
	})

	game.Screen().SetLevel(level)

	tg := &TermloopGame{
		game:          game,
		level:         level,
		wsClient:      wsClient,
		stateReceiver: stateReceiver,
		players:       make(map[string]*PlayerEntity),
	}

	// Add update handler
	level.AddEntity(NewGameUpdater(tg))

	return tg
}

// Start starts the termloop game
func (tg *TermloopGame) Start() {
	tg.game.Start()
}

// Stop stops the termloop game
func (tg *TermloopGame) Stop() {
	tg.game.End()
}

// UpdateFromState updates the game from the server state
func (tg *TermloopGame) UpdateFromState() {
	state := tg.stateReceiver.GetState()

	// Update or create players
	for id, player := range state.Players {
		if entity, exists := tg.players[id]; exists {
			// Update existing player
			entity.SetPosition(player.X, player.Y)
			entity.player = player
		} else {
			// Create new player
			entity := NewPlayerEntity(player, id == tg.localPlayerID, tg.wsClient)
			tg.players[id] = entity
			tg.level.AddEntity(entity)
		}
	}

	// Remove disconnected players
	for id := range tg.players {
		if _, exists := state.Players[id]; !exists {
			tg.level.RemoveEntity(tg.players[id])
			delete(tg.players, id)
		}
	}
}

// PlayerEntity represents a player in the termloop game
type PlayerEntity struct {
	player      protocol.Player
	isLocal     bool
	wsClient    *WSClient
	x, y        int
	prevX, prevY int
}

// NewPlayerEntity creates a new player entity
func NewPlayerEntity(player protocol.Player, isLocal bool, wsClient *WSClient) *PlayerEntity {
	return &PlayerEntity{
		player:   player,
		isLocal:  isLocal,
		wsClient: wsClient,
		x:        player.X,
		y:        player.Y,
		prevX:    player.X,
		prevY:    player.Y,
	}
}

// Draw draws the player entity
func (pe *PlayerEntity) Draw(screen *tl.Screen) {
	// Get color
	color := tl.ColorWhite
	switch pe.player.Color {
	case "red":
		color = tl.ColorRed
	case "green":
		color = tl.ColorGreen
	case "blue":
		color = tl.ColorBlue
	case "yellow":
		color = tl.ColorYellow
	case "magenta":
		color = tl.ColorMagenta
	case "cyan":
		color = tl.ColorCyan
	}

	// Draw player character
	char := '@'
	if !pe.isLocal {
		char = 'O'
	}

	screen.RenderCell(pe.x, pe.y, &tl.Cell{
		Fg: color,
		Ch: char,
	})

	// Draw player name above
	nameX := pe.x - len(pe.player.Name)/2
	for i, ch := range pe.player.Name {
		screen.RenderCell(nameX+i, pe.y-1, &tl.Cell{
			Fg: tl.ColorWhite,
			Ch: ch,
		})
	}

	// Draw score
	scoreText := fmt.Sprintf("Score: %d", pe.player.Score)
	scoreX := pe.x - len(scoreText)/2
	for i, ch := range scoreText {
		screen.RenderCell(scoreX+i, pe.y+1, &tl.Cell{
			Fg: tl.ColorYellow,
			Ch: ch,
		})
	}
}

// Tick handles player updates
func (pe *PlayerEntity) Tick(event tl.Event) {
	if !pe.isLocal {
		return
	}

	// Handle input for local player
	if event.Type == tl.EventKey {
		pe.prevX = pe.x
		pe.prevY = pe.y

		switch event.Key {
		case tl.KeyArrowUp:
			pe.y--
			pe.sendMove("up")
		case tl.KeyArrowDown:
			pe.y++
			pe.sendMove("down")
		case tl.KeyArrowLeft:
			pe.x--
			pe.sendMove("left")
		case tl.KeyArrowRight:
			pe.x++
			pe.sendMove("right")
		}
	}
}

// sendMove sends movement to the server
func (pe *PlayerEntity) sendMove(direction string) {
	if pe.wsClient != nil {
		pe.wsClient.SendMove(pe.x, pe.y, direction)
	}
}

// SetPosition sets the player position
func (pe *PlayerEntity) SetPosition(x, y int) {
	pe.x = x
	pe.y = y
}

// Position returns the player position
func (pe *PlayerEntity) Position() (int, int) {
	return pe.x, pe.y
}

// Size returns the entity size
func (pe *PlayerEntity) Size() (int, int) {
	return 1, 1
}

// GameUpdater handles game state updates
type GameUpdater struct {
	game *TermloopGame
	ticks int
}

// NewGameUpdater creates a new game updater
func NewGameUpdater(game *TermloopGame) *GameUpdater {
	return &GameUpdater{game: game}
}

// Draw does nothing for the updater
func (gu *GameUpdater) Draw(screen *tl.Screen) {
	// Draw game info
	info := fmt.Sprintf("Players: %d | Tick: %d",
		len(gu.game.players),
		gu.game.stateReceiver.GetState().Tick)

	for i, ch := range info {
		screen.RenderCell(i, 0, &tl.Cell{
			Fg: tl.ColorWhite,
			Ch: ch,
		})
	}
}

// Tick updates the game state
func (gu *GameUpdater) Tick(event tl.Event) {
	gu.ticks++

	// Update game state every few ticks
	if gu.ticks%3 == 0 {
		// Process WebSocket messages
		select {
		case msg := <-gu.game.wsClient.Receive():
			gu.game.stateReceiver.HandleMessage(msg)
			gu.game.UpdateFromState()
		default:
		}
	}

	// Handle quit
	if event.Type == tl.EventKey && event.Key == tl.KeyEsc {
		log.Println("Exiting game...")
		gu.game.Stop()
	}
}

// Position returns the updater position
func (gu *GameUpdater) Position() (int, int) {
	return 0, 0
}

// Size returns the updater size
func (gu *GameUpdater) Size() (int, int) {
	return 0, 0
}
