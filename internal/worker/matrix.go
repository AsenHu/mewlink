package worker

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/rs/zerolog/log"
	"maunium.net/go/mautrix/event"
)

func (w *Worker) FromMatrix(ctx context.Context, ev *event.Event) {
	// 如果不是指定用户发送的消息，忽略
	if ev.Sender.String() != w.Config.Content.ServedUser {
		return
	}

	// 从 kv 中获取 Telegram Chat ID
	chatID, found := w.KVStore.GetChatID(ev.RoomID)
	if !found {
		// 如果没有找到 Chat ID，说明根本没有对应的 Chat，直接返回
		return
	}

	log.Info().
		Str("RoomID", ev.RoomID.String()).
		Str("Text", ev.Content.AsMessage().Body).
		Msg("Received message from Matrix")

	// 转发消息到 Telegram
	_, err := w.Telegram.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   ev.Content.AsMessage().Body,
	})

	if err != nil {
		log.Error().Err(err).Msg("Failed to send message to Telegram")
		_, err = w.Matrix.SendText(ctx, ev.RoomID, err.Error())
		if err != nil {
			log.Error().Err(err).Msg("Failed to send message to Matrix")
		}
	}
}
