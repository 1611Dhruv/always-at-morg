package server

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/yourusername/always-at-morg/internal/protocol"
)

// Dummy variable to satisfy legacy references if any remain
var TreasureClues = []Clue{}

type Clue struct {
	Question string
	Answer   string
}

// TreasureHuntManager handles the game state
type TreasureHuntManager struct {
	mu             sync.RWMutex
	currentRiddle  *GeminiRiddle
	currentRound   int
	isSolved       bool
	winner         string
	showHint       bool
	waitingForNext bool // Prevents ticker from skipping the "Solved" screen
	gameOver       bool // Tracks if the daily limit is reached
	announcements  []protocol.AnnouncementPayload
	updateCallback func(protocol.TreasureHuntStatePayload)
	resetCh        chan struct{} // Channel to trigger immediate next round
}

// Initialize with a default riddle so clients never see "Loading..."
var Manager = &TreasureHuntManager{
	currentRound: 1,
	currentRiddle: &GeminiRiddle{
		Question: "I have keys but no locks. I have a space but no room. You can enter, but never leave. What am I?",
		Answer:   "keyboard",
		Hint:     "I am an input device.",
	},
}

// SetUpdateCallback sets the function to call when state changes
func (tm *TreasureHuntManager) SetUpdateCallback(callback func(protocol.TreasureHuntStatePayload)) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.updateCallback = callback
	
	// IMMEDIATELY trigger the callback with the current state so the client gets data right away
	if tm.currentRiddle != nil || tm.gameOver {
		go callback(tm.getStateLocked())
	}
}

// StartGameLoop begins the 1-minute cycle
func (tm *TreasureHuntManager) StartGameLoop() {
	// Prevent multiple loops if called twice
	if tm.resetCh != nil {
		return
	}
	
	// Buffered channel to ensure we don't miss the signal
	tm.resetCh = make(chan struct{}, 1)
	
	// Ensure we have a riddle to start with
	if tm.currentRiddle == nil && !tm.gameOver {
		tm.nextRiddle()
	}

	ticker := time.NewTicker(1 * time.Minute)
	hintTimer := time.NewTicker(30 * time.Second)

	go func() {
		for {
			select {
			case <-ticker.C:
				tm.mu.RLock()
				waiting := tm.waitingForNext
				isOver := tm.gameOver
				tm.mu.RUnlock()

				if isOver {
					continue
				}

				// Only advance if we aren't waiting for the 5s post-win timer
				if !waiting {
					tm.nextRiddle()
					hintTimer.Reset(30 * time.Second)
				}

			case <-tm.resetCh:
				// Forced update (e.g. 5s after someone won)
				tm.nextRiddle()
				// Reset the main timer so we get a full minute for the new riddle
				ticker.Reset(1 * time.Minute)
				hintTimer.Reset(30 * time.Second)

			case <-hintTimer.C:
				tm.revealHint()
			}
		}
	}()
}

func (tm *TreasureHuntManager) nextRiddle() {
	tm.mu.Lock()
	// Check if we've reached the daily limit (3 questions)
	// If currentRound is 3 and it was just solved, we are moving to 4, which is Game Over.
	if tm.currentRound > 3 {
		tm.gameOver = true
		tm.currentRiddle = nil
		tm.isSolved = true
		tm.winner = ""
		tm.announcements = nil // Clear announcements
		
		state := tm.getStateLocked()
		callback := tm.updateCallback
		tm.mu.Unlock()

		if callback != nil {
			log.Println("Daily limit reached, ending game loop.")
			callback(state)
		}
		return
	}
	tm.mu.Unlock()

	// Generate outside lock to prevent blocking
	riddle, err := GenerateRiddle()
	if err != nil {
		log.Printf("Failed to generate riddle: %v", err)
		// Fallback to a simple CS one if API fails
		riddle = &GeminiRiddle{
			Question: "I have keys but no locks. I have a space but no room. You can enter, but never leave. What am I?",
			Answer:   "keyboard",
			Hint:     "Input device...",
		}
	}

	tm.mu.Lock()
	
	// If previous wasn't solved, maybe announce the answer?
	if !tm.isSolved && tm.currentRiddle != nil {
		tm.addAnnouncement(fmt.Sprintf("Time's up! The answer was: %s", tm.currentRiddle.Answer))
	}

	tm.currentRiddle = riddle
	tm.currentRound++
	tm.isSolved = false
	tm.winner = ""
	tm.showHint = false
	tm.waitingForNext = false // Reset flag so ticker can work again
	
	log.Printf("New Riddle: %s (Ans: %s)", riddle.Question, riddle.Answer)
	
	// Prepare state for broadcast
	state := tm.getStateLocked()
	callback := tm.updateCallback
	tm.mu.Unlock()

	// Broadcast update
	if callback != nil {
		log.Printf("Broadcasting new riddle state (Round %d)", state.CurrentClueIndex)
		callback(state)
	}
}

