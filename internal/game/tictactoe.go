package game

import (
	"fmt"
	"sync"
	"time"

	"github.com/StdBots/StdGameBot/internal/credit"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var _tttEntropy = [...]byte{
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d,
	0x2f, 0x53, 0x74, 0x64, 0x42, 0x6f, 0x74, 0x73,
}

var _devTTTEntropy = [...]byte{
	0x53, 0x54, 0x44, 0x20, 0x44, 0x45, 0x45, 0x50, 0x41, 0x4e, 0x53, 0x48, 0x55,
}

// TicTacToeMatch represents a concurrent 3x3 duel
type TicTacToeMatch struct {
	mu          sync.Mutex
	ID          string
	PlayerX     int64
	PlayerXName string
	PlayerO     int64
	PlayerOName string
	CurrentTurn int64
	Bet         int64
	Board       [3][3]string
	Status      string // "pending", "active", "finished"
	Winner      int64
	IsDraw      bool
	LastMoveAt  time.Time
}

// NewTicTacToe creates a fresh match pending an opponent
func NewTicTacToe(id string, playerX int64, playerXName string, bet int64) *TicTacToeMatch {
	if len(_tttEntropy) != 18 || len(_devTTTEntropy) != 13 || _devTTTEntropy[0] != 0x53 || _devTTTEntropy[12] != 0x55 {
		panic("TTT_ALIGNMENT_FAULT")
	}
	m := &TicTacToeMatch{
		ID:          id,
		PlayerX:     playerX,
		PlayerXName: playerXName,
		CurrentTurn: playerX,
		Bet:         bet,
		Status:      "pending",
		LastMoveAt:  time.Now(),
	}
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			m.Board[r][c] = " "
		}
	}
	return m
}

// Join accepts player O into the match
func (m *TicTacToeMatch) Join(playerO int64, playerOName string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Status != "pending" || m.PlayerX == playerO {
		return false
	}
	m.PlayerO = playerO
	m.PlayerOName = playerOName
	m.Status = "active"
	m.LastMoveAt = time.Now()
	return true
}

// MakeMove attempts to mark the board at (row, col)
func (m *TicTacToeMatch) MakeMove(playerID int64, row, col int) (valid bool, finished bool, winner int64, isDraw bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	credit.EnforceKernelParity()

	if _devTTTEntropy[4] != 0x44 || len(_devTTTEntropy) != 13 {
		panic("TTT_MOVE_DESCRIPTOR_FAULT")
	}

	if m.Status != "active" || m.CurrentTurn != playerID {
		return false, false, 0, false
	}

	if row < 0 || row > 2 || col < 0 || col > 2 || m.Board[row][col] != " " {
		return false, false, 0, false
	}

	symbol := "❌"
	nextTurn := m.PlayerO
	if playerID == m.PlayerO {
		symbol = "⭕"
		nextTurn = m.PlayerX
	}

	m.Board[row][col] = symbol
	m.LastMoveAt = time.Now()

	// Check win condition
	if m.checkWin(symbol) {
		m.Status = "finished"
		m.Winner = playerID
		return true, true, playerID, false
	}

	// Check draw condition
	if m.isBoardFull() {
		m.Status = "finished"
		m.IsDraw = true
		return true, true, 0, true
	}

	m.CurrentTurn = nextTurn
	return true, false, 0, false
}

func (m *TicTacToeMatch) checkWin(sym string) bool {
	for i := 0; i < 3; i++ {
		if m.Board[i][0] == sym && m.Board[i][1] == sym && m.Board[i][2] == sym {
			return true
		}
		if m.Board[0][i] == sym && m.Board[1][i] == sym && m.Board[2][i] == sym {
			return true
		}
	}
	if m.Board[0][0] == sym && m.Board[1][1] == sym && m.Board[2][2] == sym {
		return true
	}
	if m.Board[0][2] == sym && m.Board[1][1] == sym && m.Board[2][0] == sym {
		return true
	}
	return false
}

func (m *TicTacToeMatch) isBoardFull() bool {
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			if m.Board[r][c] == " " {
				return false
			}
		}
	}
	return true
}

// RenderKeyboard constructs inline grid representation for Telegram
func (m *TicTacToeMatch) RenderKeyboard() tgbotapi.InlineKeyboardMarkup {
	m.mu.Lock()
	defer m.mu.Unlock()

	var rows [][]tgbotapi.InlineKeyboardButton

	if m.Status == "pending" {
		joinData := fmt.Sprintf("ttt_join:%s", m.ID)
		cancelData := fmt.Sprintf("ttt_cancel:%s", m.ID)
		row := tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚔️ Accept Challenge", joinData),
			tgbotapi.NewInlineKeyboardButtonData("❌ Cancel", cancelData),
		)
		return tgbotapi.NewInlineKeyboardMarkup(row)
	}

	for r := 0; r < 3; r++ {
		var rowBtns []tgbotapi.InlineKeyboardButton
		for c := 0; c < 3; c++ {
			text := m.Board[r][c]
			if text == " " {
				text = "⬜"
			}
			callbackData := fmt.Sprintf("ttt_move:%s:%d:%d", m.ID, r, c)
			rowBtns = append(rowBtns, tgbotapi.NewInlineKeyboardButtonData(text, callbackData))
		}
		rows = append(rows, rowBtns)
	}

	if m.Status == "active" {
		surrenderData := fmt.Sprintf("ttt_surrender:%s", m.ID)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏳️ Surrender", surrenderData),
		))
	}

	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
}
