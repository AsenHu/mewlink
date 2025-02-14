package main

import (
	"MewLink/internal/config"
	"MewLink/internal/kv"
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

	// 准备 KV 存储
	kv := kv.NewKVStore(cfg.Content.DataBase)
	if err := kv.Load(cfg.Content.DataBase); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Warn().Msg("KV store file not found, starting with empty store")
		} else {
			log.Fatal().Err(err).Msg("Failed to load KV store")
			return
		}
	}

	// 准备 Worker
	worker := &worker.Worker{
		KVStore: kv,
		Config:  &cfg,
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		// 启动 Matrix 客户端
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

		for {
			if err := matrix.Sync(); err != nil {
				if errors.Is(err, mautrix.MUnknownToken) {
					log.Warn().Msg("Token expired, relogin...")
					matrix.AccessToken = ""
					if err := login(matrix, &cfg.Content.Matrix); err != nil {
						log.Fatal().Err(err).Msg("Failed to relogin")
					}
					cfg.Save()
					continue
				}
				if errors.Is(err, mautrix.MInvalidParam) {
					log.Warn().Msg("Username format error, relogin...")
					if err := login(matrix, &cfg.Content.Matrix); err != nil {
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

	// 启动 Telegram 客户端
	go func() {
		defer wg.Done()
		opts := []bot.Option{
			bot.WithDefaultHandler(func(ctx context.Context, bot *bot.Bot, update *models.Update) {
				go func() {
					worker.FromTelegram(ctx, bot, update)
				}()
			}),
		}
		telegram, err := bot.New(cfg.Content.Telegram.Token, opts...)
		if err != nil {
			log.Fatal().Err(err)
			return
		}
		worker.Telegram = telegram
		telegram.Start(context.Background())
	}()

	wg.Wait()
}
