package main

import (
	"MewLink/internal/config"
	"context"
	"sync"

	"github.com/go-telegram/bot"
	"github.com/rs/zerolog/log"
	"maunium.net/go/mautrix"
)

func matrixLogin(ctx context.Context, client *mautrix.Client, cfg *config.Matrix) (err error) {
	// 开始登陆
	resp, err := client.Login(ctx, &mautrix.ReqLogin{
		Type: "m.login.password",
		Identifier: mautrix.UserIdentifier{
			Type: "m.id.user",
			User: cfg.Username,
		},
		Password:                 cfg.Password,
		Token:                    cfg.Token,
		DeviceID:                 cfg.DeviceID,
		InitialDeviceDisplayName: "MewLink",

		StoreCredentials:   true,
		StoreHomeserverURL: true,
	})
	if err != nil {
		return
	}

	cfg.BaseURL = resp.WellKnown.Homeserver.BaseURL
	cfg.Username = resp.UserID.String()
	cfg.DeviceID = resp.DeviceID
	cfg.Token = resp.AccessToken
	return
}

func telegramLogin(ctx context.Context, b *bot.Bot, wg *sync.WaitGroup) {
	me, err := b.GetMe(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get bot info")
		wg.Done()
	}
	// 整理用户名
	username := me.Username
	// 1. 尝试拼接 FirstName 和 LastName, 如果有的话
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
}