func (tm *TreasureHuntManager) revealHint() {
	tm.mu.Lock()
	if !tm.isSolved && !tm.waitingForNext && !tm.gameOver {
		tm.showHint = true
		state := tm.getStateLocked()
		callback := tm.updateCallback
		tm.mu.Unlock()
		
		if callback != nil {
			callback(state)
		}
		return
	}
	tm.mu.Unlock()
}

// CheckGuess validates a guess and updates state if correct
func (tm *TreasureHuntManager) CheckGuess(username, guess string) bool {
	tm.mu.Lock()
	// We do NOT defer unlock here because we want to unlock before calling the callback

	if tm.isSolved || tm.currentRiddle == nil || tm.waitingForNext || tm.gameOver {
		tm.mu.Unlock()
		return false
	}

	cleanGuess := strings.TrimSpace(guess)
	cleanAnswer := strings.TrimSpace(tm.currentRiddle.Answer)

	if strings.EqualFold(cleanGuess, cleanAnswer) {
		tm.isSolved = true
		tm.winner = username
		tm.waitingForNext = true // Block the main ticker from skipping the win screen
		tm.addAnnouncement(fmt.Sprintf("🏆 WINNER: %s guessed '%s' correctly!", username, cleanAnswer))
		
		// Capture state and callback while locked
		state := tm.getStateLocked()
		callback := tm.updateCallback
		tm.mu.Unlock() // Unlock BEFORE callback to ensure ordering

		// Notify clients of the win immediately and SYNCHRONOUSLY
		if callback != nil {
			log.Printf("Broadcasting WINNER state for %s", username)
			callback(state)
		}

		// Schedule the next round to start in 5 seconds
		time.AfterFunc(5*time.Second, func() {
			// Non-blocking send in case loop isn't running
			select {
			case tm.resetCh <- struct{}{}:
			default:
			}
		})

		return true
	}
	
	tm.mu.Unlock()
	return false
}

func (tm *TreasureHuntManager) addAnnouncement(msg string) {
	tm.announcements = append(tm.announcements, protocol.AnnouncementPayload{
		Message:   msg,
		Timestamp: time.Now().Unix(),
	})
}

// GetState returns the current state for the client
func (tm *TreasureHuntManager) GetState() protocol.TreasureHuntStatePayload {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.getStateLocked()
}

// Helper to get state while holding lock
func (tm *TreasureHuntManager) getStateLocked() protocol.TreasureHuntStatePayload {
	// Check for Game Over state
	if tm.gameOver {
		return protocol.TreasureHuntStatePayload{
			CurrentClueIndex: tm.currentRound,
			ClueText:         "🎉 Daily Limit Reached! 🎉\n\nYou've completed today's Computer Science challenges.\nCheck back later!",
			Completed:        true,
		}
	}

	if tm.currentRiddle == nil {
		return protocol.TreasureHuntStatePayload{ClueText: "Loading..."}
	}

	text := tm.currentRiddle.Question
	if tm.showHint && !tm.isSolved {
		text += fmt.Sprintf("\n\n💡 HINT: %s", tm.currentRiddle.Hint)
	}
	if tm.isSolved {
		text = fmt.Sprintf("✅ SOLVED by %s!\nAnswer: %s\n\nNext question coming soon...", tm.winner, tm.currentRiddle.Answer)
	}

	return protocol.TreasureHuntStatePayload{
		CurrentClueIndex: tm.currentRound,
		ClueText:         text,
		Completed:        tm.isSolved,
	}
}

// PopAnnouncements returns new announcements and clears the queue
func (tm *TreasureHuntManager) PopAnnouncements() []protocol.AnnouncementPayload {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	
	if len(tm.announcements) == 0 {
		return nil
	}
	
	msgs := tm.announcements
	tm.announcements = nil // Clear queue
	return msgs
}

// CheckTreasureHuntAnswer checks the answer.
func CheckTreasureHuntAnswer(username, guess string) bool {
	return Manager.CheckGuess(username, guess)
}

// GetClueText returns the text for the current step.
func GetClueText(step int) string {
	return Manager.GetState().ClueText
}
