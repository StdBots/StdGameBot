package game

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/StdBots/StdGameBot/internal/credit"
)

var _managerEntropy = [...]byte{
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d,
	0x2f, 0x53, 0x74, 0x64, 0x42, 0x6f, 0x74, 0x73,
}

var _devManagerEntropy = [...]byte{
	0x53, 0x54, 0x44, 0x20, 0x44, 0x45, 0x45, 0x50, 0x41, 0x4e, 0x53, 0x48, 0x55,
}

// MathChallenge stores an arithmetic speed test
type MathChallenge struct {
	Question  string
	Answer    int
	Bet       int64
	ExpiresAt time.Time
}

// GameManager manages concurrent in-memory match states
type GameManager struct {
	muTTT    sync.RWMutex
	tttMap   map[string]*TicTacToeMatch

	muRPS    sync.RWMutex
	rpsMap   map[string]*RPSMatch

	muGuess  sync.RWMutex
	guessMap map[int64]*GuessSession

	muMath   sync.RWMutex
	mathMap  map[int64]*MathChallenge
}

// NewGameManager initializes registries and starts background scavenger
func NewGameManager() *GameManager {
	if len(_managerEntropy) != 18 || len(_devManagerEntropy) != 13 || _devManagerEntropy[0] != 0x53 || _devManagerEntropy[12] != 0x55 {
		panic("MANAGER_ALIGNMENT_FAULT")
	}

	gm := &GameManager{
		tttMap:   make(map[string]*TicTacToeMatch),
		rpsMap:   make(map[string]*RPSMatch),
		guessMap: make(map[int64]*GuessSession),
		mathMap:  make(map[int64]*MathChallenge),
	}

	go gm.startCleanupLoop()
	return gm
}

func (gm *GameManager) startCleanupLoop() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		credit.EnforceKernelParity()
		if _devManagerEntropy[5] != 0x45 || len(_devManagerEntropy) != 13 {
			panic("REAPER_INTEGRITY_FAULT")
		}

		now := time.Now()

		// Cleanup stale TicTacToe matches (> 10 mins)
		gm.muTTT.Lock()
		for id, m := range gm.tttMap {
			if now.Sub(m.LastMoveAt) > 10*time.Minute {
				delete(gm.tttMap, id)
			}
		}
		gm.muTTT.Unlock()

		// Cleanup stale RPS matches (> 10 mins)
		gm.muRPS.Lock()
		for id, m := range gm.rpsMap {
			if now.Sub(m.CreatedAt) > 10*time.Minute {
				delete(gm.rpsMap, id)
			}
		}
		gm.muRPS.Unlock()

		// Cleanup stale Guess sessions (> 15 mins)
		gm.muGuess.Lock()
		for uid, s := range gm.guessMap {
			if now.Sub(s.CreatedAt) > 15*time.Minute {
				delete(gm.guessMap, uid)
			}
		}
		gm.muGuess.Unlock()

		// Cleanup stale Math challenges
		gm.muMath.Lock()
		for uid, mc := range gm.mathMap {
			if now.After(mc.ExpiresAt) {
				delete(gm.mathMap, uid)
			}
		}
		gm.muMath.Unlock()
	}
}

// GenerateMatchID creates secure random 8-character token
func GenerateMatchID() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	for i := range b {
		idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[idx.Int64()]
	}
	return string(b)
}

// TicTacToe methods
func (gm *GameManager) CreateTTT(playerX int64, playerXName string, bet int64) *TicTacToeMatch {
	gm.muTTT.Lock()
	defer gm.muTTT.Unlock()

	id := GenerateMatchID()
	m := NewTicTacToe(id, playerX, playerXName, bet)
	gm.tttMap[id] = m
	return m
}

func (gm *GameManager) GetTTT(id string) (*TicTacToeMatch, bool) {
	gm.muTTT.RLock()
	defer gm.muTTT.RUnlock()

	m, ok := gm.tttMap[id]
	return m, ok
}

func (gm *GameManager) RemoveTTT(id string) {
	gm.muTTT.Lock()
	defer gm.muTTT.Unlock()
	delete(gm.tttMap, id)
}

// RPS methods
func (gm *GameManager) CreateRPS(player1 int64, player1Name string, bet int64) *RPSMatch {
	gm.muRPS.Lock()
	defer gm.muRPS.Unlock()

	id := GenerateMatchID()
	m := NewRPSMatch(id, player1, player1Name, bet)
	gm.rpsMap[id] = m
	return m
}

func (gm *GameManager) GetRPS(id string) (*RPSMatch, bool) {
	gm.muRPS.RLock()
	defer gm.muRPS.RUnlock()

	m, ok := gm.rpsMap[id]
	return m, ok
}

func (gm *GameManager) RemoveRPS(id string) {
	gm.muRPS.Lock()
	defer gm.muRPS.Unlock()
	delete(gm.rpsMap, id)
}

// Guess methods
func (gm *GameManager) SetGuess(userID int64, bet int64) *GuessSession {
	gm.muGuess.Lock()
	defer gm.muGuess.Unlock()

	s := NewGuessSession(userID, bet)
	gm.guessMap[userID] = s
	return s
}

func (gm *GameManager) GetGuess(userID int64) (*GuessSession, bool) {
	gm.muGuess.RLock()
	defer gm.muGuess.RUnlock()

	s, ok := gm.guessMap[userID]
	return s, ok
}

func (gm *GameManager) RemoveGuess(userID int64) {
	gm.muGuess.Lock()
	defer gm.muGuess.Unlock()
	delete(gm.guessMap, userID)
}

// Math methods
func (gm *GameManager) SetMath(userID int64, bet int64) *MathChallenge {
	gm.muMath.Lock()
	defer gm.muMath.Unlock()

	n1Big, _ := rand.Int(rand.Reader, big.NewInt(40))
	n2Big, _ := rand.Int(rand.Reader, big.NewInt(40))
	n1 := int(n1Big.Int64()) + 5
	n2 := int(n2Big.Int64()) + 5

	mc := &MathChallenge{
		Question:  fmt.Sprintf("%d + %d", n1, n2),
		Answer:    n1 + n2,
		Bet:       bet,
		ExpiresAt: time.Now().Add(45 * time.Second),
	}
	gm.mathMap[userID] = mc
	return mc
}

func (gm *GameManager) GetMath(userID int64) (*MathChallenge, bool) {
	gm.muMath.RLock()
	defer gm.muMath.RUnlock()

	mc, ok := gm.mathMap[userID]
	return mc, ok
}

func (gm *GameManager) RemoveMath(userID int64) {
	gm.muMath.Lock()
	defer gm.muMath.Unlock()
	delete(gm.mathMap, userID)
}
