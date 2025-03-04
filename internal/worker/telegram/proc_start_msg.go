package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/AsenHu/mewlink/internal/types"
	"github.com/AsenHu/mewlink/internal/worker/misc"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog/log"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/id"
)

/*
关于这里的锁的说明

首先，这里的锁有两个关键的目的
1. 锁要保证 ChatID 从数据库读取到 index 之后，不会被其他 goroutine 修改
2. ChatID RoomID index 的修改操作是一个原子操作，需要保证在修改的时候，其他 goroutine 不能读取到错误的数据

所以，这里的锁应该这样使用
1. 在读取 ChatID 之后，立即写锁定 ChatID（尽管 ChatID 有可能不会修改，但我们在查询的时候不知道，因此只能写锁定）
2. 整理需要的信息，比如要写入的数据的内容
3. 决定开始写入数据后，立刻写锁定 index 和 RoomID
4. 必须要确保所有的锁都拿到后，才能开始写入数据，并且必须要在数据全部写入后，才能释放锁
5. 一定要注意锁的顺序，不要出现死锁
*/

func (w *TelegramWorker) procStartMsg(ctx context.Context, update *models.Update) (index []byte) {
	username := getUserName(update)
	// 检查房间是否已经存在
	// 1. 写锁定 ChatID
	chatLock, _ := w.DataBase.RoomList.ChatIDMutex.LoadOrStore(update.Message.Chat.ID, &sync.RWMutex{})
	chatLock.(*sync.RWMutex).Lock()

	// 2. 检查 ChatID 是否存在
	index, err := w.DataBase.RoomList.GetIndexByChatID(update.Message.Chat.ID)
	if err != nil {
		chatLock.(*sync.RWMutex).Unlock()
		log.Err(err).Msg("Failed to get index by ChatID")
		w.sendErrToTG(ctx, update.Message.Chat.ID, err)
		return
	}
	var info *types.RoomInfo
	if index != nil {
		// 说明这个房间已经存在,chatLock 就可以释放了
		chatLock.(*sync.RWMutex).Unlock()

		log.Info().
			Int64("ChatID", update.Message.Chat.ID).
			Str("User", username).
			Msg("Received a processed /start message")

		// 锁定 index
		// 这里使用读锁定，因为如果 index 存在，这个房间应该已经创建了，这里不应该有写操作
		indexLock, _ := w.DataBase.RoomList.RoomInfoBucket.IndexMutex.LoadOrStore(string(index), &sync.RWMutex{})
		indexLock.(*sync.RWMutex).RLock()

		// 检查 RoomID 是否存在
		info, err = w.DataBase.RoomList.GetRoomInfoByIndex(index)
		indexLock.(*sync.RWMutex).RUnlock() // 数据读出来之后就可以释放 index 锁了
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
				Int64("ChatID", update.Message.Chat.ID).
				Str("Index", fmt.Sprintf("%X", index)).
				Str("RoomInfo", string(jsonInfo)).
				Msg("RoomInfo not valid, this should not happen, database corrupted")
			w.StopProc()
			w.sendErrToTG(ctx, update.Message.Chat.ID, err)
			return
		}

		_, err = w.Telegram.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "You have sent the start message before, please don't send it again",
		})
		if err != nil {
			log.Error().Err(err).Msg("Failed to send message to Telegram.")
		}
		return
	}

	// 3. 说明这个房间不存在，开始创建
	log.Info().
		Int64("ChatID", update.Message.Chat.ID).
		Str("User", username).
		Msg("Received a /start message")
	// 发送欢迎消息
	w.WaitGroup.Add(1)
	go func() {
		defer w.WaitGroup.Done()
		_, err := w.Telegram.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Welcome to MeowLink! 🐾\nFrom this message onwards, your message will be forwarded to Matrix friends.",
		})
		if err != nil {
			log.Error().Err(err).Msg("Failed to send message to Telegram")
		}
	}()
	// 创建房间
	resp, err := w.Matrix.CreateRoom(ctx, &mautrix.ReqCreateRoom{
		Name: username,
		Invite: []id.UserID{
			id.UserID(w.Config.Content.ServedUser),
		},
		IsDirect: true,
		Preset:   "private_chat",
	})
	if err != nil {
		w.sendErrToTG(ctx, update.Message.Chat.ID, err)
		return
	}
	// 创建房间信息
	info = &types.RoomInfo{
		ChatID:   update.Message.Chat.ID,
		RoomID:   string(resp.RoomID),
		RoomName: username,
	}

	// 信息准备好了，准备锁
	// 获取未使用的 index 并锁定
	index, indexLock, err := w.DataBase.RoomList.RoomInfoBucket.FindUnusedKey()
	if err != nil {
		log.Err(err).Msg("Failed to find unused key")
		w.sendErrToTG(ctx, update.Message.Chat.ID, err)
		return
	}
	defer indexLock.Unlock()
	// 锁定 RoomID
	roomLock, _ := w.DataBase.RoomList.RoomIDMutex.LoadOrStore(string(resp.RoomID), &sync.RWMutex{})
	roomLock.(*sync.RWMutex).Lock()
	defer roomLock.(*sync.RWMutex).Unlock()

	// 三把锁都拿到了，可以开始写入数据了
	// 存入数据库
	if err = w.DataBase.RoomList.SetRoomInfoByIndex(index, info); err != nil {
		log.Err(err).Msg("Failed to set RoomInfo by index")
		w.sendErrToTG(ctx, update.Message.Chat.ID, err)
		return
	}
	if err = w.DataBase.RoomList.SetChatIDIndex(update.Message.Chat.ID, index); err != nil {
		log.Err(err).Msg("Failed to set ChatID index")
		w.sendErrToTG(ctx, update.Message.Chat.ID, err)
		return
	}
	if err = w.DataBase.RoomList.SetRoomIDIndex(resp.RoomID, index); err != nil {
		log.Err(err).Msg("Failed to set RoomID index")
		w.sendErrToTG(ctx, update.Message.Chat.ID, err)
		return
	}

	// chatLock 不是 defer 的，所以这里要手动释放
	chatLock.(*sync.RWMutex).Unlock()

	return
}
