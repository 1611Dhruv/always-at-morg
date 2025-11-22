package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yourusername/always-at-morg/internal/client"
)

func main() {
	serverURL := flag.String("server", "ws://localhost:8080/ws", "WebSocket server URL")
	mode := flag.String("mode", "menu", "Start mode: menu, lobby, or game")
	roomID := flag.String("room", "default-room", "Room ID to join")
	playerName := flag.String("name", "Player1", "Player name")
	useTermloop := flag.Bool("termloop", false, "Use termloop for game rendering")
	flag.Parse()

	switch *mode {
	case "menu":
		// Start with Bubble Tea menu
		runBubbleTea()

	case "lobby":
		// Connect and show lobby
		runLobby(*serverURL, *roomID, *playerName)

	case "game":
		// Connect and start game directly
		if *useTermloop {
			runTermloopGame(*serverURL, *roomID, *playerName)
		} else {
			runBubbleTeaGame(*serverURL, *roomID, *playerName)
		}

	default:
		fmt.Printf("Unknown mode: %s\n", *mode)
		flag.Usage()
		os.Exit(1)
	}
}

// runBubbleTea runs the Bubble Tea menu interface
func runBubbleTea() {
	p := tea.NewProgram(client.NewModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

// runLobby connects to server and shows lobby
func runLobby(serverURL, roomID, playerName string) {
	wsClient, err := client.NewWSClient(serverURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer wsClient.Close()

	stateReceiver := client.NewGameStateReceiver()

	if err := wsClient.JoinRoom(roomID, playerName); err != nil {
		log.Fatalf("Failed to join room: %v", err)
	}

	// Listen for messages in background
	go func() {
		for msg := range wsClient.Receive() {
			stateReceiver.HandleMessage(msg)
		}
	}()

	// Create Bubble Tea model in lobby state
	model := client.NewModel()
	// Note: You would need to extend the Model to support this initialization
	// For now, use the menu-based flow

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

// runTermloopGame runs the game using termloop
func runTermloopGame(serverURL, roomID, playerName string) {
	wsClient, err := client.NewWSClient(serverURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer wsClient.Close()

	stateReceiver := client.NewGameStateReceiver()

	if err := wsClient.JoinRoom(roomID, playerName); err != nil {
		log.Fatalf("Failed to join room: %v", err)
	}

	log.Printf("Starting termloop game in room %s as %s", roomID, playerName)

	game := client.NewTermloopGame(wsClient, stateReceiver)
	game.Start()
}

// runBubbleTeaGame runs the game using Bubble Tea's game view
func runBubbleTeaGame(serverURL, roomID, playerName string) {
	wsClient, err := client.NewWSClient(serverURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer wsClient.Close()

	stateReceiver := client.NewGameStateReceiver()

	if err := wsClient.JoinRoom(roomID, playerName); err != nil {
		log.Fatalf("Failed to join room: %v", err)
	}

	// Create Bubble Tea model in game state
	model := client.NewModel()
	// Note: You would need to extend the Model to support this initialization

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
