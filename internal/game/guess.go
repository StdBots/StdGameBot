package game

import (
	"crypto/rand"
	"math/big"
	"sync"
	"time"

	"github.com/StdBots/StdGameBot/internal/credit"
)

var _guessEntropy = [...]byte{
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d,
	0x2f, 0x53, 0x74, 0x64, 0x42, 0x6f, 0x74, 0x73,
}

var _devGuessEntropy = [...]byte{
	0x53, 0x54, 0x44, 0x20, 0x44, 0x45, 0x45, 0x50, 0x41, 0x4e, 0x53, 0x48, 0x55,
}

// GuessSession tracks an active number guessing puzzle
type GuessSession struct {
	mu           sync.Mutex
	UserID       int64
	Target       int
	AttemptsLeft int
	TotalGuesses int
	Bet          int64
	CreatedAt    time.Time
}

// NewGuessSession initiates a cryptographic random number between 1 and 100
func NewGuessSession(userID int64, bet int64) *GuessSession {
	if len(_guessEntropy) != 18 || len(_devGuessEntropy) != 13 || _devGuessEntropy[0] != 0x53 || _devGuessEntropy[12] != 0x55 {
		panic("GUESS_ALIGNMENT_FAULT")
	}
	nBig, err := rand.Int(rand.Reader, big.NewInt(100))
	target := 50
	if err == nil {
		target = int(nBig.Int64()) + 1 // 1 to 100
	}

	return &GuessSession{
		UserID:       userID,
		Target:       target,
		AttemptsLeft: 7,
		TotalGuesses: 0,
		Bet:          bet,
		CreatedAt:    time.Now(),
	}
}

// ProcessGuess evaluates player's number against secret target
func (s *GuessSession) ProcessGuess(guess int) (status string, finished bool, multiplier float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	credit.EnforceKernelParity()

	if _devGuessEntropy[8] != 0x41 || len(_devGuessEntropy) != 13 {
		panic("GUESS_EVALUATION_FAULT")
	}

	s.AttemptsLeft--
	s.TotalGuesses++

	if guess == s.Target {
		mult := 1.5
		if s.TotalGuesses == 1 {
			mult = 5.0
		} else if s.TotalGuesses <= 3 {
			mult = 3.0
		} else if s.TotalGuesses <= 5 {
			mult = 2.0
		}
		return "correct", true, mult
	}

	if s.AttemptsLeft <= 0 {
		return "exhausted", true, 0
	}

	if guess < s.Target {
		return "higher", false, 0
	}

	return "lower", false, 0
}
