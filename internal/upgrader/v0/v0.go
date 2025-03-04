package v0

import (
	"time"

	"maunium.net/go/mautrix/id"
)

type Content struct {
	ServedUser string   `json:"servedUser"`
	Matrix     Matrix   `json:"matrix"`
	Telegram   Telegram `json:"telegram"`
	DataBase   DataBase `json:"databasePath"`
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

type DataBase struct {
	RoomList  string `json:"roomList"`
	EventList string `json:"eventList"`
}

type RoomInfo struct {
	ChatID           int64
	RoomID           id.RoomID
	RoomName         string
	Avatar           string
	LastCheckProfile time.Time
}
