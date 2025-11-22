package ui

import "fmt"

// Avatar presets
var (
	HeadOptions = []string{
		" ō ", // happy face
		" ^ ", // cute face
		" - ", // neutral face
		" ◡ ", // smile
		" ◉ ", // wide eyes
		" ∩ ", // arc
	}

	TorsoOptions = []string{
		"/|\\", // T-pose
		"{+}", // armored
		"<|>", // wide stance
		"[|]", // box body
		"(|)", // rounded
		"\\|/", // Y-pose
	}

	LegOptions = []string{
		"/ \\", // standing
		"| |", // straight legs
		"^ ^", // feet up
		"∧ ∧", // pointed feet
		"⌐ ⌐", // boots
		"◡ ◡", // curved
	}
)

// Avatar represents a 3x3 character avatar
type Avatar struct {
	HeadIndex  int
	TorsoIndex int
	LegsIndex  int
}

// Render returns the 3-line string representation
func (a Avatar) Render() string {
	return fmt.Sprintf("%s\n%s\n%s",
		HeadOptions[a.HeadIndex],
		TorsoOptions[a.TorsoIndex],
		LegOptions[a.LegsIndex])
}

// NewAvatar creates a default avatar
func NewAvatar() Avatar {
	return Avatar{
		HeadIndex:  0,
		TorsoIndex: 0,
		LegsIndex:  0,
	}
}
