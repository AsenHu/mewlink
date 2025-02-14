package worker

import (
	"MewLink/internal/database"
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
		Str("Text", update.Message.Text)

	// 从 database.RoomList 中获取房间相关信息
	info, found := w.DB.RoomList.GetRoomInfoByChatID(update.Message.Chat.ID)
	if !found {
		log.Info().Msg("New chat, create room")
		go func() {
			_, err := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Welcome to MeowLink! 🐾\nFrom this message onwards, your message will be forwarded to Matrix friends.",
			})
			if err != nil {
				log.Error().Err(err).Msg("Failed to send message to Telegram")
			}
		}()
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
		go func() {
			info = database.RoomInfo{
				ChatID:   update.Message.Chat.ID,
				RoomID:   resp.RoomID,
				RoomName: username,
			}
			err = w.DB.RoomList.Set(info)
			if err != nil {
				w.SendErrToTG(ctx, update.Message.Chat.ID, err)
				log.Fatal().Err(err)
			}
		}()
		// 房间创建完后的第一条消息不转发，因为 Matrix HS 很可能还没准备好房间
		// 第一条消息通常也不是很重要，所以不转发也没关系
	} else {
		// 转发消息到 Matrix
		_, err := w.Matrix.SendText(ctx, info.RoomID, update.Message.Text)
		if err != nil {
			log.Error().Err(err).Msg("Failed to send message to Matrix")
			w.SendErrToTG(ctx, update.Message.Chat.ID, err)
		}
	}

	// 更新房间信息
	// 这不是很急的操作，所以使用已经用完的 goroutine 而不是新的 goroutine
	err := w.UpdateRoomNameByInput(&info, username)
	if err != nil {
		log.Err(err).Msg("Failed to update room name")
	}
	err = w.UpdateProfile(&info)
	if err != nil {
		log.Err(err).Msg("Failed to update profile")
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
