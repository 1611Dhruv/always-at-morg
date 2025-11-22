package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yourusername/always-at-morg/internal/client/ui"
)

func main() {
	serverURL := flag.String("server", "ws://localhost:8080/ws", "WebSocket server URL")
	screen := flag.String("screen", "", "Screen to display (for testing): loading, username, avatar, game")
	flag.Parse()

	var model ui.Model

	// If screen flag is provided, start at that screen (for testing)
	if *screen != "" {
		var viewState ui.ViewState
		switch *screen {
		case "loading":
			viewState = ui.ViewLoading
		case "username":
			viewState = ui.ViewUsernameEntry
		case "avatar":
			viewState = ui.ViewAvatarCustomization
		case "game":
			viewState = ui.ViewMainGame
		default:
			fmt.Printf("Unknown screen: %s\n", *screen)
			fmt.Println("Valid screens: loading, username, avatar, game")
			os.Exit(1)
		}
		model = ui.NewModelWithView(viewState)
	} else {
		// Normal flow: start with loading screen and connect to server
		model = ui.NewModel(*serverURL)
	}

	// Run Bubble Tea
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
