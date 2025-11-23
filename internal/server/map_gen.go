package server

import (
	"fmt"
	"os"
	"strings"
)

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

	// Simply copy all characters from the map file directly
	// No flood fill needed - the map file already has everything marked correctly
	for i := 0; i < 250; i++ {
		for j := 0; j < 400; j++ {
			char := mapChars[i][j]
			// Convert characters to their string representation
			if char == ' ' {
				result[i][j] = " " // Walkable space
			} else {
				result[i][j] = string(char) // r, o, i, e, b, or any other character
			}
		}
	}

	return result, nil
}

