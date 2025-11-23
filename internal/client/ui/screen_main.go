package ui

import (
	"fmt"
	"os"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	gameMap     [250][400]string
	roomMap     [250][400]int
	gameMapOnce sync.Once
	gameMapErr  error
	roomMapErr  error
)

func getGameMap() ([250][400]string, error) {
	gameMapOnce.Do(func() {
		gameMap, gameMapErr = fillGameMap()
		roomMap, roomMapErr = fillRoomMap()
	})
	return gameMap, gameMapErr
}

func getRoomMap() ([250][400]int, error) {
	// Ensure maps are loaded
	_, _ = getGameMap()
	return roomMap, roomMapErr
}

// updateMainGame handles main game screen
func (m Model) updateMainGame(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		if m.connMgr != nil {
			m.connMgr.Disconnect()
		}
		return m, tea.Quit

	// Movement keys (placeholder for now - will be handled by game renderer)
	case "up", "w":
		// TODO: Send movement to server via connection manager
		// if m.connMgr != nil && m.connMgr.IsConnected() {
		//     m.connMgr.SendMove(x, y-1, "up")
		// }
	case "down", "s":
		// TODO: Send movement to server
	case "left", "a":
		// TODO: Send movement to server
	case "right", "d":
		// TODO: Send movement to server
	}

	return m, nil
}

// viewMainGame renders the split-screen main game view
func (m Model) viewMainGame() string {
	// Calculate dimensions (70% game, 30% chat)
	gameWidth := int(float64(m.width) * 0.7)
	chatWidth := m.width - gameWidth - 10 // Account for borders and margins
	contentHeight := m.height - 10        // Leave more room for spacing

	if contentHeight < 10 {
		contentHeight = 10
	}

	// Game section (left 70%)
	gameContent := m.renderGamePanel(gameWidth, contentHeight)
	gameBox := gameBoxStyle.
		Width(gameWidth).
		Height(contentHeight + 3).
		Render(gameContent)

	// Chat section (right 30%)
	chatContent := m.renderChatPanel(chatWidth, contentHeight)
	chatBox := chatBoxStyle.
		Width(chatWidth).
		Height(contentHeight + 3).
		Render(chatContent)

	// Join game and chat horizontally
	mainContent := lipgloss.JoinHorizontal(
		lipgloss.Top,
		gameBox,
		chatBox,
	)

	// Status bar at the bottom
	statusBar := m.renderStatusBar()

	// Calculate positions
	centeredMain := lipgloss.Place(m.width, m.height-4, lipgloss.Center, lipgloss.Top, mainContent)
	bottomStatus := lipgloss.Place(m.width, 4, lipgloss.Center, lipgloss.Bottom, statusBar)

	return centeredMain + bottomStatus
}

var (
	wallStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#5C4A37")). // Warm medium brown (walls)
			Render(" ")

	inaccessibleStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#6B5B4A")). // Warm tan-brown (inaccessible)
				Render(" ")

	roomStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#4A5D4A")). // Muted sage green (rooms)
			Render(" ")

	backgroundStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#D2B48C")). // Warm light beige - easy on the eyes
			Render(" ")

	transparentStyle = lipgloss.NewStyle().
				Render(" ") // Transparent - no background color
)

