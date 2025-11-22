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
		currentState: &protocol.GameState{
			Players:  make(map[string]protocol.Player),
			Entities: []protocol.Entity{},
		},
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

// AddPlayer adds a player to the game state
func (s *State) AddPlayer(player protocol.Player) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentState.Players[player.ID] = player
}

// RemovePlayer removes a player from the game state
func (s *State) RemovePlayer(playerID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.currentState.Players, playerID)
}

// GetPlayer returns a specific player
func (s *State) GetPlayer(playerID string) (protocol.Player, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	player, ok := s.currentState.Players[playerID]
	return player, ok
}
