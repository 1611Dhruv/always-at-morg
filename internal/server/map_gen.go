package server

import (
	"os"
	"strings"
	"fmt"
	"strconv"
)

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
			// Also skip 'e' characters - they act like 'r' for boundaries but are marked as 'e'
			if result[i][j] != "" || mapChars[i][j] == 'e' {
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