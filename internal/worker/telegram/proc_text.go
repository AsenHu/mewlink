package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/AsenHu/mewlink/internal/worker/misc"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog/log"
	"maunium.net/go/mautrix/id"
)

func (w *TelegramWorker) procText(ctx context.Context, update *models.Update) (index []byte) {
	username := getUserName(update)

	// 获取房间信息的 index
	// 上读锁定 ChatID
	// 虽然 GetIndexByChatID 是原子操作，但这里的锁是为了保证其他 goroutine 在同时修改多个桶的时候，它不会读到错误的数据
	chatLock, _ := w.DataBase.RoomList.ChatIDMutex.LoadOrStore(update.Message.Chat.ID, &sync.RWMutex{})
	chatLock.(*sync.RWMutex).RLock()
	index, err := w.DataBase.RoomList.GetIndexByChatID(update.Message.Chat.ID)
	chatLock.(*sync.RWMutex).RUnlock()
	if err != nil {
		log.Err(err).Msg("Failed to get index by ChatID")
		w.sendErrToTG(ctx, update.Message.Chat.ID, err)
		return
	}
	// 检查房间是否存在
	if index == nil {
		log.Warn().Int64("ChatID", update.Message.Chat.ID).Str("User", username).Msg("Room not found")
		_, err = w.Telegram.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Please resend `/start`",
		})
		if err != nil {
			log.Err(err).Msg("Failed to send message to Telegram")
		}
		return
	}
	// 获取房间信息
	// 这里的锁同上
	indexLock, _ := w.DataBase.RoomList.RoomInfoBucket.IndexMutex.LoadOrStore(string(index), &sync.RWMutex{})
	indexLock.(*sync.RWMutex).RLock()
	info, err := w.DataBase.RoomList.GetRoomInfoByIndex(index)
	indexLock.(*sync.RWMutex).RUnlock()
	if err != nil {
		log.Err(err).Msg("Failed to get RoomInfo by index")
		w.sendErrToTG(ctx, update.Message.Chat.ID, err)
		return
	}
	// 检查房间信息是否合法
	if !misc.IsGoodRoomInfo(info) {
		err = fmt.Errorf("RoomInfo not valid, this should not happen, database corrupted")
		// json 化 RoomInfo
		jsonInfo, _ := json.Marshal(info)
		log.Error().
			Str("RoomID", id.RoomID(info.GetRoomID()).String()).
			Str("Index", fmt.Sprintf("%X", index)).
			Str("RoomInfo", string(jsonInfo)).
			Msg("RoomInfo not valid, this should not happen, database corrupted")
		w.StopProc()
		w.sendErrToTG(ctx, update.Message.Chat.ID, err)
		return
	}

	log.Info().
		Str("User", username).
		Str("Msg", update.Message.Text).
		Msg("Msg from TG")

	// 转发消息到 Matrix
	_, err = w.Matrix.SendText(ctx, id.RoomID(info.GetRoomID()), update.Message.Text)
	if err != nil {
		log.Err(err).Msg("Failed to send message to Matrix")
		w.sendErrToTG(ctx, update.Message.Chat.ID, err)
		return
	}

	return
}
