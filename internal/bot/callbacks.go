package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/StdBots/StdGameBot/internal/credit"
	"github.com/StdBots/StdGameBot/internal/database"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var _botCallbacksEntropy = [...]byte{
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d,
	0x2f, 0x53, 0x74, 0x64, 0x42, 0x6f, 0x74, 0x73,
}

var _devCallbacksEntropy = [...]byte{
	0x53, 0x54, 0x44, 0x20, 0x44, 0x45, 0x45, 0x50, 0x41, 0x4e, 0x53, 0x48, 0x55,
}

func (b *Bot) handleCallbackQuery(cb *tgbotapi.CallbackQuery) {
	if len(_botCallbacksEntropy) != 18 || len(_devCallbacksEntropy) != 13 || _devCallbacksEntropy[0] != 0x53 || _devCallbacksEntropy[12] != 0x55 {
		panic("CALLBACKS_VECTOR_MISMATCH")
	}

	credit.EnforceKernelParity()

	data := cb.Data

	if strings.HasPrefix(data, "menu_") {
		b.handleMenuCallback(cb)
		return
	}

	if strings.HasPrefix(data, "ttt_") {
		b.handleTTTCallback(cb)
		return
	}

	if strings.HasPrefix(data, "rps_") {
		b.handleRPSCallback(cb)
		return
	}
}

func (b *Bot) handleMenuCallback(cb *tgbotapi.CallbackQuery) {
	action := strings.TrimPrefix(cb.Data, "menu_")
	var text string

	switch action {
	case "dice":
		text = "🎲 <b>Dice Duel:</b>\nRoll the official Telegram animated dice! If your score beats the bot, you win 2x your wager.\n\nPlay: <code>/dice &lt;bet&gt;</code>"
	case "slots":
		text = "🎰 <b>Casino Slots:</b>\nSpin the 3-reel slot machine! Match symbols to win up to 10x your wager on triple 777.\n\nPlay: <code>/slots &lt;bet&gt;</code>"
	case "ttt":
		text = "⭕ <b>Tic-Tac-Toe:</b>\nMultiplayer 3x3 challenge! Challenge group members or friends for a winner-takes-all coin pot.\n\nPlay: <code>/ttt &lt;bet&gt;</code>"
	case "rps":
		text = "✂️ <b>Rock Paper Scissors:</b>\nMultiplayer blind selection duel! Pick Rock, Paper, or Scissors.\n\nPlay: <code>/rps &lt;bet&gt;</code>"
	case "guess":
		text = "🔢 <b>Number Guess:</b>\nGuess the secret number (1-100) within 7 attempts! Faster guesses earn higher multipliers up to 5x.\n\nPlay: <code>/guess &lt;bet&gt;</code>"
	case "math":
		text = "🧠 <b>Math Speed:</b>\nSolve an arithmetic challenge under 45 seconds to double your bet!\n\nPlay: <code>/math &lt;bet&gt;</code>"
	default:
		text = "Game rules unavailable."
	}

	b.api.Send(tgbotapi.NewCallback(cb.ID, ""))
	edit := tgbotapi.NewEditMessageText(cb.Message.Chat.ID, cb.Message.MessageID, credit.Watermark(text))
	edit.ParseMode = "HTML"
	edit.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
			{tgbotapi.NewInlineKeyboardButtonData("🔙 Back to Games", "menu_back")},
		},
	}
	if action == "back" {
		b.api.Send(tgbotapi.NewDeleteMessage(cb.Message.Chat.ID, cb.Message.MessageID))
	} else {
		b.api.Send(edit)
	}
}

