package ui

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yourusername/always-at-morg/internal/protocol"
)

var (
	roomMap     [250][400]string
	roomMapOnce sync.Once
	roomMapErr  error
)

func getRoomMap() ([250][400]string, error) {
	roomMapOnce.Do(func() {
		roomMap, roomMapErr = fillRoomMap()
	})
	return roomMap, roomMapErr
}

// parsePosition parses a position string "Y:X" into integer coordinates
// Server uses format "Y:X" (e.g., "52:200" means Y=52, X=200)
func parsePosition(pos string) (x, y int) {
	parts := strings.Split(pos, ":")
	if len(parts) != 2 {
		return 0, 0
	}
	y, _ = strconv.Atoi(parts[0]) // First part is Y
	x, _ = strconv.Atoi(parts[1]) // Second part is X
	return x, y
}

// isWalkable checks if a position is walkable (not a wall)
func isWalkable(x, y int) bool {
	roomMap, err := getRoomMap()
	if err != nil {
		return false
	}

	// Check bounds
	if y < 0 || y >= 250 || x < 0 || x >= 400 {
		return false
	}

	// Wall characters ("r", "o", "i") are not walkable
	// "e" (entrances), "-1" (hallways), and room numbers ("1", "2", "3", ...) are walkable
	value := roomMap[y][x]
	return value != "r" && value != "o" && value != "i"
}

// hasPlayerNearby checks if there's a player within 4 tiles (Chebyshev distance)
func (m *Model) hasPlayerNearby(newX, newY int) bool {
	if m.connMgr == nil {
		return false
	}

	gameState := m.connMgr.GetState()
	if gameState == nil {
		return false
	}

	// Check all players
	for username, player := range gameState.Players {
		// Skip self
		if username == m.userName {
			continue
		}

		// Parse player position
		playerX, playerY := parsePosition(player.Pos)

		// Calculate Chebyshev distance (max of abs differences)
		// This creates a square area around each player
		dx := abs(newX - playerX)
		dy := abs(newY - playerY)
		distance := max(dx, dy)

		// If any player is within 4 tiles, can't move there
		if distance <= 4 {
			return true
		}
	}

	return false
}

// abs returns the absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// canMoveTo checks if the player can move to a position
func (m *Model) canMoveTo(newX, newY int) bool {
	// Check if position is walkable (not a wall)
	if !isWalkable(newX, newY) {
		return false
	}

	// Check if there's a player nearby
	if m.hasPlayerNearby(newX, newY) {
		return false
	}

	return true
}

// createAvatarFromIndices creates an Avatar from protocol avatar indices
func createAvatarFromIndices(indices []int) Avatar {
	if len(indices) != 3 {
		return NewAvatar() // Default avatar if invalid
	}
	return Avatar{
		HeadIndex:  indices[0],
		TorsoIndex: indices[1],
		LegsIndex:  indices[2],
	}
}

// updateMainGame handles main game screen
func (m Model) updateMainGame(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle chat input if active
	if m.chatInputActive {
		switch msg.String() {
		case "esc":
			// Cancel chat input
			m.chatInputActive = false
			m.chatInput = ""
			return m, func() tea.Msg { return tea.ClearScreen() }

		case "enter":
			// Send message
			if len(m.chatInput) > 0 {
				if m.connMgr != nil && m.connMgr.IsConnected() {
					if m.chatMode == ChatModeGlobal {
						// Send global chat message
						m.connMgr.SendGlobalChat(m.userName, m.chatInput)
					}
				}
				// Clear input but stay in chat mode
				m.chatInput = ""
			}
			return m, nil

		case "backspace":
			if len(m.chatInput) > 0 {
				m.chatInput = m.chatInput[:len(m.chatInput)-1]
			}
			return m, nil

		case " ":
			// Handle space explicitly
			if len(m.chatInput) < 100 {
				m.chatInput += " "
			}
			return m, nil

		default:
			// Add character to input (limit to 100 chars)
			// Use Runes to properly handle shift+letter for capitals
			if msg.Type == tea.KeyRunes && len(m.chatInput) < 100 {
				for _, r := range msg.Runes {
					m.chatInput += string(r)
				}
			}
			return m, nil
		}
	}

	// Normal game controls
	switch msg.String() {
	case "ctrl+c":
		if m.connMgr != nil {
			m.connMgr.Disconnect()
		}
		return m, tea.Quit

	// Chat controls
	case "t", "T":
		// Start typing in chat
		m.chatInputActive = true
		m.chatInput = ""
		return m, func() tea.Msg { return tea.ClearScreen() }

	case "g", "G":
		// Switch to global chat
		m.chatMode = ChatModeGlobal
		m.chatTarget = ""
		return m, nil

	case "p", "P":
		// Switch to private chat (for now, set to placeholder)
		m.chatMode = ChatModePrivate
		// TODO: Implement player selection UI
		m.chatTarget = "Player2" // Placeholder
		return m, nil

	// Movement keys - WASD and arrow keys
	case "up", "w", "W":
		m.handleMovement(0, -1) // Move up (Y decreases)
	case "down", "s", "S":
		m.handleMovement(0, 1) // Move down (Y increases)
	case "left", "a", "A":
		m.handleMovement(-1, 0) // Move left (X decreases)
	case "right", "d", "D":
		m.handleMovement(1, 0) // Move right (X increases)
	}

	return m, nil
}

