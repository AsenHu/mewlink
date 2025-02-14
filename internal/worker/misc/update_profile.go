package misc

import (
	"context"

	"maunium.net/go/mautrix"
)

/*
func (w *Worker) UpdateRoomNameByInput(info *database.RoomInfo, name string) (err error) {
	if info.RoomName == name {
		return nil
	}

	// 准备结构体
	content := event.RoomNameEventContent{
		Name: name,
	}
	// 更新房间名
	_, err = w.Matrix.SendMessageEvent(context.Background(), info.RoomID, event.StateRoomName, content)
	if err != nil {
		return
	}
	info.RoomName = name
	return
}
*/

func UpdateProfile(ctx context.Context, index []byte, cli *mautrix.Client) (err error) {
	// 等待实现
	return
}
