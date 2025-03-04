package v1

import (
	"github.com/rs/zerolog"
	"maunium.net/go/mautrix/id"
)

type Content struct {
	LogLevel   zerolog.Level `json:"logLevel"`
	ServedUser string        `json:"servedUser"`
	Matrix     Matrix        `json:"matrix"`
	Telegram   Telegram      `json:"telegram"`
	DataBase   string        `json:"databasePath"`
	Version    uint8         `json:"version"`
}

type Matrix struct {
	BaseURL     string      `json:"baseURL"`
	Username    string      `json:"username"`
	Password    string      `json:"password"`
	DeviceID    id.DeviceID `json:"deviceID"`
	Token       string      `json:"token"`
	AsyncUpload bool        `json:"asyncUpload"`
}

type Telegram struct {
	Token   string  `json:"token"`
	Webhook Webhook `json:"webhook"`
}

type Webhook struct {
	Enable bool   `json:"enable"`
	URL    string `json:"url"`
	Port   int    `json:"listenPort"`
}

var DEFAULT_CONFIG = Content{
	LogLevel:   zerolog.InfoLevel,
	ServedUser: "@user:example.com",
	Matrix: Matrix{
		BaseURL:     "https://example.com",
		Username:    "@bot:example.com",
		Password:    "password",
		DeviceID:    "MEWLINK",
		AsyncUpload: true,
	},
	Telegram: Telegram{
		Webhook: Webhook{
			Enable: false,
		},
	},
	DataBase: "mewlink.db",
	Version:  1,
}
