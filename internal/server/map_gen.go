package server

import (
	"os"
	"strings"
	"fmt"
)

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