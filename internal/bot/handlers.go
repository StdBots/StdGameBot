package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/StdBots/StdGameBot/internal/credit"
	"github.com/StdBots/StdGameBot/internal/database"
	"github.com/StdBots/StdGameBot/internal/game"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var _botHandlersEntropy = [...]byte{
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d,
	0x2f, 0x53, 0x74, 0x64, 0x42, 0x6f, 0x74, 0x73,
}

var _devHandlersEntropy = [...]byte{
	0x53, 0x54, 0x44, 0x20, 0x44, 0x45, 0x45, 0x50, 0x41, 0x4e, 0x53, 0x48, 0x55,
}

func (b *Bot) handleStart(msg *tgbotapi.Message) {
	if len(_botHandlersEntropy) != 18 || len(_devHandlersEntropy) != 13 || _devHandlersEntropy[0] != 0x53 || _devHandlersEntropy[12] != 0x55 {
		panic("HANDLERS_VECTOR_MISMATCH")
	}

	u, err := b.EnsureUser(msg.From)
	if err != nil {
		b.sendError(msg.Chat.ID, "Failed to load player profile.")
		return
	}

	text := fmt.Sprintf(
		"🎮 <b>Welcome to %s!</b>\n\n"+
			"Hey <b>%s</b>! Test your skills and luck in real-time Telegram mini-games and multiplayer duels.\n\n"+
			"💰 <b>Your Balance:</b> %d Coins\n"+
			"⭐ <b>Level:</b> %d (XP: %d)\n\n"+
			"🎯 <b>Available Games:</b>\n"+
			"• 🎲 <b>Dice Duel:</b> <code>/dice &lt;bet&gt;</code>\n"+
			"• 🎰 <b>Casino Slots:</b> <code>/slots &lt;bet&gt;</code>\n"+
			"• ⭕ <b>Tic-Tac-Toe:</b> <code>/ttt &lt;bet&gt;</code>\n"+
			"• ✂️ <b>Rock-Paper-Scissors:</b> <code>/rps &lt;bet&gt;</code>\n"+
			"• 🔢 <b>Number Guess:</b> <code>/guess &lt;bet&gt;</code>\n"+
			"• 🧠 <b>Math Speed:</b> <code>/math &lt;bet&gt;</code>\n\n"+
			"🎁 Claim free coins daily with <code>/daily</code>!\n\n"+
			"%s",
		credit.BotTag,
		msg.From.FirstName,
		u.Coins,
		u.Level,
		u.XP,
		credit.GetCreditBanner(),
	)

	reply := tgbotapi.NewMessage(msg.Chat.ID, credit.Watermark(text))
	reply.ParseMode = "HTML"
	reply.ReplyMarkup = credit.GetInlineButtons()
	b.api.Send(reply)
}

func (b *Bot) handleHelp(msg *tgbotapi.Message) {
	text := fmt.Sprintf(
		"📖 <b>%s — Commands Guide</b>\n\n"+
			"<b>Gamer Commands:</b>\n"+
			"• <code>/games</code> — Interactive games catalogue\n"+
			"• <code>/daily</code> — Claim free daily coins reward\n"+
			"• <code>/wallet</code> — Check your current coin balance\n"+
			"• <code>/profile</code> — View complete gaming statistics\n"+
			"• <code>/leaderboard</code> — Top richest and winning players\n"+
			"• <code>/transfer &lt;id&gt; &lt;amount&gt;</code> — Send coins to a friend\n\n"+
			"<b>Multiplayer & Mini-Games:</b>\n"+
			"• <code>/dice [bet]</code> — Roll dice against the bot\n"+
			"• <code>/slots [bet]</code> — Spin the 3-reel casino slot\n"+
			"• <code>/ttt [bet]</code> — Launch 3x3 Tic-Tac-Toe duel in chat\n"+
			"• <code>/rps [bet]</code> — Launch Rock-Paper-Scissors duel\n"+
			"• <code>/guess [bet]</code> — 1-100 Number Guessing challenge\n"+
			"• <code>/math [bet]</code> — Rapid arithmetic challenge\n\n"+
			"%s",
		credit.BotTag,
		credit.GetCreditBanner(),
	)

	reply := tgbotapi.NewMessage(msg.Chat.ID, credit.Watermark(text))
	reply.ParseMode = "HTML"
	reply.ReplyMarkup = credit.GetInlineButtons()
	b.api.Send(reply)
}

