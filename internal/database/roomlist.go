package database

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"maunium.net/go/mautrix/id"
)

type RoomList struct {
	Mutex            sync.RWMutex
	Path             string
	IsSyncedWithFile bool
	ByChatID         map[int64]RoomInfo
	ByRoomID         map[id.RoomID]RoomInfo
}

type RoomInfo struct {
	ChatID           int64
	RoomID           id.RoomID
	RoomName         string
	Avatar           string
	LastCheckProfile time.Time
}

func (rl *RoomList) GetRoomInfoByChatID(chatID int64) (RoomInfo, bool) {
	rl.Mutex.RLock()
	defer rl.Mutex.RUnlock()
	val, ok := rl.ByChatID[chatID]
	return val, ok
}

func (rl *RoomList) GetRoomInfoByRoomID(roomID id.RoomID) (RoomInfo, bool) {
	rl.Mutex.RLock()
	defer rl.Mutex.RUnlock()
	val, ok := rl.ByRoomID[roomID]
	return val, ok
}

func (rl *RoomList) Load() error {
	f, err := os.Open(rl.Path)
	if err != nil {
		return err
	}
	defer f.Close()
	dec := gob.NewDecoder(f)
	var list []RoomInfo
	err = dec.Decode(&list)
	if err != nil {
		return err
	}

	rl.Mutex.Lock()
	defer rl.Mutex.Unlock()
	for _, room := range list {
		rl.ByChatID[room.ChatID] = room
		rl.ByRoomID[room.RoomID] = room
	}
	return nil
}

func (rl *RoomList) Set(info RoomInfo) (err error) {
	rl.Mutex.Lock()
	defer rl.Mutex.Unlock()
	// 在房间已存在的情况下，更新房间信息
	if info.ChatID != 0 {
		if room, exists := rl.ByChatID[info.ChatID]; exists {
			if info.RoomID != "" {
				room.RoomID = info.RoomID
			}
			if info.RoomName != "" {
				room.RoomName = info.RoomName
			}
			if info.Avatar != "" {
				room.Avatar = info.Avatar
			}
			rl.ByChatID[info.ChatID] = room
		} else {
			rl.ByChatID[info.ChatID] = info
		}
	} else if info.RoomID != "" {
		if room, exists := rl.ByRoomID[info.RoomID]; exists {
			if info.ChatID != 0 {
				room.ChatID = info.ChatID
			}
			if info.RoomName != "" {
				room.RoomName = info.RoomName
			}
			if info.Avatar != "" {
				room.Avatar = info.Avatar
			}
			rl.ByRoomID[info.RoomID] = room
		} else {
			rl.ByRoomID[info.RoomID] = info
		}
	} else {
		return fmt.Errorf("no key provided")
	}

	// 如果 ChatID 和 RoomID 都有，说明这是重要的更新，需要立即保存
	if info.ChatID != 0 && info.RoomID != "" {
		err = rl.SaveNow()
		if err != nil {
			return
		}
	} else {
		rl.IsSyncedWithFile = false
	}
	return
}

// SaveNow 没有加锁，调用时请确保已经加锁
func (rl *RoomList) SaveNow() (err error) {
	// 创建目录
	err = os.MkdirAll(filepath.Dir(rl.Path), 0700)
	if err != nil {
		return err
	}

	f, err := os.Create(rl.Path)
	if err != nil {
		return err
	}
	defer f.Close()
	var list []RoomInfo
	for _, room := range rl.ByChatID {
		list = append(list, room)
	}
	enc := gob.NewEncoder(f)
	err = enc.Encode(list)
	rl.IsSyncedWithFile = true
	return
}

func (rl *RoomList) LazySave() {
	go func() {
		log.Info().Msg("LazySave started in RoomList")
		for {
			time.Sleep(5 * time.Minute)
			// 检查是否需要同步
			rl.Mutex.RLock()
			if rl.IsSyncedWithFile {
				rl.Mutex.RUnlock()
				log.Debug().Msg("LazySave skipped in RoomList, Because we are lazy")
				continue
			}
			rl.Mutex.RUnlock()

			// 同步
			rl.Mutex.Lock()
			err := rl.SaveNow()
			if err != nil {
				log.Warn().Err(err).Msg("LazySave failed in RoomList. Some not important data may be lost")
			} else {
				log.Info().Msg("LazySave succeeded in RoomList")
				rl.IsSyncedWithFile = true
			}
			rl.Mutex.Unlock()
		}
	}()
}
