package worker

import (
	"MewLink/internal/database"
	"context"
	"fmt"
	"time"

	"maunium.net/go/mautrix/event"
)

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

func (w *Worker) UpdateProfile(info *database.RoomInfo) error {
	if info.ChatID == 0 {
		return fmt.Errorf("no Chat ID provided")
	}
	if info.RoomID == "" {
		return fmt.Errorf("no Room ID provided")
	}
	// 如果现在的时间减去上次更新的时间小于 30 分钟，就不更新
	if time.Since(info.LastCheckProfile) < 30*time.Minute {
		return nil
	}
	// 等待实现
	return nil
}
