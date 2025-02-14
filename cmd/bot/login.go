package main

import (
	"MewLink/internal/config"
	"context"
	"time"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/id"
)

func login(client *mautrix.Client, cfg *config.Matrix) (err error) {
	// 准备登陆
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// 开始登陆
	resp, err := client.Login(ctx, &mautrix.ReqLogin{
		Type: "m.login.password",
		Identifier: mautrix.UserIdentifier{
			Type: "m.id.user",
			User: cfg.Username,
		},
		Password:                 cfg.Password,
		Token:                    cfg.Token,
		DeviceID:                 id.DeviceID(cfg.DeviceID),
		InitialDeviceDisplayName: "MewLink",

		StoreCredentials:   true,
		StoreHomeserverURL: true,
	})
	if err != nil {
		return
	}

	cfg.BaseURL = resp.WellKnown.Homeserver.BaseURL
	cfg.Username = resp.UserID.String()
	cfg.DeviceID = resp.DeviceID.String()
	cfg.Token = resp.AccessToken
	return
}
