package database

import (
	"sync"

	"github.com/AsenHu/mewlink/internal/types"
	"github.com/rs/zerolog/log"
	"go.etcd.io/bbolt"
	"google.golang.org/protobuf/proto"
	"maunium.net/go/mautrix/id"
)

type RoomList struct {
	RoomInfoBucket Bucket
	chatIDIndex    Bucket
	roomIDIndex    Bucket
	ChatIDMutex    sync.Map
	RoomIDMutex    sync.Map
}

// 新建 RoomList

func newRoomList(db *bbolt.DB) (rl *RoomList, err error) {
	rl = &RoomList{
		RoomInfoBucket: Bucket{
			database: db,
			bucket:   []byte{bucketRoomListRoomInfo},
			keyLen:   1,
		},
		chatIDIndex: Bucket{
			database: db,
			bucket:   []byte{bucketRoomListChatIDIndex},
			keyLen:   1,
		},
		roomIDIndex: Bucket{
			database: db,
			bucket:   []byte{bucketRoomListRoomIDIndex},
			keyLen:   1,
		},
	}

	// 检查 RoomInfo bucket 是否存在，如果不存在需要创建

	ext, err := rl.RoomInfoBucket.Exists()
	if err != nil {
		return
	}
	if !ext {
		log.Warn().Msg("RoomInfo bucket not found, creating")
		err = rl.RoomInfoBucket.Create()
		if err != nil {
			return
		}
	}

	// 检查两个 index bucket 是否存在，如果不存在需要从 RoomInfo bucket 重建

	chatIDExt, err := rl.chatIDIndex.Exists()
	if err != nil {
		return
	}
	roomIDExt, err := rl.roomIDIndex.Exists()
	if err != nil {
		return
	}
	if !chatIDExt || !roomIDExt {
		log.Warn().Msg("Index bucket not found, rebuilding")
		err = rl.rebuildIndex()
		if err != nil {
			return
		}
	}

	return
}

// 重建 index bucket
func (rl *RoomList) rebuildIndex() (err error) {
	// 确保两个 index bucket 不存在
	chatIDExt, err := rl.chatIDIndex.Exists()
	if err != nil {
		return
	}
	roomIDExt, err := rl.roomIDIndex.Exists()
	if err != nil {
		return
	}
	if chatIDExt {
		err = rl.chatIDIndex.DeleteBucket()
		if err != nil {
			return
		}
	}
	if roomIDExt {
		err = rl.roomIDIndex.DeleteBucket()
		if err != nil {
			return
		}
	}
	err = rl.chatIDIndex.Create()
	if err != nil {
		return
	}
	err = rl.roomIDIndex.Create()
	if err != nil {
		return
	}

	// 从 RoomInfo bucket 重建 index bucket
	err = rl.RoomInfoBucket.database.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(rl.RoomInfoBucket.bucket).ForEach(func(k, v []byte) error {
			var roomInfo types.RoomInfo
			err := proto.Unmarshal(v, &roomInfo)
			if err != nil {
				return err
			}

			log.Debug().
				Int64("ChatID", roomInfo.ChatID).
				Str("RoomID", roomInfo.RoomID).
				Str("Index", string(k)).
				Msg("Rebuilding index")
			// 重建 ChatID index
			err = tx.Bucket([]byte{bucketRoomListChatIDIndex}).Put(chatID2Bytes(roomInfo.ChatID), k)
			if err != nil {
				return err
			}

			// 重建 RoomID index
			err = tx.Bucket([]byte{bucketRoomListRoomIDIndex}).Put([]byte(roomInfo.RoomID), k)
			if err != nil {
				return err
			}

			return nil
		})
	})
	return
}

// 使用 ChatID 查询 Index

func (rl *RoomList) GetIndexByChatID(chatID int64) (index []byte, err error) {
	// 从 ChatID 查询 Index
	index, err = rl.chatIDIndex.Get(chatID2Bytes(chatID))
	return
}

// 使用 RoomID 查询 Index

func (rl *RoomList) GetIndexByRoomID(roomID id.RoomID) (index []byte, err error) {
	// 从 RoomID 查询 Index
	return rl.roomIDIndex.Get([]byte(roomID))
}

// 使用 Index 查询 RoomInfo

func (rl *RoomList) GetRoomInfoByIndex(index []byte) (roomInfo *types.RoomInfo, err error) {
	// 从 Index 查询 RoomInfo
	data, err := rl.RoomInfoBucket.Get(index)
	if err != nil {
		return
	}
	if data == nil {
		return
	}

	roomInfo = &types.RoomInfo{}

	// 反序列化 RoomInfo
	err = proto.Unmarshal(data, roomInfo)
	return
}

// 设置 RoomInfo 到 Index

func (rl *RoomList) SetRoomInfoByIndex(index []byte, roomInfo *types.RoomInfo) (err error) {
	data, err := proto.Marshal(roomInfo)
	if err != nil {
		return
	}
	return rl.RoomInfoBucket.Put(index, data)
}

// 设置 ChatID 到 Index

func (rl *RoomList) SetChatIDIndex(chatID int64, index []byte) (err error) {
	return rl.chatIDIndex.Put(chatID2Bytes(chatID), index)
}

// 设置 RoomID 到 Index

func (rl *RoomList) SetRoomIDIndex(roomID id.RoomID, index []byte) (err error) {
	return rl.roomIDIndex.Put([]byte(roomID), index)
}
