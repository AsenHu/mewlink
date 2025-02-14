package matrix

import (
	"context"
	"encoding/json"

	"github.com/AsenHu/mewlink/internal/worker"
	"github.com/AsenHu/mewlink/internal/worker/misc"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type MatrixWorker struct {
	*worker.Worker
}

func (w MatrixWorker) FromMatrix(_ context.Context, ev *event.Event) {
	w.WaitGroup.Add(1)
	go func() {
		defer w.WaitGroup.Done()
		// log.Debug().Str("EventID", ev.ID.String()).Msg("Received message from Matrix")
		// 检查消息是否是被服务的用户发送的
		if ev.Sender != id.UserID(w.Config.Content.ServedUser) {
			log.Debug().Str("EventID", ev.ID.String()).Msg("Message not sent by served user")
			return
		}

		// 检查消息是否处理过
		exi, err := w.DataBase.EventList.IsExi(ev.ID)
		if err != nil {
			log.Err(err).Msg("Failed to check if event exists")
			w.sendErrToMatrix(w.Context, ev.RoomID, err)
			return
		}

		if exi {
			log.Debug().Str("EventID", ev.ID.String()).Msg("Event already exists")
			return
		}

		// 确定消息类型，然后调用相应的处理函数
		// 1. 如果是普通消息，调用 `procText`
		// 2. 如果是其他消息，直接返回
		var index []byte
		switch ev.Content.AsMessage().MsgType {
		case event.MsgText:
			index = w.procText(w.Context, ev)
		default:
			if w.Config.Content.LogLevel == zerolog.DebugLevel {
				jsonEvent, _ := json.Marshal(ev)
				log.Debug().
					Str("Event", string(jsonEvent)).
					Msg("Unsupported message type")
			}
			return
		}

		// 杂项操作

		// 发送已读回执
		if err = w.Matrix.SendReceipt(w.Context, ev.RoomID, ev.ID, "m.read", nil); err != nil {
			log.Warn().Err(err).Msg("Failed to send receipt")
		}

		// 更新房间信息
		if err = misc.UpdateProfile(w.Context, index, w.Matrix); err != nil {
			log.Warn().Err(err).Msg("Failed to update profile")
		}
	}()
}

func (w *MatrixWorker) sendErrToMatrix(ctx context.Context, roomID id.RoomID, err error) {
	message := err.Error() + "\nAn error occurred, please check the logs"
	_, err = w.Matrix.SendText(ctx, roomID, message)
	if err != nil {
		log.Error().Err(err).Msg("Failed to send message to Matrix")
	}
}
