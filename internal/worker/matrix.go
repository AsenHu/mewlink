package worker

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/rs/zerolog/log"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

func (w *Worker) FromMatrix(ctx context.Context, ev *event.Event) {
	// 检查消息是否是自己发的
	if ev.Sender == id.UserID(w.Config.Content.Matrix.Username) {
		return
	}
	// 从 database.RoomList 中获取房间相关信息
	info, found := w.DB.RoomList.GetRoomInfoByRoomID(ev.RoomID)
	log.Info().Bool("Found", found).
		Str("RoomID", ev.RoomID.String())
	if !found {
		return
	}

	log.Info().
		Str("RoomName", info.RoomName).
		Str("Text", ev.Content.AsMessage().Body)

	// 转发消息到 Telegram
	_, err := w.Telegram.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: info.ChatID,
		Text:   ev.Content.AsMessage().Body,
	})

	if err != nil {
		log.Error().Err(err).Msg("Failed to send message to Telegram")
		_, err = w.Matrix.SendText(ctx, ev.RoomID, err.Error())
		if err != nil {
			log.Error().Err(err).Msg("Failed to send message to Matrix")
		}
	}

	// 发送已读回执
	err = w.Matrix.SendReceipt(ctx, ev.RoomID, ev.ID, "m.read", nil)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to send receipt")
	}

	// 更新房间信息
	// 这不是很急的操作，所以使用已经用完的 goroutine 而不是新的 goroutine
	err = w.UpdateProfile(&info)
	if err != nil {
		log.Err(err).Msg("Failed to update profile")
	}
}