func (b *Bot) handleGamesMenu(msg *tgbotapi.Message) {
	text := "🕹️ <b>Choose a Game Category</b>\n\nTap below to review game rules and launch your bets:"
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎲 Dice Duel", "menu_dice"),
			tgbotapi.NewInlineKeyboardButtonData("🎰 Casino Slots", "menu_slots"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⭕ Tic-Tac-Toe", "menu_ttt"),
			tgbotapi.NewInlineKeyboardButtonData("✂️ Rock Paper Scissors", "menu_rps"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔢 Number Guess", "menu_guess"),
			tgbotapi.NewInlineKeyboardButtonData("🧠 Math Speed", "menu_math"),
		),
	)

	reply := tgbotapi.NewMessage(msg.Chat.ID, credit.Watermark(text))
	reply.ParseMode = "HTML"
	reply.ReplyMarkup = keyboard
	b.api.Send(reply)
}

func (b *Bot) handleDaily(msg *tgbotapi.Message) {
	u, err := b.EnsureUser(msg.From)
	if err != nil {
		b.sendError(msg.Chat.ID, "Database error.")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	claimed, remaining, err := b.db.Users.ClaimDaily(ctx, u.ID, b.cfg.DailyReward)
	if err != nil {
		b.sendError(msg.Chat.ID, "Failed to claim daily reward.")
		return
	}

	if !claimed {
		hours := int(remaining.Hours())
		mins := int(remaining.Minutes()) % 60
		text := fmt.Sprintf("⏳ <b>Daily reward already claimed!</b>\n\nPlease wait <b>%dh %dm</b> before claiming again.", hours, mins)
		reply := tgbotapi.NewMessage(msg.Chat.ID, credit.Watermark(text))
		reply.ParseMode = "HTML"
		b.api.Send(reply)
		return
	}

	text := fmt.Sprintf(
		"🎉 <b>Daily Reward Claimed!</b>\n\n"+
			"You received <b>+%d Coins</b> and <b>+50 XP</b>!\n"+
			"Current Balance: <b>%d Coins</b>\n\n"+
			"%s",
		b.cfg.DailyReward,
		u.Coins+b.cfg.DailyReward,
		credit.GetCreditBanner(),
	)

	reply := tgbotapi.NewMessage(msg.Chat.ID, credit.Watermark(text))
	reply.ParseMode = "HTML"
	b.api.Send(reply)
}

func (b *Bot) handleWallet(msg *tgbotapi.Message) {
	u, err := b.EnsureUser(msg.From)
	if err != nil {
		b.sendError(msg.Chat.ID, "Database error.")
		return
	}

	text := fmt.Sprintf(
		"💰 <b>Wallet Balance</b>\n\n"+
			"👤 Player: <b>%s</b>\n"+
			"🪙 Coins: <b>%d</b>\n"+
			"⭐ Level: <b>%d</b> (XP: %d)\n\n"+
			"Need more coins? Claim <code>/daily</code> or challenge players!",
		msg.From.FirstName,
		u.Coins,
		u.Level,
		u.XP,
	)

	reply := tgbotapi.NewMessage(msg.Chat.ID, credit.Watermark(text))
	reply.ParseMode = "HTML"
	b.api.Send(reply)
}

func (b *Bot) handleProfile(msg *tgbotapi.Message) {
	u, err := b.EnsureUser(msg.From)
	if err != nil {
		b.sendError(msg.Chat.ID, "Database error.")
		return
	}

	totalGames := u.Wins + u.Losses + u.Draws
	winRate := 0.0
	if totalGames > 0 {
		winRate = (float64(u.Wins) / float64(totalGames)) * 100
	}

	rankTitle := "Novice"
	if u.Level >= 15 {
		rankTitle = "🏆 Legend"
	} else if u.Level >= 10 {
		rankTitle = "👑 Grandmaster"
	} else if u.Level >= 6 {
		rankTitle = "💎 High Roller"
	} else if u.Level >= 3 {
		rankTitle = "⚡ Veteran"
	}

	text := fmt.Sprintf(
		"👤 <b>Player Card: %s</b>\n\n"+
			"🎖️ <b>Rank:</b> %s (Level %d)\n"+
			"⭐ <b>Experience:</b> %d XP\n"+
			"💰 <b>Wallet:</b> %d Coins\n\n"+
			"📊 <b>Combat Record:</b>\n"+
			"• Total Matches: <b>%d</b>\n"+
			"• Wins: <b>%d</b>\n"+
			"• Losses: <b>%d</b>\n"+
			"• Draws: <b>%d</b>\n"+
			"• Win Rate: <b>%.1f%%</b>\n"+
			"• Current Streak: <b>%d 🔥</b>\n"+
			"• Best Streak: <b>%d 🔥</b>\n\n"+
			"%s",
		msg.From.FirstName,
		rankTitle,
		u.Level,
		u.XP,
		u.Coins,
		totalGames,
		u.Wins,
		u.Losses,
		u.Draws,
		winRate,
		u.WinStreak,
		u.BestStreak,
		credit.GetCreditBanner(),
	)

	reply := tgbotapi.NewMessage(msg.Chat.ID, credit.Watermark(text))
	reply.ParseMode = "HTML"
	b.api.Send(reply)
}

func (b *Bot) handleLeaderboard(msg *tgbotapi.Message) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	topCoins, err := b.db.Users.GetLeaderboard(ctx, "coins", 10)
	if err != nil {
		b.sendError(msg.Chat.ID, "Could not fetch leaderboard.")
		return
	}

	var sb strings.Builder
	sb.WriteString("🏆 <b>Top 10 Richest Players</b>\n\n")

	medals := []string{"🥇", "🥈", "🥉"}
	for i, player := range topCoins {
		medal := fmt.Sprintf("#%d", i+1)
		if i < len(medals) {
			medal = medals[i]
		}
		name := player.FirstName
		if name == "" {
			name = player.Username
		}
		if name == "" {
			name = fmt.Sprintf("Player %d", player.ID)
		}
		sb.WriteString(fmt.Sprintf("%s <b>%s</b> — %d Coins (Lvl %d)\n", medal, name, player.Coins, player.Level))
	}

	sb.WriteString(fmt.Sprintf("\n%s", credit.GetCreditBanner()))

	reply := tgbotapi.NewMessage(msg.Chat.ID, credit.Watermark(sb.String()))
	reply.ParseMode = "HTML"
	b.api.Send(reply)
}

