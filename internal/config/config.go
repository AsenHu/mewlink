package config

import (
	"encoding/json"
	"os"

	config "github.com/AsenHu/mewlink/internal/upgrader/v1"
	"github.com/rs/zerolog"
)

type Config struct {
	Path    string
	Content config.Content
}

func NewConfig(path string) *Config {
	return &Config{
		Path: path,
		Content: config.Content{
			LogLevel:   zerolog.InfoLevel,
			ServedUser: "@user:example.com",
			Matrix: config.Matrix{
				BaseURL:     "https://example.com",
				Username:    "@bot:example.com",
				Password:    "password",
				DeviceID:    "MEWLINK",
				AsyncUpload: true,
			},
			Telegram: config.Telegram{
				Webhook: config.Webhook{
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
	return c.unmarshal(buffer)
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
