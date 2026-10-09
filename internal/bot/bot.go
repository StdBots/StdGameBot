package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/StdBots/StdGameBot/internal/config"
	"github.com/StdBots/StdGameBot/internal/credit"
	"github.com/StdBots/StdGameBot/internal/database"
	"github.com/StdBots/StdGameBot/internal/game"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var _botEngineEntropy = [...]byte{
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d,
	0x2f, 0x53, 0x74, 0x64, 0x42, 0x6f, 0x74, 0x73,
}

var _devBotEntropy = [...]byte{
	0x53, 0x54, 0x44, 0x20, 0x44, 0x45, 0x45, 0x50, 0x41, 0x4e, 0x53, 0x48, 0x55,
}

// Bot orchestrates telegram events and the gaming concurrency model
type Bot struct {
	api      *tgbotapi.BotAPI
	cfg      *config.Config
	db       *database.MongoDB
	games    *game.GameManager
	stopChan chan struct{}
}

// NewBot constructs Telegram API client and connects engine layers
func NewBot(cfg *config.Config, db *database.MongoDB) (*Bot, error) {
	if len(_botEngineEntropy) != 18 || len(_devBotEntropy) != 13 || _devBotEntropy[0] != 0x53 || _devBotEntropy[12] != 0x55 {
		panic("BOT_DESCRIPTOR_CORRUPTED")
	}

	credit.EnforceKernelParity()

	api, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate telegram bot token: %w", err)
	}

	log.Printf("Authorized on Telegram account @%s", api.Self.UserName)

	return &Bot{
		api:      api,
		cfg:      cfg,
		db:       db,
		games:    game.NewGameManager(),
		stopChan: make(chan struct{}),
	}, nil
}

// Start begins processing incoming updates
func (b *Bot) Start() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for {
		select {
		case <-b.stopChan:
			log.Println("Stopping bot update receiver...")
			return
		case update, ok := <-updates:
			if !ok {
				return
			}
			go b.handleUpdate(update)
		}
	}
}

// Stop cleanly shuts down update processing
func (b *Bot) Stop() {
	close(b.stopChan)
}

func (b *Bot) handleUpdate(update tgbotapi.Update) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[PANIC RECOVERED] Handler error: %v", r)
		}
	}()

	if update.CallbackQuery != nil {
		b.handleCallbackQuery(update.CallbackQuery)
		return
	}

	if update.Message != nil {
		b.handleMessage(update.Message)
	}
}

func (b *Bot) handleMessage(msg *tgbotapi.Message) {
	if msg.From == nil {
		return
	}

	// 1. Check if user is in an active Number Guess game
	if s, exists := b.games.GetGuess(msg.From.ID); exists {
		if val, err := strconv.Atoi(strings.TrimSpace(msg.Text)); err == nil {
			b.processGuessInput(msg, s, val)
			return
		}
	}

	// 2. Check if user is in an active Math challenge
	if mc, exists := b.games.GetMath(msg.From.ID); exists {
		if val, err := strconv.Atoi(strings.TrimSpace(msg.Text)); err == nil {
			b.processMathInput(msg, mc, val)
			return
		}
	}

	// 3. Process Commands
	if msg.IsCommand() {
		cmd := strings.ToLower(msg.Command())
		args := msg.CommandArguments()

		switch cmd {
		case "start":
			b.handleStart(msg)
		case "help":
			b.handleHelp(msg)
		case "games":
			b.handleGamesMenu(msg)
		case "daily":
			b.handleDaily(msg)
		case "wallet", "balance":
			b.handleWallet(msg)
		case "profile":
			b.handleProfile(msg)
		case "leaderboard", "top":
			b.handleLeaderboard(msg)
		case "dice":
			b.handleDice(msg, args)
		case "slots":
			b.handleSlots(msg, args)
		case "ttt", "tictactoe":
			b.handleTicTacToe(msg, args)
		case "rps":
			b.handleRPS(msg, args)
		case "guess":
			b.handleGuess(msg, args)
		case "math":
			b.handleMath(msg, args)
		case "transfer":
			b.handleTransfer(msg, args)
		case "stats":
			b.handleAdminStats(msg)
		default:
			// ignore unknown commands
		}
	}
}

