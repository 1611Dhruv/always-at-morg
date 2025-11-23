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
	nextRiddle     *GeminiRiddle // Pre-fetched next riddle during cooldown
	currentRound   int
	isSolved       bool
	winner         string
	showHint       bool
	inCooldown     bool // True during 2-minute cooldown period
	waitingForNext bool // Prevents ticker from skipping the "Solved" screen
	gameOver       bool // Tracks if the daily limit is reached
	announcements  []protocol.AnnouncementPayload
	updateCallback func(protocol.TreasureHuntStatePayload)
	startNextCh    chan struct{} // Channel to signal next round is ready
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

// StartGameLoop begins the game cycle: 1 min round + 2 min cooldown
func (tm *TreasureHuntManager) StartGameLoop() {
	// Prevent multiple loops if called twice
	if tm.startNextCh != nil {
		return
	}

	// Buffered channel to signal when next round is ready
	tm.startNextCh = make(chan struct{}, 1)

	// Ensure we have a riddle to start with
	if tm.currentRiddle == nil && !tm.gameOver {
		tm.loadNextRiddle()
	}

	roundTimer := time.NewTicker(1 * time.Minute)   // 1 minute active round
	hintTimer := time.NewTicker(30 * time.Second)   // Hint at 30 seconds (halfway)

	go func() {
		for {
			select {
			case <-roundTimer.C:
				tm.mu.RLock()
				waiting := tm.waitingForNext
				isOver := tm.gameOver
				inCooldown := tm.inCooldown
				tm.mu.RUnlock()

				if isOver {
					continue
				}

				// If we're in cooldown, don't do anything (waiting for next riddle)
				if inCooldown {
					continue
				}

				// Only start cooldown if we aren't waiting for the 5s post-win timer
				if !waiting {
					tm.startCooldown()
					hintTimer.Stop() // Stop hint timer during cooldown
				}

			case <-tm.startNextCh:
				// Next riddle is ready! Start the new round
				tm.activateNextRound()
				roundTimer.Reset(1 * time.Minute)
				hintTimer.Reset(30 * time.Second)

			case <-hintTimer.C:
				tm.revealHint()
			}
		}
	}()
}

// startCooldown begins the 2-minute cooldown and fetches next riddle
func (tm *TreasureHuntManager) startCooldown() {
	tm.mu.Lock()

	// Check if we've reached the daily limit (3 questions)
	if tm.currentRound >= 3 {
		tm.gameOver = true
		tm.currentRiddle = nil
		tm.isSolved = true
		tm.winner = ""
		tm.announcements = nil
		tm.inCooldown = false

		state := tm.getStateLocked()
		callback := tm.updateCallback
		tm.mu.Unlock()

		if callback != nil {
			log.Println("Daily limit reached, ending game loop.")
			callback(state)
		}
		return
	}

	// If previous wasn't solved, announce the answer
	if !tm.isSolved && tm.currentRiddle != nil {
		tm.addAnnouncement(fmt.Sprintf("Time's up! The answer was: %s", tm.currentRiddle.Answer))
	}

	tm.inCooldown = true
	tm.waitingForNext = false

	// Show cooldown message to clients
	state := tm.getStateLocked()
	callback := tm.updateCallback
	tm.mu.Unlock()

	if callback != nil {
		log.Println("Broadcasting cooldown state...")
		callback(state)
	}

	// Start fetching next riddle in background (2 minute cooldown)
	go func() {
		log.Println("Starting 2-minute cooldown, fetching next riddle from Gemini...")

		// Generate riddle (this may take a few seconds)
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

		// Wait for the remainder of 2 minutes after fetching
		// (If fetch took 5 seconds, wait 115 more seconds)
		time.Sleep(2 * time.Minute)

		tm.mu.Lock()
		tm.nextRiddle = riddle
		tm.mu.Unlock()

		log.Printf("Riddle ready: %s (Ans: %s)", riddle.Question, riddle.Answer)
		log.Println("Cooldown complete, signaling next round...")

		// Signal that next round is ready
		select {
		case tm.startNextCh <- struct{}{}:
		default:
		}
	}()
}

// activateNextRound switches to the pre-fetched next riddle
func (tm *TreasureHuntManager) activateNextRound() {
	tm.mu.Lock()

	if tm.nextRiddle == nil {
		log.Println("WARNING: activateNextRound called but nextRiddle is nil!")
		tm.mu.Unlock()
		return
	}

	tm.currentRiddle = tm.nextRiddle
	tm.nextRiddle = nil
	tm.currentRound++
	tm.isSolved = false
	tm.winner = ""
	tm.showHint = false
	tm.inCooldown = false

	log.Printf("New Round %d: %s (Ans: %s)", tm.currentRound, tm.currentRiddle.Question, tm.currentRiddle.Answer)

	state := tm.getStateLocked()
	callback := tm.updateCallback
	tm.mu.Unlock()

	if callback != nil {
		log.Printf("Broadcasting new riddle state (Round %d)", state.CurrentClueIndex)
		callback(state)
	}
}

// loadNextRiddle is used for initial setup only
func (tm *TreasureHuntManager) loadNextRiddle() {
	riddle, err := GenerateRiddle()
	if err != nil {
		log.Printf("Failed to generate initial riddle: %v", err)
		riddle = &GeminiRiddle{
			Question: "I have keys but no locks. I have a space but no room. You can enter, but never leave. What am I?",
			Answer:   "keyboard",
			Hint:     "Input device...",
		}
	}

	tm.mu.Lock()
	tm.currentRiddle = riddle
	tm.currentRound = 1
	tm.mu.Unlock()

	log.Printf("Initial Riddle: %s (Ans: %s)", riddle.Question, riddle.Answer)
}

func (tm *TreasureHuntManager) revealHint() {
	tm.mu.Lock()
	if !tm.isSolved && !tm.waitingForNext && !tm.gameOver && !tm.inCooldown {
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

	if tm.isSolved || tm.currentRiddle == nil || tm.waitingForNext || tm.gameOver || tm.inCooldown {
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

		// Wait 5 seconds to show win screen, then start cooldown
		time.AfterFunc(5*time.Second, func() {
			log.Println("Win screen timeout, starting cooldown...")
			tm.startCooldown()
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

	// Check for cooldown state
	if tm.inCooldown {
		return protocol.TreasureHuntStatePayload{
			CurrentClueIndex: tm.currentRound,
			ClueText:         "⏳ Cooldown Period ⏳\n\nPreparing next riddle...\nTake a break, next question coming in ~2 minutes!",
			Completed:        false,
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
