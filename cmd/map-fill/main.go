package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	// Parse command-line arguments
	if len(os.Args) < 5 {
		fmt.Println("Usage: map-fill <input.txt> <output.txt> <x> <y> <to_replace> <replace_with>")
		fmt.Println("Example: map-fill map.txt map_filled.txt 50 100")
		os.Exit(1)
	}

	inputFile := os.Args[1]
	outputFile := os.Args[2]
	x, err := strconv.Atoi(os.Args[3])
	if err != nil {
		fmt.Printf("Error: invalid X coordinate '%s': %v\n", os.Args[3], err)
		os.Exit(1)
	}
	y, err := strconv.Atoi(os.Args[4])
	if err != nil {
		fmt.Printf("Error: invalid Y coordinate '%s': %v\n", os.Args[4], err)
		os.Exit(1)
	}

	to_replace := ' '
	replace_with := 'b'

	if len(os.Args) >= 6 {
		to_replace = rune(os.Args[5][0])
		replace_with = rune(os.Args[6][0])
	}

	// Read input file
	data, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Printf("Error reading file '%s': %v\n", inputFile, err)
		os.Exit(1)
	}

	// Parse map into 2D array
	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 {
		fmt.Println("Error: empty map file")
		os.Exit(1)
	}

	// Find maximum line length to determine map width
	maxWidth := 0
	for _, line := range lines {
		if len(line) > maxWidth {
			maxWidth = len(line)
		}
	}

	// Create 2D character array (map)
	height := len(lines)
	width := maxWidth
	mapGrid := make([][]rune, height)
	for i := range mapGrid {
		mapGrid[i] = make([]rune, width)
		// Initialize with spaces
		for j := range mapGrid[i] {
			mapGrid[i][j] = ' '
		}
		// Copy line content
		if i < len(lines) {
			for j, char := range lines[i] {
				if j < width {
					mapGrid[i][j] = char
				}
			}
		}
	}

	fmt.Printf("Map loaded: %d rows x %d columns\n", height, width)

	// Validate starting coordinates
	if y < 0 || y >= height || x < 0 || x >= width {
		fmt.Printf("Error: coordinates (%d, %d) are out of bounds (map is %dx%d)\n", x, y, width, height)
		// Print everything out
		for i, row := range mapGrid {
			for _, char := range row {
				fmt.Print(string(char))
			}
			if i < len(mapGrid)-1 {
				fmt.Print("\n")
			}
		}
		os.Exit(1)
	}

	// Perform flood fill
	fillCount := floodFill(mapGrid, x, y, width, height, to_replace, replace_with)

	fmt.Printf("Filled %d cells with 'b'\n", fillCount)

	// Write output file
	var output strings.Builder
	for i, row := range mapGrid {
		for _, char := range row {
			output.WriteRune(char)
		}
		// Don't add newline after last line if original didn't have it
		if i < len(mapGrid)-1 {
			output.WriteRune('\n')
		}
	}

	err = os.WriteFile(outputFile, []byte(output.String()), 0644)
	if err != nil {
		fmt.Printf("Error writing output file '%s': %v\n", outputFile, err)
		os.Exit(1)
	}

	fmt.Printf("Output written to: %s\n", outputFile)
}

// floodFill fills the region starting from (startX, startY) with 'b'
// Stops at border characters: 'o', 'i', 'r', 'e'
func floodFill(mapGrid [][]rune, startX, startY, width, height int, to_replace, replace_with rune) int {
	type point struct {
		x, y int
	}

	// Track visited cells to avoid infinite loops
	visited := make(map[point]bool)
	fillCount := 0

	stack := []point{{startX, startY}}

	for len(stack) > 0 {
		// Pop from stack
		p := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		// Check bounds
		if p.y < 0 || p.y >= height || p.x < 0 || p.x >= width {
			continue
		}

		// Skip if already visited
		if visited[p] {
			continue
		}
		visited[p] = true

		// Get character at this position
		char := mapGrid[p.y][p.x]

		// If it's a border character, don't fill it
		if char != to_replace && char != replace_with {
			continue
		}

		// If it's a space, fill it with 'b'
		if char == to_replace {
			mapGrid[p.y][p.x] = replace_with
			fillCount++

			// Add all 4 neighbors to stack
			stack = append(stack, point{p.x, p.y - 1}) // up
			stack = append(stack, point{p.x, p.y + 1}) // down
			stack = append(stack, point{p.x - 1, p.y}) // left
			stack = append(stack, point{p.x + 1, p.y}) // right
		}
		// If it's any other character (not space, not border), skip it
		// This preserves existing map content
	}

	return fillCount
}
