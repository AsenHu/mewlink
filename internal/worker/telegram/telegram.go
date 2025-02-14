package telegram

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/AsenHu/mewlink/internal/worker"
	"github.com/AsenHu/mewlink/internal/worker/misc"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type TelegramWorker struct {
	*worker.Worker
}

func (w TelegramWorker) FromTelegram(_ context.Context, _ *bot.Bot, update *models.Update) {
	w.WaitGroup.Add(1)
	go func() {
		defer w.WaitGroup.Done()
		// 确定消息类型，然后调用相应的处理函数

		// 1. 如果是 `/start`，调用 `procStartMsg`
		// 2. 如果是普通消息，调用 `procText`
		// 3. 如果是其他消息，直接返回

		var index []byte
		switch {
		case update.Message.Text == "/start":
			index = w.procStartMsg(w.Context, update)
		case update.Message.Text != "":
			index = w.procText(w.Context, update)
		default:
			if w.Config.Content.LogLevel == zerolog.DebugLevel {
				jsonUpdate, _ := json.Marshal(update)
				log.Debug().
					Str("Update", string(jsonUpdate)).
					Msg("Unsupported message type")
			}
			return
		}

		// 杂项操作
		// 更新房间信息
		if err := misc.UpdateProfile(w.Context, index, w.Matrix); err != nil {
			log.Warn().Err(err).Msg("Failed to update profile")
		}
	}()
}

func getUserName(update *models.Update) string {
	// 1. 尝试拼接 FirstName 和 LastName
	if update.Message.Chat.FirstName != "" || update.Message.Chat.LastName != "" {
		if update.Message.Chat.FirstName == "" {
			return update.Message.Chat.LastName
		}
		if update.Message.Chat.LastName == "" {
			return update.Message.Chat.FirstName
		}
		return update.Message.Chat.FirstName + " " + update.Message.Chat.LastName
	}

	// 2. 如果拼接失败，使用 Username
	if update.Message.Chat.Username != "" {
		return update.Message.Chat.Username
	}

	// 3. 如果 Username 为空，使用 UserID
	return strconv.FormatInt(update.Message.Chat.ID, 10)
}

func (w *TelegramWorker) sendErrToTG(ctx context.Context, chatID int64, err error) {
	message := err.Error() + "\nAn error occurred on the Matrix side. Please try to contact the user through other means."
	_, err = w.Telegram.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   message,
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to send message to Telegram")
	}
}
