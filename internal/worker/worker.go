package worker

import (
	"MewLink/internal/config"
	"MewLink/internal/kv"

	"github.com/go-telegram/bot"
	"maunium.net/go/mautrix"
)

type Worker struct {
	Matrix   *mautrix.Client
	Telegram *bot.Bot
	KVStore  *kv.KVStore
	Config   *config.Config
}