// handleMovement handles player movement requests
func (m *Model) handleMovement(dx, dy int) {
	// Check if connected
	if m.connMgr == nil || !m.connMgr.IsConnected() {
		return
	}

	// Get current player position
	gameState := m.connMgr.GetState()
	if gameState == nil {
		return
	}

	player, exists := gameState.Players[m.userName]
	if !exists {
		return
	}

	// Parse current position
	currentX, currentY := parsePosition(player.Pos)

	// Calculate new position
	newX := currentX + dx
	newY := currentY + dy

	// Validate movement
	if !m.canMoveTo(newX, newY) {
		return // Invalid move, do nothing
	}

	// Send move request to server
	m.connMgr.SendPlayerMove(newX, newY)
}

// viewMainGame renders the split-screen main game view
func (m Model) viewMainGame() string {
	// Repopulate grids to ensure viewport is current (player may have moved)
	mPtr := &m
	mPtr.populateGrids()

	// Calculate dimensions (70% game, 30% chat)
	gameWidth := int(float64(m.width) * 0.7)
	chatWidth := m.width - gameWidth - 10 // Account for borders and margins
	contentHeight := m.height - 10        // Leave more room for spacing

	if contentHeight < 10 {
		contentHeight = 10
	}

	// Chat section (right 30%)
	chatPanelHeight := contentHeight - 4 // Leave room for input box below
	chatContent := m.renderChatPanel(chatWidth, chatPanelHeight)
	chatBox := chatBoxStyle.
		Width(chatWidth).
		Height(chatPanelHeight).
		Render(chatContent)

	// Chat input box (below chat panel, adapts to chat width)
	chatInputBox := m.renderChatInputBox(chatWidth)

	// Combine chat panel and input box vertically
	chatSection := lipgloss.JoinVertical(
		lipgloss.Left,
		chatBox,
		chatInputBox,
	)

	// Game section (left 70%) - extends to match full chat section height
	gameContent := m.renderGamePanel(gameWidth, contentHeight)
	gameBox := gameBoxStyle.
		Width(gameWidth).
		Height(contentHeight + 3).
		Render(gameContent)

	// Join game and chat sections horizontally
	mainContent := lipgloss.JoinHorizontal(
		lipgloss.Top,
		gameBox,
		chatSection,
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

	entranceStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#6A7D6A")). // Lighter sage green (entrances)
			Render(" ")

	backgroundStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#D2B48C")). // Warm light beige - easy on the eyes
			Render(" ")

	transparentStyle = lipgloss.NewStyle().
				Render(" ") // Transparent - no background color
	// Player rendering styles
	currentPlayerUsernameStyle = lipgloss.NewStyle().
					Foreground(successColor).
					Background(lipgloss.Color("#3A4A3A")).
					Bold(true)

	otherPlayerUsernameStyle = lipgloss.NewStyle().
					Foreground(accentColor).
					Background(lipgloss.Color("#3A4A3A"))

	currentPlayerAvatarStyle = lipgloss.NewStyle().
					Foreground(highlightColor).
					Bold(true)

	otherPlayerAvatarStyle = lipgloss.NewStyle().
				Foreground(fgColor)
)

// StyledCell represents a single grid cell with optional player overlay
type StyledCell struct {
	StyledString string
	HasContent   bool
}

// getStyledCharFromRoomValue converts a room map value to a styled string for rendering
func getStyledCharFromRoomValue(value string) string {
	switch value {
	case "r": // room wall
		return roomStyle
	case "o": // outer wall
		return wallStyle
	case "i": // inaccessible
		return inaccessibleStyle
	case "e": // entrance
		return entranceStyle
	case "-1": // non-room space (hallway)
		return backgroundStyle
	default:
		// Room number or empty space - check if it's a numeric room number
		if value != "" {
			// It's a room number, use background style
			return backgroundStyle
		}
		// Empty/uninitialized, use transparent
		return transparentStyle
	}
}

// fillRoomMap fills the room map with string annotations.
// Returns map characters as keys ('r', 'o', 'i', 'e'), "-1" for spaces not in rooms, room number strings ("1", "2", ...) for spaces in rooms.
// Rooms are defined by four walls ('r' or 'e' characters), and adjacent rooms are separated by 'r'/'e' walls.
func fillRoomMap() ([250][400]string, error) {
	data, err := os.ReadFile("internal/client/game_assets/map.txt")
	if err != nil {
		return [250][400]string{}, fmt.Errorf("failed to load map.txt: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	var result [250][400]string
	var mapChars [250][400]rune

	// Initialize all cells and read map characters
		for i, line := range lines {
		if i >= 250 {
			break
		}
		line = strings.TrimRight(line, " \t\r")

		for j := range result[i] {
			result[i][j] = "" // Uninitialized marker
			if j < len(line) {
				mapChars[i][j] = rune(line[j])
			} else {
				mapChars[i][j] = ' '
			}
		}
	}

	// First pass: mark all walls (r, o, i, e) with their character as the key
	for i := 0; i < 250; i++ {
		for j := 0; j < 400; j++ {
			char := mapChars[i][j]
			if char == 'r' || char == 'o' || char == 'i' || char == 'e' {
				result[i][j] = string(char)
			}
		}
	}

	// Second pass: identify and mark spaces outside 'r'/'e' boundaries as "-1"
	// This uses flood fill from edges to mark unenclosed spaces
	// Only 'r' and 'e' characters block the flood fill - 'o' and 'i' don't block it
	for i := 0; i < 250; i++ {
		for j := 0; j < 400; j++ {
			// Start flood fill from edge spaces that aren't 'r' or 'e' characters
			// 'o' and 'i' are walls but don't define room boundaries
			if (i == 0 || i == 249 || j == 0 || j == 399) && mapChars[i][j] != 'r' && mapChars[i][j] != 'e' {
				markOutsideSpaces(&result, &mapChars, i, j)
			}
		}
	}

	// Third pass: assign room numbers only to spaces that are enclosed by 'r'/'e' boundaries
	// A space is in a room only if it cannot reach the edge without passing through 'r'/'e' characters
	roomNum := 1
	for i := 0; i < 250; i++ {
		for j := 0; j < 400; j++ {
			// Skip if already marked (wall, outside, or already in a room)
			if result[i][j] != "" {
				continue
			}

			// This is an unvisited space that couldn't be reached from edges
			// It must be enclosed by 'r'/'e' boundaries - assign it a room number
			// Flood fill to assign all connected spaces the same room number
			roomNumStr := strconv.Itoa(roomNum)
			floodFillRoom(&result, &mapChars, i, j, roomNumStr)
			roomNum++
		}
	}

	// Convert any remaining empty strings (shouldn't happen, but safety check) to "-1"
	for i := 0; i < 250; i++ {
		for j := 0; j < 400; j++ {
			if result[i][j] == "" {
				result[i][j] = "-1"
			}
		}
	}

	return result, nil
}

// markOutsideSpaces marks spaces outside 'r'/'e' boundaries as "-1" using flood fill
// Only 'r' and 'e' characters block the flood fill - 'o' and 'i' don't block it
// This ensures that only spaces enclosed by 'r'/'e' boundaries are considered rooms
func markOutsideSpaces(result *[250][400]string, mapChars *[250][400]rune, startY, startX int) {
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
		if result[p.y][p.x] == "-1" {
			continue
		}

		// Only 'r' and 'e' characters block the flood fill - 'o' and 'i' are passable for this check
		// This is because rooms are defined by 'r'/'e' boundaries, not 'o' or 'i'
		if mapChars[p.y][p.x] == 'r' || mapChars[p.y][p.x] == 'e' {
			// This is an 'r' or 'e' wall - don't mark it as outside, don't continue flood fill
			continue
		}

		// Mark as outside (not enclosed by 'r'/'e' boundaries)
		result[p.y][p.x] = "-1"

		// Add neighbors to stack (only if not 'r' or 'e' characters)
		if p.y > 0 && mapChars[p.y-1][p.x] != 'r' && mapChars[p.y-1][p.x] != 'e' {
			stack = append(stack, point{p.y - 1, p.x}) // up
		}
		if p.y < 249 && mapChars[p.y+1][p.x] != 'r' && mapChars[p.y+1][p.x] != 'e' {
			stack = append(stack, point{p.y + 1, p.x}) // down
		}
		if p.x > 0 && mapChars[p.y][p.x-1] != 'r' && mapChars[p.y][p.x-1] != 'e' {
			stack = append(stack, point{p.y, p.x - 1}) // left
		}
		if p.x < 399 && mapChars[p.y][p.x+1] != 'r' && mapChars[p.y][p.x+1] != 'e' {
			stack = append(stack, point{p.y, p.x + 1}) // right
		}
	}
}

// floodFillRoom assigns a room number to all connected spaces starting from (startY, startX)
// This ensures all spaces in the same enclosed region get the same room number
func floodFillRoom(result *[250][400]string, mapChars *[250][400]rune, startY, startX int, roomNumStr string) {
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
		// Also skip 'e' characters - they act like 'r' for boundaries but are marked as 'e'
		if result[p.y][p.x] != "" || mapChars[p.y][p.x] == 'e' {
			continue
		}

		// Assign room number
		result[p.y][p.x] = roomNumStr

		// Add neighbors to stack (only unvisited spaces, not walls or 'e' characters)
		if p.y > 0 && result[p.y-1][p.x] == "" && mapChars[p.y-1][p.x] != 'e' {
			stack = append(stack, point{p.y - 1, p.x}) // up
		}
		if p.y < 249 && result[p.y+1][p.x] == "" && mapChars[p.y+1][p.x] != 'e' {
			stack = append(stack, point{p.y + 1, p.x}) // down
		}
		if p.x > 0 && result[p.y][p.x-1] == "" && mapChars[p.y][p.x-1] != 'e' {
			stack = append(stack, point{p.y, p.x - 1}) // left
		}
		if p.x < 399 && result[p.y][p.x+1] == "" && mapChars[p.y][p.x+1] != 'e' {
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
		Render("Morgridge Hall")

	// Calculate actual viewport dimensions (accounting for borders and padding)
	viewportWidth := width - 4
	viewportHeight := height - 2

	// Render the actual game grid
	gameGrid := m.renderGameWorld(viewportWidth, viewportHeight)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		gameTitle,
		gameGrid,
	)
}

// calculateViewport calculates the camera position centered on the current player
func (m *Model) calculateViewport() (cameraX, cameraY int) {
	// Get game state
	if m.connMgr == nil {
		return -1, -1 // Signal: no connection, show blank/loading
	}

	gameState := m.connMgr.GetState()
	if gameState == nil {
		return -1, -1 // Signal: no game state, show blank/loading
	}

	// Get current player
	currentPlayer, exists := gameState.Players[m.userName]
	if !exists {
		return -1, -1 // Signal: player not spawned yet, show blank/loading
	}

	// Check if player has a valid position
	if currentPlayer.Pos == "" {
		return -1, -1 // Signal: no position assigned, show blank/loading
	}

	// Parse player position
	playerX, playerY := parsePosition(currentPlayer.Pos)

	// Validate parsed position
	if playerX == 0 && playerY == 0 {
		return -1, -1 // Signal: invalid position, show blank/loading
	}

	// Center camera on player
	cameraX = playerX - (m.GameWorldWidth / 2)
	cameraY = playerY - (m.GameWorldHeight / 2)

	// Clamp to world bounds [0, 400) x [0, 250)
	if cameraX < 0 {
		cameraX = 0
	}
	if cameraX+m.GameWorldWidth > 400 {
		cameraX = 400 - m.GameWorldWidth
	}

	if cameraY < 0 {
		cameraY = 0
	}
	if cameraY+m.GameWorldHeight > 250 {
		cameraY = 250 - m.GameWorldHeight
	}

	return cameraX, cameraY
}

// populateGrids fills GameWorldGrid from the room map (consolidated - only room map is used)
func (m *Model) populateGrids() {
	roomData, err := getRoomMap()
	if err != nil {
		// If error, initialize empty grid
		m.GameWorldGrid = make([][]string, m.GameWorldHeight)
		for i := range m.GameWorldGrid {
			m.GameWorldGrid[i] = make([]string, m.GameWorldWidth)
		}
		return
	}

	// Initialize grid
	m.GameWorldGrid = make([][]string, m.GameWorldHeight)
	for i := range m.GameWorldGrid {
		m.GameWorldGrid[i] = make([]string, m.GameWorldWidth)
	}

	// Populate from room map (viewport centered on player)
	cameraX, cameraY := m.calculateViewport()

	// If camera is at -1, -1, show blank/loading state (player not spawned yet)
	if cameraX == -1 && cameraY == -1 {
		// Fill with transparent/blank cells
		for y := 0; y < m.GameWorldHeight; y++ {
			for x := 0; x < m.GameWorldWidth; x++ {
				m.GameWorldGrid[y][x] = transparentStyle
			}
		}
		return
	}

	// Normal viewport rendering with valid camera position
	for y := 0; y < m.GameWorldHeight; y++ {
		sourceY := cameraY + y
		if sourceY < 0 || sourceY >= 250 {
			// Out of bounds, show transparent
			for x := 0; x < m.GameWorldWidth; x++ {
				m.GameWorldGrid[y][x] = transparentStyle
			}
			continue
		}
		for x := 0; x < m.GameWorldWidth; x++ {
			sourceX := cameraX + x
			if sourceX < 0 || sourceX >= 400 {
				// Out of bounds, show transparent
				m.GameWorldGrid[y][x] = transparentStyle
				continue
			}
			// Render directly from room map value
			roomValue := roomData[sourceY][sourceX]
			m.GameWorldGrid[y][x] = getStyledCharFromRoomValue(roomValue)
		}
	}
}

// getCurrentPlayerRoom returns the room number string where the current player is located
// Returns empty string for walls/hallways, room number string ("1", "2", ...) for rooms
func (m *Model) getCurrentPlayerRoom() string {
	if m.connMgr == nil {
		return ""
	}

	gameState := m.connMgr.GetState()
	if gameState == nil {
		return ""
	}

	currentPlayer, exists := gameState.Players[m.userName]
	if !exists {
		return ""
	}

	playerX, playerY := parsePosition(currentPlayer.Pos)

	// Bounds check
	if playerX < 0 || playerX >= 400 || playerY < 0 || playerY >= 250 {
		return ""
	}

	// Get room value from roomMap
	roomData, err := getRoomMap()
	if err != nil {
		return ""
	}

	roomValue := roomData[playerY][playerX]
	// Return room number if it's a numeric string (room), empty string otherwise
	if roomValue != "" && roomValue != "-1" && roomValue != "r" && roomValue != "o" && roomValue != "i" && roomValue != "e" {
		// Check if it's a valid room number
		if _, err := strconv.Atoi(roomValue); err == nil {
			return roomValue
		}
	}
	return ""
}

// isPlayerInRoom checks if the player is in any room (room number string is not empty)
func (m *Model) isPlayerInRoom() bool {
	return m.getCurrentPlayerRoom() != ""
}

// isPlayerInSpecificRoom checks if the player is in a specific room number
func (m *Model) isPlayerInSpecificRoom(roomNum int) bool {
	roomStr := m.getCurrentPlayerRoom()
	if roomStr == "" {
		return false
	}
	roomNumFromStr, err := strconv.Atoi(roomStr)
	if err != nil {
		return false
	}
	return roomNumFromStr == roomNum
}

// renderPlayerToOverlay renders a single player to the overlay grid
func (m *Model) renderPlayerToOverlay(
	overlay [][]StyledCell,
	player protocol.Player,
	username string,
	cameraX, cameraY int,
	isCurrentPlayer bool,
) {
	// Parse player world position
	playerX, playerY := parsePosition(player.Pos)

	// Convert to viewport coordinates
	vx := playerX - cameraX
	vy := playerY - cameraY

	// Get avatar and split into lines
	avatar := createAvatarFromIndices(player.Avatar)
	avatarLines := strings.Split(avatar.Render(), "\n")

	// Choose styles
	usernameStyle := otherPlayerUsernameStyle
	avatarStyle := otherPlayerAvatarStyle
	if isCurrentPlayer {
		usernameStyle = currentPlayerUsernameStyle
		avatarStyle = currentPlayerAvatarStyle
	}

	// Truncate username to 5 characters
	displayUsername := username
	if len(displayUsername) > 5 {
		displayUsername = displayUsername[:5]
	}

	// Render username (1 line above avatar)
	usernameY := vy - 1
	usernameX := vx - 1 // Center 5-char username above 3-char avatar
	if usernameY >= 0 && usernameY < len(overlay) {
		for i, ch := range displayUsername {
			charX := usernameX + i
			if charX >= 0 && charX < len(overlay[0]) {
				overlay[usernameY][charX].StyledString = usernameStyle.Render(string(ch))
				overlay[usernameY][charX].HasContent = true
			}
		}
	}

	// Render avatar (only torso and legs for top-down view - lines 1 and 2)
	for line := 1; line < 3 && line < len(avatarLines); line++ {
		avatarY := vy + (line - 1) // Line 1 -> vy, Line 2 -> vy+1
		if avatarY < 0 || avatarY >= len(overlay) {
			continue
		}

		avatarLine := avatarLines[line]
		for charIdx := 0; charIdx < len(avatarLine) && charIdx < 3; charIdx++ {
			avatarX := vx + charIdx
			if avatarX < 0 || avatarX >= len(overlay[0]) {
				continue
			}

			styledChar := avatarStyle.Render(string(avatarLine[charIdx]))
			overlay[avatarY][avatarX].StyledString = styledChar
			overlay[avatarY][avatarX].HasContent = true
		}
	}
}

// compositePlayerLayer creates an overlay grid with all players rendered
func (m *Model) compositePlayerLayer(cameraX, cameraY int) [][]StyledCell {
	// Create empty overlay grid
	overlay := make([][]StyledCell, m.GameWorldHeight)
	for i := range overlay {
		overlay[i] = make([]StyledCell, m.GameWorldWidth)
	}

	// Get game state
	if m.connMgr == nil {
		return overlay
	}

	gameState := m.connMgr.GetState()
	if gameState == nil {
		return overlay
	}

	// Render other players first (z-order: back)
	for username, player := range gameState.Players {
		if username == m.userName {
			continue // Skip current player, render last
		}
		m.renderPlayerToOverlay(overlay, player, username, cameraX, cameraY, false)
	}

	// Render current player on top (z-order: front)
	if currentPlayer, exists := gameState.Players[m.userName]; exists {
		m.renderPlayerToOverlay(overlay, currentPlayer, m.userName, cameraX, cameraY, true)
	}

	return overlay
}

// renderGameWorld creates a grid representation of the game world with players
func (m Model) renderGameWorld(width, height int) string {
	var builder strings.Builder

	// Get camera position and player overlay
	mPtr := &m
	cameraX, cameraY := mPtr.calculateViewport()
	playerOverlay := mPtr.compositePlayerLayer(cameraX, cameraY)

	// Render each cell: background + player overlay
	for y := 0; y < height && y < len(m.GameWorldGrid); y++ {
		for x := 0; x < width && x < len(m.GameWorldGrid[y]); x++ {
			// Check if there's a player overlay at this position
			if y < len(playerOverlay) && x < len(playerOverlay[y]) && playerOverlay[y][x].HasContent {
				// Render player overlay
				builder.WriteString(playerOverlay[y][x].StyledString)
			} else if m.GameWorldGrid[y][x] == "" {
				// Render transparent background
				builder.WriteString(transparentStyle)
			} else {
				// Render background
				builder.WriteString(m.GameWorldGrid[y][x])
			}
		}
		if y < height-1 {
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

// renderChatPanel renders the chat panel (right 30%)
func (m Model) renderChatPanel(width, height int) string {
	// Split chat panel vertically: 30% announcements, 70% chat
	announcementHeight := int(float64(height) * 0.3)
	chatHeight := height - announcementHeight - 2 // -2 for borders

	// Render announcement section (top 30%)
	announcementSection := m.renderAnnouncementsSection(width, announcementHeight)

	// Render chat section (bottom 70%)
	chatSection := m.renderChatSection(width, chatHeight)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		announcementSection,
		chatSection,
	)
}

// renderAnnouncementsSection renders the announcements area
func (m Model) renderAnnouncementsSection(width, height int) string {
	// Title
	announcementTitle := lipgloss.NewStyle().
		Foreground(accentColor).
		Bold(true).
		Width(width).
		Align(lipgloss.Center).
		Render("ANNOUNCEMENTS")

	// Announcements content
	var announcementLines []string
	displayCount := height - 3 // Reserve space for title and padding
	if displayCount < 1 {
		displayCount = 1
	}

	// Show most recent announcements
	startIdx := 0
	if len(m.announcements) > displayCount {
		startIdx = len(m.announcements) - displayCount
	}

	for i := startIdx; i < len(m.announcements); i++ {
		announcementLines = append(announcementLines, mutedStyle.Render("• "+m.announcements[i]))
	}

	// If no announcements, show placeholder
	if len(announcementLines) == 0 {
		announcementLines = append(announcementLines, mutedStyle.Render("No announcements"))
	}

	announcementContent := lipgloss.NewStyle().
		Width(width).
		Height(height-1).
		Padding(0, 1).
		Render(strings.Join(announcementLines, "\n"))

	return lipgloss.JoinVertical(
		lipgloss.Left,
		announcementTitle,
		announcementContent,
	)
}

// renderChatSection renders the chat messages area
func (m Model) renderChatSection(width, height int) string {
	// Chat mode indicator
	var modeIndicator string
	if m.chatMode == ChatModeGlobal {
		modeIndicator = highlightStyle.Render("[GLOBAL]") + mutedStyle.Render(" Press 'p' for private")
	} else {
		target := m.chatTarget
		if target == "" {
			target = "none"
		}
		modeIndicator = highlightStyle.Render("[PRIVATE: "+target+"]") + mutedStyle.Render(" Press 'g' for global")
	}

	chatTitle := lipgloss.NewStyle().
		Foreground(accentColor).
		Bold(true).
		Width(width).
		Align(lipgloss.Center).
		Render("CHAT")

	modeBar := lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		Render(modeIndicator)

	// Chat messages
	var messageLines []string
	displayCount := height - 3 // Reserve space for title, mode bar, padding (no input box here)
	if displayCount < 1 {
		displayCount = 1
	}

	// Show most recent messages
	startIdx := 0
	if len(m.chatMessages) > displayCount {
		startIdx = len(m.chatMessages) - displayCount
	}

	for i := startIdx; i < len(m.chatMessages); i++ {
		messageLines = append(messageLines, m.chatMessages[i])
	}

	// If no messages, show placeholder
	if len(messageLines) == 0 {
		messageLines = append(messageLines, mutedStyle.Render("No messages yet. Press 't' to type."))
	}

	messagesContent := lipgloss.NewStyle().
		Width(width).
		Height(displayCount).
		Padding(0, 1).
		Render(strings.Join(messageLines, "\n"))

	return lipgloss.JoinVertical(
		lipgloss.Left,
		chatTitle,
		modeBar,
		messagesContent,
	)
}

// renderChatInputBox renders the chat input box (adapts to width)
func (m Model) renderChatInputBox(width int) string {
	inputPrefix := "> "
	inputText := m.chatInput

	// Always ensure we have content to maintain consistent height
	if m.chatInputActive {
		if inputText == "" {
			inputText = cursorStyle.Render("|")
		} else {
			inputText += cursorStyle.Render("|")
		}
	} else {
		if inputText == "" {
			inputText = mutedStyle.Render("Press 't' to type...")
		}
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(1). // Fixed height to prevent shifting
		Border(lipgloss.RoundedBorder()).
		BorderForeground(mutedColor).
		Render(inputPrefix + inputText)
}

// renderStatusBar renders the bottom status bar
func (m Model) renderStatusBar() string {
	playerInfo := lipgloss.NewStyle().
		Foreground(successColor).
		Bold(true).
		Render("Player: " + m.userName)

	avatarDisplay := lipgloss.NewStyle().
		Foreground(secondaryColor).
		Render(strings.ReplaceAll(m.avatar.Render(), "\n", " "))

	var controls string
	if m.chatInputActive {
		controls = mutedStyle.Render("ENTER: Send  •  ESC: Cancel")
	} else {
		controls = mutedStyle.Render("T: Chat  •  G/P: Mode  •  CTRL+C: Quit")
	}

	return lipgloss.NewStyle().
		Foreground(fgColor).
		Width(m.width).
		Padding(1, 0).
		Align(lipgloss.Center).
		Render(playerInfo + "  " + avatarDisplay + "  •  " + controls)
}
