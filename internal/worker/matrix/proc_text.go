package matrix

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/AsenHu/mewlink/internal/worker/misc"
	"github.com/go-telegram/bot"
	"github.com/rs/zerolog/log"
	"maunium.net/go/mautrix/event"
)

func (w *MatrixWorker) procText(ctx context.Context, ev *event.Event) (index []byte) {
	// 检查是否是空消息
	if ev.Content.AsMessage().Body == "" {
		log.Warn().Str("EventID", ev.ID.String()).Msg("Empty message")
		return
	}

	// 获取房间信息的 index
	// 上读锁定 RoomID
	roomLock, _ := w.DataBase.RoomList.RoomIDMutex.LoadOrStore(ev.RoomID, &sync.RWMutex{})
	roomLock.(*sync.RWMutex).RLock()
	index, err := w.DataBase.RoomList.GetIndexByRoomID(ev.RoomID)
	roomLock.(*sync.RWMutex).RUnlock()
	if err != nil {
		log.Err(err).Msg("Failed to get index by RoomID")
		w.sendErrToMatrix(ctx, ev.RoomID, err)
		return
	}
	if index == nil {
		log.Debug().
			Str("EventID", ev.ID.String()).
			Str("RoomID", ev.RoomID.String()).
			Msg("Room not found")
		return
	}

	// 获取房间信息
	indexLock, _ := w.DataBase.RoomList.RoomInfoBucket.IndexMutex.LoadOrStore(string(index), &sync.RWMutex{})
	indexLock.(*sync.RWMutex).RLock()
	info, err := w.DataBase.RoomList.GetRoomInfoByIndex(index)
	indexLock.(*sync.RWMutex).RUnlock()
	if err != nil {
		log.Err(err).Msg("Failed to get RoomInfo by index")
		w.sendErrToMatrix(ctx, ev.RoomID, err)
		return
	}

	// 检查房间信息是否合法
	if !misc.IsGoodRoomInfo(info) {
		err = fmt.Errorf("RoomInfo not valid, this should not happen, database corrupted")
		// json 化 RoomInfo
		jsonInfo, _ := json.Marshal(info)
		log.Error().
			Str("RoomID", ev.RoomID.String()).
			Str("Index", fmt.Sprintf("%X", index)).
			Str("RoomInfo", string(jsonInfo)).
			Msg("RoomInfo not valid, this should not happen, database corrupted")
		w.sendErrToMatrix(ctx, ev.RoomID, err)
		w.StopProc()
		return
	}

	log.Info().
		Str("SendTo", info.GetRoomName()).
		Str("Msg", ev.Content.AsMessage().Body).
		Msg("Msg from MX")

	// 转发消息到 Telegram
	_, err = w.Telegram.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: info.ChatID,
		Text:   ev.Content.AsMessage().Body,
	})
	if err != nil {
		log.Err(err).Msg("Failed to send message to Telegram")
		w.sendErrToMatrix(ctx, ev.RoomID, err)
		return
	}

	// 保存消息
	if err = w.DataBase.EventList.Set(ev.ID); err != nil {
		log.Err(err).Msg("Failed to set event")
		w.sendErrToMatrix(ctx, ev.RoomID, err)
	}

	return
}
