package connection

import (
	"sync"

	"github.com/yourusername/always-at-morg/internal/protocol"
)

// State manages the current game state
type State struct {
	currentState *protocol.GameState
	mu           sync.RWMutex
}

// NewState creates a new game state manager
func NewState() *State {
	return &State{
		currentState: &protocol.GameState{},
	}
}

// UpdateState updates the entire game state
func (s *State) UpdateState(state *protocol.GameState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentState = state
}

// GetState returns the current game state (thread-safe copy)
func (s *State) GetState() *protocol.GameState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentState
}
