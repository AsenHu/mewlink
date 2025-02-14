package worker

import (
	"context"
	"strconv"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog/log"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/id"
)

func (w *Worker) FromTelegram(ctx context.Context, b *bot.Bot, update *models.Update) {
	username := GetUserName(update)
	log.Info().
		Str("Username", username).
		Str("Text", update.Message.Text).
		Msg("Received message from Telegram")

	// 从 kv 中获取 Matrix 房间 ID
	roomID, found := w.KVStore.GetRoomID(update.Message.Chat.ID)
	if !found {
		log.Info().Msg("New chat, create room")
		resp, err := w.Matrix.CreateRoom(ctx, &mautrix.ReqCreateRoom{
			Name: username,
			Invite: []id.UserID{
				id.UserID(w.Config.Content.ServedUser),
			},
			IsDirect: true,
			Preset:   "private_chat",
		})
		if err != nil {
			log.Error().Err(err).Msg("Failed to create room")
			w.SendErrToTG(ctx, update.Message.Chat.ID, err)
			return
		}
		_, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Welcome to MeowLink! 🐾\nYour message has been forwarded to your Matrix friend. Please be patient as they reply.",
		})
		if err != nil {
			log.Error().Err(err).Msg("Failed to send message to Telegram")
		}
		roomID = resp.RoomID
		err = w.KVStore.Set(update.Message.Chat.ID, roomID)
		if err != nil {
			log.Error().Err(err).Msg("Failed to save KV store")
		}
	}

	// 转发消息到 Matrix
	_, err := w.Matrix.SendText(ctx, roomID, update.Message.Text)
	if err != nil {
		log.Error().Err(err).Msg("Failed to send message to Matrix")
		w.SendErrToTG(ctx, update.Message.Chat.ID, err)
	}
}

func GetUserName(update *models.Update) string {
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

func (w *Worker) SendErrToTG(ctx context.Context, chatID int64, err error) {
	_, err = w.Telegram.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   err.Error(),
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to send message to Telegram")
	}
	_, err = w.Telegram.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   "An error occurred on the Matrix side. Please try to contact the user through other means.",
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to send message to Telegram")
	}
}