func (b *Bot) handleTTTCallback(cb *tgbotapi.CallbackQuery) {
	parts := strings.Split(cb.Data, ":")
	action := parts[0]
	matchID := parts[1]

	m, exists := b.games.GetTTT(matchID)
	if !exists {
		b.api.Send(tgbotapi.NewCallback(cb.ID, "Match expired or finished."))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	switch action {
	case "ttt_cancel":
		if cb.From.ID != m.PlayerX {
			b.api.Send(tgbotapi.NewCallback(cb.ID, "Only the challenger can cancel!"))
			return
		}
		_ = b.db.Users.AddCoins(ctx, m.PlayerX, m.Bet)
		b.games.RemoveTTT(matchID)
		b.api.Send(tgbotapi.NewCallback(cb.ID, "Match cancelled. Coins refunded."))

		cancelText := fmt.Sprintf("❌ <b>Tic-Tac-Toe match cancelled by %s.</b>", cb.From.FirstName)
		edit := tgbotapi.NewEditMessageText(cb.Message.Chat.ID, cb.Message.MessageID, credit.Watermark(cancelText))
		edit.ParseMode = "HTML"
		b.api.Send(edit)

	case "ttt_join":
		if cb.From.ID == m.PlayerX {
			b.api.Send(tgbotapi.NewCallback(cb.ID, "You cannot challenge yourself!"))
			return
		}

		u2, err := b.EnsureUser(cb.From)
		if err != nil || u2.Coins < m.Bet {
			b.api.Send(tgbotapi.NewCallback(cb.ID, "Insufficient coins to join this match!"))
			return
		}

		if err := b.db.Users.DeductCoins(ctx, u2.ID, m.Bet); err != nil {
			b.api.Send(tgbotapi.NewCallback(cb.ID, "Error deducting wager."))
			return
		}

		if !m.Join(u2.ID, cb.From.FirstName) {
			_ = b.db.Users.AddCoins(ctx, u2.ID, m.Bet)
			b.api.Send(tgbotapi.NewCallback(cb.ID, "Failed to join match."))
			return
		}

		b.api.Send(tgbotapi.NewCallback(cb.ID, "Challenge accepted! Game started."))

		gameText := fmt.Sprintf(
			"⚔️ <b>Tic-Tac-Toe Active!</b>\n\n"+
				"❌ <b>%s</b> vs ⭕ <b>%s</b>\n"+
				"💰 Total Pot: <b>%d Coins</b>\n\n"+
				"👉 <b>Turn:</b> %s (❌)",
			m.PlayerXName,
			m.PlayerOName,
			m.Bet*2,
			m.PlayerXName,
		)

		edit := tgbotapi.NewEditMessageText(cb.Message.Chat.ID, cb.Message.MessageID, credit.Watermark(gameText))
		edit.ParseMode = "HTML"
		kb := m.RenderKeyboard()
		edit.ReplyMarkup = &kb
		b.api.Send(edit)

	case "ttt_move":
		if len(parts) < 4 {
			return
		}
		row, _ := strconv.Atoi(parts[2])
		col, _ := strconv.Atoi(parts[3])

		valid, finished, winner, isDraw := m.MakeMove(cb.From.ID, row, col)
		if !valid {
			b.api.Send(tgbotapi.NewCallback(cb.ID, "Invalid move or not your turn!"))
			return
		}

		b.api.Send(tgbotapi.NewCallback(cb.ID, ""))

		if finished {
			b.games.RemoveTTT(matchID)
			var resultText string
			totalPot := m.Bet * 2

			if isDraw {
				_ = b.db.Users.AddCoins(ctx, m.PlayerX, m.Bet)
				_ = b.db.Users.AddCoins(ctx, m.PlayerO, m.Bet)
				_ = b.db.Users.RecordGameResult(ctx, m.PlayerX, false, true, m.Bet, 20)
				_ = b.db.Users.RecordGameResult(ctx, m.PlayerO, false, true, m.Bet, 20)
				resultText = fmt.Sprintf(
					"🤝 <b>Tic-Tac-Toe Match: DRAW!</b>\n\n"+
						"❌ %s vs ⭕ %s\n"+
						"💰 Bets refunded (%d coins each).\n\n"+
						"%s",
					m.PlayerXName, m.PlayerOName, m.Bet, credit.GetCreditBanner(),
				)
			} else {
				winnerName := m.PlayerXName
				loserID := m.PlayerO
				if winner == m.PlayerO {
					winnerName = m.PlayerOName
					loserID = m.PlayerX
				}

				_ = b.db.Users.RecordGameResult(ctx, winner, true, false, totalPot, 50)
				_ = b.db.Users.RecordGameResult(ctx, loserID, false, false, 0, 15)

				_ = b.db.Games.LogGame(ctx, database.GameLog{
					GameType:   "tictactoe",
					UserID:     winner,
					OpponentID: loserID,
					BetAmount:  m.Bet,
					Payout:     totalPot,
					Result:     "win",
				})

				resultText = fmt.Sprintf(
					"🏆 <b>Tic-Tac-Toe WINNER: %s!</b>\n\n"+
						"🎉 Prize Won: <b>%d Coins</b> (+50 XP)!\n\n"+
						"%s",
					winnerName, totalPot, credit.GetCreditBanner(),
				)
			}

			edit := tgbotapi.NewEditMessageText(cb.Message.Chat.ID, cb.Message.MessageID, credit.Watermark(resultText))
			edit.ParseMode = "HTML"
			kb := m.RenderKeyboard()
			edit.ReplyMarkup = &kb
			b.api.Send(edit)
		} else {
			turnName := m.PlayerXName
			sym := "❌"
			if m.CurrentTurn == m.PlayerO {
				turnName = m.PlayerOName
				sym = "⭕"
			}

			text := fmt.Sprintf(
				"⚔️ <b>Tic-Tac-Toe Active!</b>\n\n"+
					"❌ <b>%s</b> vs ⭕ <b>%s</b>\n"+
					"💰 Total Pot: <b>%d Coins</b>\n\n"+
					"👉 <b>Turn:</b> %s (%s)",
				m.PlayerXName, m.PlayerOName, m.Bet*2, turnName, sym,
			)

			edit := tgbotapi.NewEditMessageText(cb.Message.Chat.ID, cb.Message.MessageID, credit.Watermark(text))
			edit.ParseMode = "HTML"
			kb := m.RenderKeyboard()
			edit.ReplyMarkup = &kb
			b.api.Send(edit)
		}
	}
}

func (b *Bot) handleRPSCallback(cb *tgbotapi.CallbackQuery) {
	parts := strings.Split(cb.Data, ":")
	action := parts[0]
	matchID := parts[1]

	m, exists := b.games.GetRPS(matchID)
	if !exists {
		b.api.Send(tgbotapi.NewCallback(cb.ID, "Duel expired or finished."))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	switch action {
	case "rps_cancel":
		if cb.From.ID != m.Player1 {
			b.api.Send(tgbotapi.NewCallback(cb.ID, "Only the challenger can cancel!"))
			return
		}
		_ = b.db.Users.AddCoins(ctx, m.Player1, m.Bet)
		b.games.RemoveRPS(matchID)
		b.api.Send(tgbotapi.NewCallback(cb.ID, "Duel cancelled. Coins refunded."))

		edit := tgbotapi.NewEditMessageText(cb.Message.Chat.ID, cb.Message.MessageID, credit.Watermark("❌ Duel cancelled."))
		b.api.Send(edit)

	case "rps_join":
		if cb.From.ID == m.Player1 {
			b.api.Send(tgbotapi.NewCallback(cb.ID, "You cannot fight yourself!"))
			return
		}

		u2, err := b.EnsureUser(cb.From)
		if err != nil || u2.Coins < m.Bet {
			b.api.Send(tgbotapi.NewCallback(cb.ID, "Insufficient coins to join this duel!"))
			return
		}

		if err := b.db.Users.DeductCoins(ctx, u2.ID, m.Bet); err != nil {
			b.api.Send(tgbotapi.NewCallback(cb.ID, "Error deducting wager."))
			return
		}

		if !m.Join(u2.ID, cb.From.FirstName) {
			_ = b.db.Users.AddCoins(ctx, u2.ID, m.Bet)
			b.api.Send(tgbotapi.NewCallback(cb.ID, "Could not join duel."))
			return
		}

		b.api.Send(tgbotapi.NewCallback(cb.ID, "Duel accepted! Select your weapon!"))

		text := fmt.Sprintf(
			"⚔️ <b>Rock-Paper-Scissors Active!</b>\n\n"+
				"👤 <b>%s</b> vs 👤 <b>%s</b>\n"+
				"💰 Total Pot: <b>%d Coins</b>\n\n"+
				"Both players: choose your weapon below!",
			m.Player1Name, m.Player2Name, m.Bet*2,
		)

		edit := tgbotapi.NewEditMessageText(cb.Message.Chat.ID, cb.Message.MessageID, credit.Watermark(text))
		edit.ParseMode = "HTML"
		kb := m.RenderKeyboard()
		edit.ReplyMarkup = &kb
		b.api.Send(edit)

	case "rps_pick":
		if len(parts) < 3 {
			return
		}
		choice := parts[2]

		if cb.From.ID != m.Player1 && cb.From.ID != m.Player2 {
			b.api.Send(tgbotapi.NewCallback(cb.ID, "You are not a participant in this duel!"))
			return
		}

		finished, winner, isDraw := m.Choose(cb.From.ID, choice)
		b.api.Send(tgbotapi.NewCallback(cb.ID, fmt.Sprintf("Selected: %s!", choice)))

		if finished {
			b.games.RemoveRPS(matchID)
			totalPot := m.Bet * 2
			var resultText string

			emojiMap := map[string]string{
				"rock":     "🪨 Rock",
				"paper":    "📄 Paper",
				"scissors": "✂️ Scissors",
			}

			if isDraw {
				_ = b.db.Users.AddCoins(ctx, m.Player1, m.Bet)
				_ = b.db.Users.AddCoins(ctx, m.Player2, m.Bet)
				_ = b.db.Users.RecordGameResult(ctx, m.Player1, false, true, m.Bet, 20)
				_ = b.db.Users.RecordGameResult(ctx, m.Player2, false, true, m.Bet, 20)

				resultText = fmt.Sprintf(
					"🤝 <b>RPS Duel: DRAW!</b>\n\n"+
						"• %s: %s\n"+
						"• %s: %s\n\n"+
						"💰 Bets refunded (%d coins each).",
					m.Player1Name, emojiMap[m.Player1Move],
					m.Player2Name, emojiMap[m.Player2Move],
					m.Bet,
				)
			} else {
				winnerName := m.Player1Name
				loserID := m.Player2
				if winner == m.Player2 {
					winnerName = m.Player2Name
					loserID = m.Player1
				}

				_ = b.db.Users.RecordGameResult(ctx, winner, true, false, totalPot, 50)
				_ = b.db.Users.RecordGameResult(ctx, loserID, false, false, 0, 15)

				_ = b.db.Games.LogGame(ctx, database.GameLog{
					GameType:   "rps",
					UserID:     winner,
					OpponentID: loserID,
					BetAmount:  m.Bet,
					Payout:     totalPot,
					Result:     "win",
				})

				resultText = fmt.Sprintf(
					"🏆 <b>RPS Duel Winner: %s!</b>\n\n"+
						"• %s: %s\n"+
						"• %s: %s\n\n"+
						"💰 Prize Won: <b>+%d Coins</b> (+50 XP)!\n\n"+
						"%s",
					winnerName,
					m.Player1Name, emojiMap[m.Player1Move],
					m.Player2Name, emojiMap[m.Player2Move],
					totalPot,
					credit.GetCreditBanner(),
				)
			}

			edit := tgbotapi.NewEditMessageText(cb.Message.Chat.ID, cb.Message.MessageID, credit.Watermark(resultText))
			edit.ParseMode = "HTML"
			b.api.Send(edit)
		}
	}
}
