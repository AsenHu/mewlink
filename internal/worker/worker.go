package worker

import (
	"MewLink/internal/config"
	"MewLink/internal/database"

	"github.com/go-telegram/bot"
	"maunium.net/go/mautrix"
)

type Worker struct {
	Matrix   *mautrix.Client
	Telegram *bot.Bot
	DB       *database.DataBase
	Config   *config.Config
}
