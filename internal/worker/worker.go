package worker

import (
	"context"
	"sync"

	"github.com/AsenHu/mewlink/internal/config"
	"github.com/AsenHu/mewlink/internal/database"
	"github.com/go-telegram/bot"
	"maunium.net/go/mautrix"
)

type Worker struct {
	Matrix    *mautrix.Client
	Telegram  *bot.Bot
	DataBase  *database.DataBase
	Config    *config.Config
	WaitGroup *sync.WaitGroup
	Context   context.Context
	StopProc  context.CancelFunc
}