// fillGameMap loads the map file and returns a 2D array of styled strings
func fillGameMap() ([250][400]string, error) {
	data, err := os.ReadFile("internal/client/game_assets/map.txt")
	if err != nil {
		return [250][400]string{}, fmt.Errorf("failed to load map.txt: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	var result [250][400]string

	for i := range result {
		for j := range result[i] {
			result[i][j] = transparentStyle
		}
	}

	// Fill in the map data
	for i, line := range lines {
		if i >= 250 { // Safety check for rows
			break
		}

		line = strings.TrimRight(line, " \t\r")
		wallFound := false
		for j, char := range line {
			if j >= 400 { // Safety check for columns
				break
			}

			// Check if current character is a wall/room/inaccessible
			if char == 'o' || char == 'i' || char == 'r' {
				wallFound = true
			}

			var styledChar string
			if !wallFound {
				styledChar = transparentStyle
			} else {
				switch char {
				case 'o': // outer wall - solid dark brown
					styledChar = wallStyle
				case 'i': // inaccessible - solid dark gray
					styledChar = inaccessibleStyle
				case 'r': // room - solid dark gray
					styledChar = roomStyle
				default: // space or other characters - preserve as-is
					styledChar = backgroundStyle
				}
			}
			result[i][j] = styledChar
		}
	}

	return result, nil
}

// fillRoomMap fills the room map with int annotations to indicate room number.
// Returns -1 for walls (r, o, i), 2 for empty spaces not in rooms, room number (>=3) for spaces in rooms.
// Rooms are defined by four walls ('r' characters), and adjacent rooms are separated by 'r' walls.
func fillRoomMap() ([250][400]int, error) {
	data, err := os.ReadFile("internal/client/game_assets/map.txt")
	if err != nil {
		return [250][400]int{}, fmt.Errorf("failed to load map.txt: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	var result [250][400]int
	var mapChars [250][400]rune

	// Initialize all cells and read map characters
	for i, line := range lines {
		if i >= 250 {
			break
		}
		line = strings.TrimRight(line, " \t\r")

		for j := range result[i] {
			result[i][j] = 0 // Uninitialized marker
			if j < len(line) {
				mapChars[i][j] = rune(line[j])
			} else {
				mapChars[i][j] = ' '
			}
		}
	}

	// First pass: mark all walls (r, o, i) as -1
	for i := 0; i < 250; i++ {
		for j := 0; j < 400; j++ {
			char := mapChars[i][j]
			if char == 'r' || char == 'o' || char == 'i' {
				result[i][j] = -1
			}
		}
	}

	// Second pass: identify and mark spaces outside 'r' boundaries as 2
	// This uses flood fill from edges to mark unenclosed spaces
	// Only 'r' characters block the flood fill - 'o' and 'i' don't block it
	for i := 0; i < 250; i++ {
		for j := 0; j < 400; j++ {
			// Start flood fill from edge spaces that aren't 'r' characters
			// 'o' and 'i' are walls but don't define room boundaries
			if (i == 0 || i == 249 || j == 0 || j == 399) && mapChars[i][j] != 'r' {
				markOutsideSpaces(&result, &mapChars, i, j)
			}
		}
	}

	// Third pass: assign room numbers only to spaces that are enclosed by 'r' boundaries
	// A space is in a room only if it cannot reach the edge without passing through 'r' characters
	roomNum := 3
	for i := 0; i < 250; i++ {
		for j := 0; j < 400; j++ {
			// Skip if already marked (wall, outside, or already in a room)
			if result[i][j] != 0 {
				continue
			}

			// This is an unvisited space that couldn't be reached from edges
			// It must be enclosed by 'r' boundaries - assign it a room number
			// Flood fill to assign all connected spaces the same room number
			floodFillRoom(&result, &mapChars, i, j, roomNum)
			roomNum++
		}
	}

	// Convert any remaining 0s (shouldn't happen, but safety check) to 2
	for i := 0; i < 250; i++ {
		for j := 0; j < 400; j++ {
			if result[i][j] == 0 {
				result[i][j] = 2
			}
		}
	}

	return result, nil
}

// markOutsideSpaces marks spaces outside 'r' boundaries as 2 using flood fill
// Only 'r' characters block the flood fill - 'o' and 'i' don't block it
// This ensures that only spaces enclosed by 'r' boundaries are considered rooms
func markOutsideSpaces(result *[250][400]int, mapChars *[250][400]rune, startY, startX int) {
	type point struct {
		y, x int
	}
	stack := []point{{startY, startX}}

	for len(stack) > 0 {
		p := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		// Check bounds
		if p.y < 0 || p.y >= 250 || p.x < 0 || p.x >= 400 {
			continue
		}

		// Skip if already marked
		if result[p.y][p.x] == 2 {
			continue
		}

		// Only 'r' characters block the flood fill - 'o' and 'i' are passable for this check
		// This is because rooms are defined by 'r' boundaries, not 'o' or 'i'
		if mapChars[p.y][p.x] == 'r' {
			// This is an 'r' wall - don't mark it as outside, don't continue flood fill
			continue
		}

		// Mark as outside (not enclosed by 'r' boundaries)
		result[p.y][p.x] = 2

		// Add neighbors to stack (only if not 'r' characters)
		if p.y > 0 && mapChars[p.y-1][p.x] != 'r' {
			stack = append(stack, point{p.y - 1, p.x}) // up
		}
		if p.y < 249 && mapChars[p.y+1][p.x] != 'r' {
			stack = append(stack, point{p.y + 1, p.x}) // down
		}
		if p.x > 0 && mapChars[p.y][p.x-1] != 'r' {
			stack = append(stack, point{p.y, p.x - 1}) // left
		}
		if p.x < 399 && mapChars[p.y][p.x+1] != 'r' {
			stack = append(stack, point{p.y, p.x + 1}) // right
		}
	}
}

// floodFillRoom assigns a room number to all connected spaces starting from (startY, startX)
// This ensures all spaces in the same enclosed region get the same room number
func floodFillRoom(result *[250][400]int, mapChars *[250][400]rune, startY, startX, roomNum int) {
	type point struct {
		y, x int
	}
	stack := []point{{startY, startX}}

	for len(stack) > 0 {
		p := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		// Check bounds
		if p.y < 0 || p.y >= 250 || p.x < 0 || p.x >= 400 {
			continue
		}

		// Skip if already marked (wall, outside, or already in a room)
		if result[p.y][p.x] != 0 {
			continue
		}

		// Assign room number
		result[p.y][p.x] = roomNum

		// Add neighbors to stack (only unvisited spaces, not walls)
		if p.y > 0 && result[p.y-1][p.x] == 0 {
			stack = append(stack, point{p.y - 1, p.x}) // up
		}
		if p.y < 249 && result[p.y+1][p.x] == 0 {
			stack = append(stack, point{p.y + 1, p.x}) // down
		}
		if p.x > 0 && result[p.y][p.x-1] == 0 {
			stack = append(stack, point{p.y, p.x - 1}) // left
		}
		if p.x < 399 && result[p.y][p.x+1] == 0 {
			stack = append(stack, point{p.y, p.x + 1}) // right
		}
	}
}

// renderGamePanel renders the game world panel (left 70%)
func (m Model) renderGamePanel(width, height int) string {
	gameTitle := lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true).
		Width(width).
		Align(lipgloss.Center).
		Render("🎮 GAME WORLD")

	// TODO: This is where your teammate's termloop/game rendering will go
	gamePlaceholder := centerStyle.
		Width(width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center).
		Render("COMING SOON\n\n" + mutedStyle.Render("(Game renderer will be integrated here)"))

	return lipgloss.JoinVertical(
		lipgloss.Left,
		gameTitle,
		gamePlaceholder,
	)
}

// renderChatPanel renders the chat panel (right 30%)
func (m Model) renderChatPanel(width, height int) string {
	chatTitle := lipgloss.NewStyle().
		Foreground(accentColor).
		Bold(true).
		Width(width).
		Align(lipgloss.Center).
		Render("💬 CHAT")

	// TODO: Chat messages will be rendered here
	chatPlaceholder := centerStyle.
		Width(width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center).
		Render("COMING SOON\n\n" + mutedStyle.Render("(Chat messages here)"))

	return lipgloss.JoinVertical(
		lipgloss.Left,
		chatTitle,
		chatPlaceholder,
	)
}

// renderStatusBar renders the bottom status bar
func (m Model) renderStatusBar() string {
	playerInfo := lipgloss.NewStyle().
		Foreground(successColor).
		Bold(true).
		Render("Player: " + m.playerName)

	avatarDisplay := lipgloss.NewStyle().
		Foreground(secondaryColor).
		Render(strings.ReplaceAll(m.avatar.Render(), "\n", " "))

	controls := mutedStyle.Render("WASD/Arrows: Move  •  ESC: Quit")

	return lipgloss.NewStyle().
		Foreground(fgColor).
		Width(m.width).
		Padding(1, 0).
		Align(lipgloss.Center).
		Render(playerInfo + "  " + avatarDisplay + "  •  " + controls)
}
