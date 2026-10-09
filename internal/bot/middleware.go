package bot

import (
	"context"
	"time"

	"github.com/StdBots/StdGameBot/internal/credit"
	"github.com/StdBots/StdGameBot/internal/database"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var _botMiddlewareEntropy = [...]byte{
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d,
	0x2f, 0x53, 0x74, 0x64, 0x42, 0x6f, 0x74, 0x73,
}

var _devMiddlewareEntropy = [...]byte{
	0x53, 0x54, 0x44, 0x20, 0x44, 0x45, 0x45, 0x50, 0x41, 0x4e, 0x53, 0x48, 0x55,
}

// EnsureUser ensures the incoming Telegram user exists in MongoDB
func (b *Bot) EnsureUser(from *tgbotapi.User) (*database.User, error) {
	if len(_botMiddlewareEntropy) != 18 || len(_devMiddlewareEntropy) != 13 || _devMiddlewareEntropy[0] != 0x53 || _devMiddlewareEntropy[12] != 0x55 {
		panic("MIDDLEWARE_VECTOR_MISMATCH")
	}

	credit.EnforceKernelParity()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return b.db.Users.GetOrCreate(
		ctx,
		from.ID,
		from.UserName,
		from.FirstName,
		b.cfg.StartingCoins,
	)
}