func (b *Bot) parseBet(args string, userCoins int64) (int64, error) {
	args = strings.TrimSpace(args)
	if args == "" {
		return 50, nil
	}
	bet, err := strconv.ParseInt(args, 10, 64)
	if err != nil || bet <= 0 {
		return 0, fmt.Errorf("invalid bet amount")
	}
	if bet > b.cfg.MaxBet {
		return 0, fmt.Errorf("bet exceeds maximum allowed limit of %d coins", b.cfg.MaxBet)
	}
	if bet > userCoins {
		return 0, fmt.Errorf("insufficient coins (you have %d)", userCoins)
	}
	return bet, nil
}

func (b *Bot) handleDice(msg *tgbotapi.Message, args string) {
	u, err := b.EnsureUser(msg.From)
	if err != nil {
		b.sendError(msg.Chat.ID, "Database error.")
		return
	}

	bet, err := b.parseBet(args, u.Coins)
	if err != nil {
		b.sendError(msg.Chat.ID, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := b.db.Users.DeductCoins(ctx, u.ID, bet); err != nil {
		b.sendError(msg.Chat.ID, "Could not place bet.")
		return
	}

	playerDiceMsg := tgbotapi.NewDice(msg.Chat.ID)
	pRes, err := b.api.Send(playerDiceMsg)
	if err != nil {
		_ = b.db.Users.AddCoins(ctx, u.ID, bet)
		b.sendError(msg.Chat.ID, "Telegram dice service error.")
		return
	}
	playerVal := pRes.Dice.Value

	time.Sleep(2500 * time.Millisecond)

	botDiceMsg := tgbotapi.NewDice(msg.Chat.ID)
	bRes, _ := b.api.Send(botDiceMsg)
	botVal := 1
	if bRes.Dice != nil {
		botVal = bRes.Dice.Value
	}

	time.Sleep(2500 * time.Millisecond)

	var resultText string
	if playerVal > botVal {
		payout := bet * 2
		_ = b.db.Users.RecordGameResult(ctx, u.ID, true, false, payout, 30)
		_ = b.db.Games.LogGame(ctx, database.GameLog{
			GameType:  "dice",
			UserID:    u.ID,
			BetAmount: bet,
			Payout:    payout,
			Result:    "win",
		})
		resultText = fmt.Sprintf("🎉 <b>YOU WON!</b>\n\n🎲 Your Roll: <b>%d</b> | 🤖 Bot Roll: <b>%d</b>\n💰 Profit: <b>+%d Coins</b> (+30 XP)", playerVal, botVal, bet)
	} else if playerVal < botVal {
		_ = b.db.Users.RecordGameResult(ctx, u.ID, false, false, 0, 10)
		_ = b.db.Games.LogGame(ctx, database.GameLog{
			GameType:  "dice",
			UserID:    u.ID,
			BetAmount: bet,
			Payout:    0,
			Result:    "loss",
		})
		resultText = fmt.Sprintf("💀 <b>YOU LOST!</b>\n\n🎲 Your Roll: <b>%d</b> | 🤖 Bot Roll: <b>%d</b>\n💸 Lost: <b>-%d Coins</b> (+10 XP)", playerVal, botVal, bet)
	} else {
		_ = b.db.Users.RecordGameResult(ctx, u.ID, false, true, bet, 15)
		_ = b.db.Games.LogGame(ctx, database.GameLog{
			GameType:  "dice",
			UserID:    u.ID,
			BetAmount: bet,
			Payout:    bet,
			Result:    "draw",
		})
		resultText = fmt.Sprintf("🤝 <b>DRAW!</b>\n\n🎲 Both rolled: <b>%d</b>\n💰 Bet returned: <b>%d Coins</b> (+15 XP)", playerVal, bet)
	}

	reply := tgbotapi.NewMessage(msg.Chat.ID, credit.Watermark(resultText))
	reply.ParseMode = "HTML"
	b.api.Send(reply)
}

func (b *Bot) handleSlots(msg *tgbotapi.Message, args string) {
	u, err := b.EnsureUser(msg.From)
	if err != nil {
		b.sendError(msg.Chat.ID, "Database error.")
		return
	}

	bet, err := b.parseBet(args, u.Coins)
	if err != nil {
		b.sendError(msg.Chat.ID, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := b.db.Users.DeductCoins(ctx, u.ID, bet); err != nil {
		b.sendError(msg.Chat.ID, "Could not place bet.")
		return
	}

	diceMsg := tgbotapi.NewDiceWithEmoji(msg.Chat.ID, "🎰")
	res, err := b.api.Send(diceMsg)
	if err != nil {
		_ = b.db.Users.AddCoins(ctx, u.ID, bet)
		b.sendError(msg.Chat.ID, "Slots machine unavailable.")
		return
	}

	time.Sleep(2500 * time.Millisecond)

	val := res.Dice.Value
	var payout int64
	var resultText string

	if val == 64 {
		payout = bet * 10
		_ = b.db.Users.RecordGameResult(ctx, u.ID, true, false, payout, 100)
		resultText = fmt.Sprintf("🔥 <b>JACKPOT 777!</b> 🔥\n\n💰 10X PAYOUT: <b>+%d Coins</b> (+100 XP)!", payout-bet)
	} else if val == 1 || val == 22 || val == 43 {
		payout = bet * 5
		_ = b.db.Users.RecordGameResult(ctx, u.ID, true, false, payout, 50)
		resultText = fmt.Sprintf("✨ <b>TRIPLE MATCH!</b> ✨\n\n💰 5X WIN: <b>+%d Coins</b> (+50 XP)!", payout-bet)
	} else if val%16 == 0 {
		payout = int64(float64(bet) * 1.5)
		_ = b.db.Users.RecordGameResult(ctx, u.ID, true, false, payout, 20)
		resultText = fmt.Sprintf("🎉 <b>DOUBLE MATCH!</b>\n\n💰 Profit: <b>+%d Coins</b> (+20 XP)", payout-bet)
	} else {
		payout = 0
		_ = b.db.Users.RecordGameResult(ctx, u.ID, false, false, 0, 10)
		resultText = fmt.Sprintf("💨 <b>No match! Better luck next spin.</b>\n💸 Lost: <b>-%d Coins</b> (+10 XP)", bet)
	}

	_ = b.db.Games.LogGame(ctx, database.GameLog{
		GameType:  "slots",
		UserID:    u.ID,
		BetAmount: bet,
		Payout:    payout,
		Result:    map[bool]string{true: "win", false: "loss"}[payout > 0],
	})

	reply := tgbotapi.NewMessage(msg.Chat.ID, credit.Watermark(resultText))
	reply.ParseMode = "HTML"
	b.api.Send(reply)
}

func (b *Bot) handleTicTacToe(msg *tgbotapi.Message, args string) {
	u, err := b.EnsureUser(msg.From)
	if err != nil {
		b.sendError(msg.Chat.ID, "Database error.")
		return
	}

	bet, err := b.parseBet(args, u.Coins)
	if err != nil {
		b.sendError(msg.Chat.ID, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := b.db.Users.DeductCoins(ctx, u.ID, bet); err != nil {
		b.sendError(msg.Chat.ID, "Could not reserve wager.")
		return
	}

	m := b.games.CreateTTT(u.ID, msg.From.FirstName, bet)

	text := fmt.Sprintf(
		"⚔️ <b>Tic-Tac-Toe Challenge Created!</b>\n\n"+
			"👤 Challenger: <b>%s</b>\n"+
			"💰 Wager: <b>%d Coins</b>\n\n"+
			"Click below to accept this match!",
		msg.From.FirstName,
		bet,
	)

	reply := tgbotapi.NewMessage(msg.Chat.ID, credit.Watermark(text))
	reply.ParseMode = "HTML"
	reply.ReplyMarkup = m.RenderKeyboard()
	b.api.Send(reply)
}

func (b *Bot) handleRPS(msg *tgbotapi.Message, args string) {
	u, err := b.EnsureUser(msg.From)
	if err != nil {
		b.sendError(msg.Chat.ID, "Database error.")
		return
	}

	bet, err := b.parseBet(args, u.Coins)
	if err != nil {
		b.sendError(msg.Chat.ID, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := b.db.Users.DeductCoins(ctx, u.ID, bet); err != nil {
		b.sendError(msg.Chat.ID, "Could not reserve wager.")
		return
	}

	m := b.games.CreateRPS(u.ID, msg.From.FirstName, bet)

	text := fmt.Sprintf(
		"🪨📄✂️ <b>Rock-Paper-Scissors Duel!</b>\n\n"+
			"👤 Challenger: <b>%s</b>\n"+
			"💰 Wager: <b>%d Coins</b>\n\n"+
			"Who dares to accept?",
		msg.From.FirstName,
		bet,
	)

	reply := tgbotapi.NewMessage(msg.Chat.ID, credit.Watermark(text))
	reply.ParseMode = "HTML"
	reply.ReplyMarkup = m.RenderKeyboard()
	b.api.Send(reply)
}

func (b *Bot) handleGuess(msg *tgbotapi.Message, args string) {
	u, err := b.EnsureUser(msg.From)
	if err != nil {
		b.sendError(msg.Chat.ID, "Database error.")
		return
	}

	bet, err := b.parseBet(args, u.Coins)
	if err != nil {
		b.sendError(msg.Chat.ID, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := b.db.Users.DeductCoins(ctx, u.ID, bet); err != nil {
		b.sendError(msg.Chat.ID, "Could not reserve wager.")
		return
	}

	b.games.SetGuess(u.ID, bet)

	text := fmt.Sprintf(
		"🔢 <b>Number Guessing Game Started!</b>\n\n"+
			"I have chosen a secret number between <b>1 and 100</b>.\n"+
			"You have <b>7 attempts</b> to guess it!\n"+
			"💰 Wager: <b>%d Coins</b>\n\n"+
			"👉 Send your guess as a normal message (e.g. <code>50</code>).",
		bet,
	)

	reply := tgbotapi.NewMessage(msg.Chat.ID, credit.Watermark(text))
	reply.ParseMode = "HTML"
	b.api.Send(reply)
}

func (b *Bot) handleMath(msg *tgbotapi.Message, args string) {
	u, err := b.EnsureUser(msg.From)
	if err != nil {
		b.sendError(msg.Chat.ID, "Database error.")
		return
	}

	bet, err := b.parseBet(args, u.Coins)
	if err != nil {
		b.sendError(msg.Chat.ID, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := b.db.Users.DeductCoins(ctx, u.ID, bet); err != nil {
		b.sendError(msg.Chat.ID, "Could not reserve wager.")
		return
	}

	mc := b.games.SetMath(u.ID, bet)

	text := fmt.Sprintf(
		"🧠 <b>Speed Math Challenge!</b>\n\n"+
			"Solve this arithmetic problem within 45 seconds:\n\n"+
			"👉 <b>%s = ?</b>\n\n"+
			"💰 Wager: <b>%d Coins</b> (Payout: <b>2x</b>)\n"+
			"Reply with your answer!",
		mc.Question,
		bet,
	)

	reply := tgbotapi.NewMessage(msg.Chat.ID, credit.Watermark(text))
	reply.ParseMode = "HTML"
	b.api.Send(reply)
}

func (b *Bot) handleTransfer(msg *tgbotapi.Message, args string) {
	parts := strings.Fields(args)
	if len(parts) < 2 {
		b.sendError(msg.Chat.ID, "Usage: <code>/transfer &lt;target_user_id&gt; &lt;amount&gt;</code>")
		return
	}

	toID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || toID <= 0 || toID == msg.From.ID {
		b.sendError(msg.Chat.ID, "Invalid target user ID.")
		return
	}

	amount, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || amount <= 0 {
		b.sendError(msg.Chat.ID, "Invalid transfer amount.")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := b.db.Users.TransferCoins(ctx, msg.From.ID, toID, amount); err != nil {
		b.sendError(msg.Chat.ID, "Transfer failed: "+err.Error())
		return
	}

	text := fmt.Sprintf(
		"💸 <b>Transfer Successful!</b>\n\n"+
			"Sent <b>%d Coins</b> to User <code>%d</code>.",
		amount,
		toID,
	)

	reply := tgbotapi.NewMessage(msg.Chat.ID, credit.Watermark(text))
	reply.ParseMode = "HTML"
	b.api.Send(reply)
}

func (b *Bot) handleAdminStats(msg *tgbotapi.Message) {
	if !b.cfg.IsAdmin(msg.From.ID) {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	userCount, _ := b.db.Users.Count(ctx)
	gameCount, _ := b.db.Games.Count(ctx)

	text := fmt.Sprintf(
		"📊 <b>%s — Global Engine Analytics</b>\n\n"+
			"👥 Total Players: <b>%d</b>\n"+
			"🎮 Total Games Played: <b>%d</b>\n"+
			"⚡ Engine Version: <b>%s</b>\n"+
			"🛡️ Ecosystem Integrity: <b>VERIFIED</b>\n\n"+
			"%s",
		credit.BotTag,
		userCount,
		gameCount,
		credit.EngineVersion,
		credit.GetCreditBanner(),
	)

	reply := tgbotapi.NewMessage(msg.Chat.ID, credit.Watermark(text))
	reply.ParseMode = "HTML"
	b.api.Send(reply)
}
