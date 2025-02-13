package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	Path    string
	Content Content
}

type Content struct {
	ServedUser string `json:"servedUser"`
	Matrix   Matrix   `json:"matrix"`
	Telegram Telegram `json:"telegram"`
	DataBase string   `json:"database"`
}

type Matrix struct {
	BaseURL     string `json:"baseURL"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	DeviceID    string `json:"deviceID"`
	Token       string `json:"token"`
	AsyncUpload bool   `json:"asyncUpload"`
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

func NewConfig(path string) Config {
	return Config{
		Path: path,
		Content: Content{
			ServedUser: "@user:example.com",
			Matrix: Matrix{
				BaseURL:     "https://example.com",
				Username:    "@bot:example.com",
				Password:    "password",
				AsyncUpload: true,
			},
			Telegram: Telegram{
				Webhook: Webhook{
					Enable: false,
				},
			},
			DataBase: "MeowLink.db",
		},
	}
}

func (c *Config) Load() (err error) {
	// 检查文件是否存在
	_, err = os.Stat(c.Path)
	if os.IsNotExist(err) {
		// 文件不存在则创建
		err = c.Save()
		if err != nil {
			return
		}
	}

	// 读取文件
	buffer, err := os.ReadFile(c.Path)
	if err != nil {
		return
	}
	// 解析文件
	err = json.Unmarshal(buffer, &c.Content)
	if err != nil {
		return
	}
	return
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
