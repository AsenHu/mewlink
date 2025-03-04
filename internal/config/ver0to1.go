package config

import (
	"encoding/json"
	"os"
	"strconv"

	v0 "github.com/AsenHu/mewlink/internal/upgrader/v0"
	v1 "github.com/AsenHu/mewlink/internal/upgrader/v1"
	"github.com/rs/zerolog/log"
	"go.etcd.io/bbolt"
	"google.golang.org/protobuf/proto"
)

func ver0to1(v0Data v0.Content) (v1Data v1.Content, err error) {
	// 转换配置文件
	v1Data = v1.DEFAULT_CONFIG

	v1Data.ServedUser = v0Data.ServedUser
	v1Data.DataBase = v0Data.DataBase.RoomList

	v1Data.Matrix.BaseURL = v0Data.Matrix.BaseURL
	v1Data.Matrix.Username = v0Data.Matrix.Username
	v1Data.Matrix.Password = v0Data.Matrix.Password
	v1Data.Matrix.DeviceID = v0Data.Matrix.DeviceID
	v1Data.Matrix.Token = v0Data.Matrix.Token
	v1Data.Matrix.AsyncUpload = v0Data.Matrix.AsyncUpload

	v1Data.Telegram.Token = v0Data.Telegram.Token

	v1Data.Telegram.Webhook.Enable = v0Data.Telegram.Webhook.Enable
	v1Data.Telegram.Webhook.URL = v0Data.Telegram.Webhook.URL
	v1Data.Telegram.Webhook.Port = v0Data.Telegram.Webhook.Port

	// 转换数据库
	// 检查是否有数据库文件
	if v0Data.DataBase.RoomList == "" {
		log.Debug().Msg("No database file")
		// 没有数据库文件，直接返回
		return
	}
	// 我们只需要转换 RoomList，其他的不需要
	// 给原始数据库文件加上 .bak 后缀
	// 以免出现问题
	if err = os.Rename(v0Data.DataBase.RoomList, v0Data.DataBase.RoomList+".bak"); err != nil {
		// 如果文件不存在，直接返回
		if os.IsNotExist(err) {
			return v1Data, nil
		}
		// 如果有其他错误，返回错误
		return
	}
	// 读取原始数据库文件
	rawData, err := os.ReadFile(v0Data.DataBase.RoomList + ".bak")
	if err != nil {
		return
	}
	// json 解析
	var roomList []v0.RoomInfo
	if err = json.Unmarshal(rawData, &roomList); err != nil {
		return
	}
	// 检查是否有数据
	if len(roomList) == 0 {
		return
	}
	// 打开新数据库文件
	db, err := bbolt.Open(v1Data.DataBase, 0600, nil)
	if err != nil {
		return
	}
	defer db.Close()
	// 打开数据库
	err = db.Update(func(tx *bbolt.Tx) (err error) {
		// 新建 RoomInfo bucket
		roomInfoBucket, err := tx.CreateBucket([]byte{1})
		if err != nil {
			return
		}

		// 遍历 roomList
		for index, room := range roomList {
			newRoomInfo := v1.RoomInfo{}
			// 转换 RoomInfo
			newRoomInfo.ChatID = room.ChatID
			newRoomInfo.RoomID = string(room.RoomID)
			// 剩下的不重要，不转了
			// 序列化 RoomInfo
			var roomInfoBytes []byte
			roomInfoBytes, err = proto.Marshal(&newRoomInfo)
			if err != nil {
				break
			}
			// 写入数据库
			log.Debug().
				Int("index", index).
				Msg("Writing RoomInfo")
			err = roomInfoBucket.Put([]byte(strconv.Itoa(index)), roomInfoBytes)
			if err != nil {
				break
			}
		}
		return
	})
	return
}
