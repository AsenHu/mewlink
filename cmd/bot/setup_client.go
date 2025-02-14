package main

import (
	"github.com/AsenHu/mewlink/internal/worker"
	"github.com/AsenHu/mewlink/internal/worker/matrix"
	"github.com/AsenHu/mewlink/internal/worker/telegram"
	"github.com/go-telegram/bot"
	"github.com/rs/zerolog/log"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

func setMatrixClient(w *worker.Worker) {
	// 创建 Matrix 客户端
	var err error
	w.Matrix, err = mautrix.NewClient(w.Config.Content.Matrix.BaseURL, id.UserID(w.Config.Content.Matrix.Username), w.Config.Content.Matrix.Token)
	if err != nil {
		log.Err(err).Msg("Failed to create Matrix client")
		w.StopProc()
		return
	}

	// 设置回调函数
	syncer := mautrix.NewDefaultSyncer()
	syncer.OnEventType(event.EventMessage, matrix.MatrixWorker{Worker: w}.FromMatrix)
	w.Matrix.Syncer = syncer
}

func setTelegramClient(w *worker.Worker) {
	// 设置回调函数
	opts := []bot.Option{
		bot.WithDefaultHandler(telegram.TelegramWorker{Worker: w}.FromTelegram),
		bot.WithSkipGetMe(),
	}

	// 创建 Telegram 客户端
	var err error
	w.Telegram, err = bot.New(w.Config.Content.Telegram.Token, opts...)
	if err != nil {
		log.Error().Err(err)
		w.StopProc()
		return
	}
}

func checkTelegramClient(w *worker.Worker) {
	// 检查客户端
	w.WaitGroup.Add(1)
	go func() {
		defer w.WaitGroup.Done()
		me, err := w.Telegram.GetMe(w.Context)
		if err != nil {
			log.Error().Err(err).Msg("Failed to get bot info")
			w.StopProc()
			return
		}
		// 整理用户名
		username := me.Username
		// 尝试拼接 FirstName 和 LastName, 如果有的话
		if me.FirstName != "" || me.LastName != "" {
			if me.FirstName == "" {
				username = me.LastName
			}
			if me.LastName == "" {
				username = me.FirstName
			}
			if me.FirstName != "" && me.LastName != "" {
				username = me.FirstName + " " + me.LastName
			}
		}
		log.Info().
			Msg("Hello, I'm " + username + ". Let your friends send messages to @" + me.Username + " on Telegram")
	}()
}

func matrixLogin(w *worker.Worker) (err error) {
	// 开始登陆
	_, err = w.Matrix.Login(w.Context, &mautrix.ReqLogin{
		Type: "m.login.password",
		Identifier: mautrix.UserIdentifier{
			Type: "m.id.user",
			User: w.Config.Content.Matrix.Username,
		},
		Password:                 w.Config.Content.Matrix.Password,
		Token:                    w.Config.Content.Matrix.Token,
		DeviceID:                 w.Config.Content.Matrix.DeviceID,
		InitialDeviceDisplayName: "MewLink",

		StoreCredentials:   true,
		StoreHomeserverURL: true,
	})
	if err != nil {
		return
	}

	w.Config.Content.Matrix.BaseURL = w.Matrix.HomeserverURL.String()
	w.Config.Content.Matrix.Username = w.Matrix.UserID.String()
	w.Config.Content.Matrix.DeviceID = w.Matrix.DeviceID
	w.Config.Content.Matrix.Token = w.Matrix.AccessToken
	return
}
