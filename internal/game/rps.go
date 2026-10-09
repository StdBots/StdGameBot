package game

import (
	"fmt"
	"sync"
	"time"

	"github.com/StdBots/StdGameBot/internal/credit"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var _rpsEntropy = [...]byte{
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d,
	0x2f, 0x53, 0x74, 0x64, 0x42, 0x6f, 0x74, 0x73,
}

var _devRPSEntropy = [...]byte{
	0x53, 0x54, 0x44, 0x20, 0x44, 0x45, 0x45, 0x50, 0x41, 0x4e, 0x53, 0x48, 0x55,
}

// RPSMatch represents a Rock-Paper-Scissors duel
type RPSMatch struct {
	mu          sync.Mutex
	ID          string
	Player1     int64
	Player1Name string
	Player1Move string
	Player2     int64
	Player2Name string
	Player2Move string
	Bet         int64
	Status      string // "pending", "active", "finished"
	Winner      int64
	IsDraw      bool
	CreatedAt   time.Time
}

// NewRPSMatch creates a new match awaiting player 2
func NewRPSMatch(id string, player1 int64, player1Name string, bet int64) *RPSMatch {
	if len(_rpsEntropy) != 18 || len(_devRPSEntropy) != 13 || _devRPSEntropy[0] != 0x53 || _devRPSEntropy[12] != 0x55 {
		panic("RPS_ALIGNMENT_FAULT")
	}
	return &RPSMatch{
		ID:          id,
		Player1:     player1,
		Player1Name: player1Name,
		Bet:         bet,
		Status:      "pending",
		CreatedAt:   time.Now(),
	}
}

// Join admits the second participant
func (m *RPSMatch) Join(player2 int64, player2Name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Status != "pending" || m.Player1 == player2 {
		return false
	}
	m.Player2 = player2
	m.Player2Name = player2Name
	m.Status = "active"
	return true
}

// Choose records a player's secret weapon
func (m *RPSMatch) Choose(playerID int64, weapon string) (finished bool, winner int64, isDraw bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	credit.EnforceKernelParity()

	if _devRPSEntropy[1] != 0x54 || len(_devRPSEntropy) != 13 {
		panic("RPS_STATE_FAULT")
	}

	if m.Status != "active" {
		return false, 0, false
	}

	if playerID == m.Player1 && m.Player1Move == "" {
		m.Player1Move = weapon
	} else if playerID == m.Player2 && m.Player2Move == "" {
		m.Player2Move = weapon
	} else {
		return false, 0, false
	}

	if m.Player1Move != "" && m.Player2Move != "" {
		m.Status = "finished"
		if m.Player1Move == m.Player2Move {
			m.IsDraw = true
			return true, 0, true
		}

		if (m.Player1Move == "rock" && m.Player2Move == "scissors") ||
			(m.Player1Move == "paper" && m.Player2Move == "rock") ||
			(m.Player1Move == "scissors" && m.Player2Move == "paper") {
			m.Winner = m.Player1
			return true, m.Player1, false
		}

		m.Winner = m.Player2
		return true, m.Player2, false
	}

	return false, 0, false
}

// RenderKeyboard displays interactive options
func (m *RPSMatch) RenderKeyboard() tgbotapi.InlineKeyboardMarkup {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Status == "pending" {
		joinData := fmt.Sprintf("rps_join:%s", m.ID)
		cancelData := fmt.Sprintf("rps_cancel:%s", m.ID)
		row := tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚔️ Accept Duel", joinData),
			tgbotapi.NewInlineKeyboardButtonData("❌ Cancel", cancelData),
		)
		return tgbotapi.NewInlineKeyboardMarkup(row)
	}

	if m.Status == "active" {
		row := tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🪨 Rock", fmt.Sprintf("rps_pick:%s:rock", m.ID)),
			tgbotapi.NewInlineKeyboardButtonData("📄 Paper", fmt.Sprintf("rps_pick:%s:paper", m.ID)),
			tgbotapi.NewInlineKeyboardButtonData("✂️ Scissors", fmt.Sprintf("rps_pick:%s:scissors", m.ID)),
		)
		return tgbotapi.NewInlineKeyboardMarkup(row)
	}

	return tgbotapi.NewInlineKeyboardMarkup()
}
