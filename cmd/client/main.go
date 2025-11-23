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
	serverURL := flag.String("server", "ws://join.always-at-morg.bid/ws", "WebSocket server URL")
	screen := flag.String("screen", "", "Screen to display (for testing): loading, username, avatar, game")
	flag.Parse()

	// Allow positional argument as server URL (for backwards compatibility)
	if flag.NArg() > 0 {
		url := flag.Arg(0)
		serverURL = &url
	}

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
	finalModel, err := p.Run()
	if err != nil {
		log.Fatal(err)
	}

	// Clean up: disconnect when exiting
	if m, ok := finalModel.(ui.Model); ok {
		m.Disconnect()
	}
}
