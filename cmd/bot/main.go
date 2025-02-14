package main

import (
	"MewLink/internal/config"
	"MewLink/internal/database"
	"MewLink/internal/worker"
	"context"
	"errors"
	"flag"
	"os"
	"sync"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog/log"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

var CONFIG_PATH string

func init() {
	flag.StringVar(&CONFIG_PATH, "c", "config.json", "Path to the configuration file")
}

func main() {
	flag.Parse()

	// 准备配置
	cfg := config.NewConfig(CONFIG_PATH)
	if err := cfg.Load(); err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
		return
	}
	if cfg.Content.ServedUser == "@user:example.com" {
		log.Fatal().Msg("Please edit the configuration file first")
		return
	}

	// 准备数据库
	roomlist := &database.RoomList{
		Path:     cfg.Content.DataBase.RoomList,
		ByChatID: make(map[int64]database.RoomInfo),
		ByRoomID: make(map[id.RoomID]database.RoomInfo),
	}
	if err := roomlist.Load(); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Warn().Msg("KV store file not found, starting with empty store")
		} else {
			log.Fatal().Err(err).Msg("Failed to load KV store")
			return
		}
	}
	roomlist.IsSyncedWithFile = true
	roomlist.LazySave()
	database := &database.DataBase{
		RoomList: roomlist,
	}

	// 准备 Worker
	worker := &worker.Worker{
		DB:     database,
		Config: &cfg,
	}

	// 准备 Matrix 客户端
	matrix, err := mautrix.NewClient(cfg.Content.Matrix.BaseURL, id.UserID(cfg.Content.Matrix.Username), cfg.Content.Matrix.Token)
	if err != nil {
		log.Fatal().Err(err)
		return
	}
	worker.Matrix = matrix

	syncer := mautrix.NewDefaultSyncer()
	syncer.OnEventType(event.EventMessage, func(ctx context.Context, ev *event.Event) {
		go func() {
			worker.FromMatrix(ctx, ev)
		}()
	})
	matrix.Syncer = syncer

	// 准备 Telegram 客户端
	opts := []bot.Option{
		bot.WithDefaultHandler(func(ctx context.Context, bot *bot.Bot, update *models.Update) {
			go func() {
				worker.FromTelegram(ctx, bot, update)
			}()
		}),
		bot.WithSkipGetMe(),
	}
	telegram, err := bot.New(cfg.Content.Telegram.Token, opts...)
	if err != nil {
		log.Fatal().Err(err)
		return
	}
	worker.Telegram = telegram

	var wg sync.WaitGroup
	wg.Add(1)

	// 启动客户端必须在准备好 Worker 之后，而不是在不同的 goroutine 中

	go func() {
		defer wg.Done()
		// 启动 Matrix 客户端

		// 检查是否需要登陆
		if cfg.Content.Matrix.Token == "" {
			log.Info().Msg("Login Matrix...")
			if err := matrixLogin(matrix, &cfg.Content.Matrix); err != nil {
				log.Fatal().Err(err).Msg("Failed to login")
				return
			}
			cfg.Save()
		} else {
			log.Info().Msg("Get Token from configuration, skip login")
		}

		for {
			if err := matrix.Sync(); err != nil {
				if errors.Is(err, mautrix.MUnknownToken) {
					log.Warn().Msg("Token expired, relogin...")
					matrix.AccessToken = ""
					if err := matrixLogin(matrix, &cfg.Content.Matrix); err != nil {
						log.Fatal().Err(err).Msg("Failed to relogin")
					}
					cfg.Save()
					continue
				}
				if errors.Is(err, mautrix.MInvalidParam) {
					log.Warn().Msg("Username format error, relogin...")
					if err := matrixLogin(matrix, &cfg.Content.Matrix); err != nil {
						log.Fatal().Err(err).Msg("Failed to relogin")
					}
					cfg.Save()
					continue
				}
				log.Fatal().Err(err).Msg("Sync failed")
				return
			}
		}
	}()

	// 检查 Telegram 客户端
	go func() {
		telegramLogin(telegram, &wg)
	}()
	// 启动 Telegram 客户端
	go func() {
		defer wg.Done()
		telegram.Start(context.Background())
	}()

	wg.Wait()
}
