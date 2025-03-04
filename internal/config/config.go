package config

import (
	"encoding/json"
	"os"

	"github.com/rs/zerolog"
	"maunium.net/go/mautrix/id"
)

type Config struct {
	Path    string
	Content Content
}

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

func NewConfig(path string) *Config {
	return &Config{
		Path: path,
		Content: Content{
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
		},
	}
}

func (c *Config) Load() (err error) {
	// 读取文件
	buffer, err := os.ReadFile(c.Path)
	if err != nil {
		return
	}
	// 解析文件
	return json.Unmarshal(buffer, &c.Content)
}

func (c *Config) Save() (err error) {
	// 序列化文件
	buffer, err := json.MarshalIndent(c.Content, "", "  ")
	if err != nil {
		return
	}
	// 写入文件
	err = os.WriteFile(c.Path, buffer, 0600)
	if err != nil {
		return
	}
	return
}