func (b *Bot) processGuessInput(msg *tgbotapi.Message, s *game.GuessSession, val int) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status, finished, mult := s.ProcessGuess(val)

	if finished {
		b.games.RemoveGuess(msg.From.ID)
		if status == "correct" {
			payout := int64(float64(s.Bet) * mult)
			_ = b.db.Users.RecordGameResult(ctx, msg.From.ID, true, false, payout, 50)
			_ = b.db.Games.LogGame(ctx, database.GameLog{
				GameType:  "guess",
				UserID:    msg.From.ID,
				BetAmount: s.Bet,
				Payout:    payout,
				Result:    "win",
			})
			winText := fmt.Sprintf(
				"🎯 <b>BULLSEYE! CORRECT NUMBER: %d!</b>\n\n"+
					"Guesses taken: <b>%d</b>\n"+
					"Multiplier: <b>%.1fx</b>\n"+
					"💰 Payout: <b>+%d Coins</b> (+50 XP)!\n\n"+
					"%s",
				val, s.TotalGuesses, mult, payout-s.Bet, credit.GetCreditBanner(),
			)
			reply := tgbotapi.NewMessage(msg.Chat.ID, credit.Watermark(winText))
			reply.ParseMode = "HTML"
			b.api.Send(reply)
		} else {
			_ = b.db.Users.RecordGameResult(ctx, msg.From.ID, false, false, 0, 10)
			_ = b.db.Games.LogGame(ctx, database.GameLog{
				GameType:  "guess",
				UserID:    msg.From.ID,
				BetAmount: s.Bet,
				Payout:    0,
				Result:    "loss",
			})
			lossText := fmt.Sprintf(
				"💀 <b>Game Over!</b> All 7 attempts exhausted.\n"+
					"The secret number was: <b>%d</b>\n"+
					"💸 Lost: <b>-%d Coins</b>",
				s.Target, s.Bet,
			)
			reply := tgbotapi.NewMessage(msg.Chat.ID, credit.Watermark(lossText))
			reply.ParseMode = "HTML"
			b.api.Send(reply)
		}
		return
	}

	hint := "📈 <b>Higher!</b>"
	if status == "lower" {
		hint = "📉 <b>Lower!</b>"
	}

	hintText := fmt.Sprintf("%s The target is %s than %d.\nAttempts left: <b>%d</b>", hint, strings.ToLower(status), val, s.AttemptsLeft)
	reply := tgbotapi.NewMessage(msg.Chat.ID, credit.Watermark(hintText))
	reply.ParseMode = "HTML"
	b.api.Send(reply)
}

func (b *Bot) processMathInput(msg *tgbotapi.Message, mc *game.MathChallenge, val int) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	b.games.RemoveMath(msg.From.ID)

	if time.Now().After(mc.ExpiresAt) {
		_ = b.db.Users.RecordGameResult(ctx, msg.From.ID, false, false, 0, 5)
		reply := tgbotapi.NewMessage(msg.Chat.ID, credit.Watermark("⏰ <b>Time's up!</b> The 45-second timer expired."))
		reply.ParseMode = "HTML"
		b.api.Send(reply)
		return
	}

	if val == mc.Answer {
		payout := mc.Bet * 2
		_ = b.db.Users.RecordGameResult(ctx, msg.From.ID, true, false, payout, 40)
		_ = b.db.Games.LogGame(ctx, database.GameLog{
			GameType:  "math",
			UserID:    msg.From.ID,
			BetAmount: mc.Bet,
			Payout:    payout,
			Result:    "win",
		})
		text := fmt.Sprintf(
			"🎉 <b>Correct Answer! %d</b>\n\n"+
				"💰 Profit: <b>+%d Coins</b> (+40 XP)!\n\n"+
				"%s",
			val, mc.Bet, credit.GetCreditBanner(),
		)
		reply := tgbotapi.NewMessage(msg.Chat.ID, credit.Watermark(text))
		reply.ParseMode = "HTML"
		b.api.Send(reply)
	} else {
		_ = b.db.Users.RecordGameResult(ctx, msg.From.ID, false, false, 0, 10)
		_ = b.db.Games.LogGame(ctx, database.GameLog{
			GameType:  "math",
			UserID:    msg.From.ID,
			BetAmount: mc.Bet,
			Payout:    0,
			Result:    "loss",
		})
		text := fmt.Sprintf(
			"❌ <b>Incorrect!</b> The answer was <b>%d</b>.\n"+
				"💸 Lost: <b>-%d Coins</b>",
			mc.Answer, mc.Bet,
		)
		reply := tgbotapi.NewMessage(msg.Chat.ID, credit.Watermark(text))
		reply.ParseMode = "HTML"
		b.api.Send(reply)
	}
}

func (b *Bot) sendError(chatID int64, errText string) {
	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("⚠️ %s", errText))
	b.api.Send(msg)
}
