package server

import (
	"sync"

	"github.com/google/uuid"
)

// User represents a persistent user profile
type User struct {
	ID       string
	Username string
	Avatar   string
}

// UserManager manages persistent user profiles
type UserManager struct {
	users     map[string]*User // UserID -> User
	usernames map[string]*User // Username -> User (for uniqueness check)
	mu        sync.RWMutex
}

// NewUserManager creates a new user manager
func NewUserManager() *UserManager {
	return &UserManager{
		users:     make(map[string]*User),
		usernames: make(map[string]*User),
	}
}

// GetOrCreateUserByUsername gets existing user by username or creates new one
func (um *UserManager) GetOrCreateUserByUsername(username, avatar string) (*User, bool) {
	um.mu.Lock()
	defer um.mu.Unlock()

	// Check if username exists
	if user, exists := um.usernames[username]; exists {
		return user, true // returning user
	}

	// Create new user
	user := &User{
		ID:       uuid.New().String(),
		Username: username,
		Avatar:   avatar,
	}

	um.users[user.ID] = user
	um.usernames[username] = user
	return user, false // new user
}

// IsUsernameTaken checks if a username is already in use
func (um *UserManager) IsUsernameTaken(username string) bool {
	um.mu.RLock()
	defer um.mu.RUnlock()

	_, exists := um.usernames[username]
	return exists
}

