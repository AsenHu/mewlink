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
	"github.com/rs/zerolog"
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

	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// 准备上下文
	ctx, cancel := context.WithCancel(context.Background())
	var dbWg, wg sync.WaitGroup

	// 接受信号
	stopBySig(cancel)

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
	// RoomList 数据库
	roomlist := &database.RoomList{
		Path:     cfg.Content.DataBase.RoomList,
		ByChatID: make(map[int64]database.RoomInfo),
		ByRoomID: make(map[id.RoomID]database.RoomInfo),
	}
	if err := roomlist.Load(); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Warn().Msg("RoomList file not found, starting with empty store")
		} else {
			log.Fatal().Err(err).Msg("Failed to load RoomList")
			return
		}
	}
	roomlist.IsSyncedWithFile = true
	roomlist.LazySave(ctx, &dbWg)

	// EventList 数据库
	events := &database.EventList{
		Path:      cfg.Content.DataBase.EventList,
		ByEventID: make(map[id.EventID]database.EventInfo),
	}
	if err := events.Load(); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Warn().Msg("EventList file not found, starting with empty store")
		} else {
			log.Fatal().Err(err).Msg("Failed to load EventList")
			return
		}
	}
	events.IsSyncedWithFile = true
	events.LazySave(ctx, &dbWg)

	// 载入数据库
	database := &database.DataBase{
		RoomList:  roomlist,
		EventList: events,
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
		wg.Add(1)
		go func() {
			worker.FromMatrix(ctx, ev)
			wg.Done()
		}()
	})
	matrix.Syncer = syncer

	// 准备 Telegram 客户端
	opts := []bot.Option{
		bot.WithDefaultHandler(func(ctx context.Context, bot *bot.Bot, update *models.Update) {
			wg.Add(1)
			go func() {
				worker.FromTelegram(ctx, &wg, bot, update)
				wg.Done()
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

	// 启动客户端必须在准备好 Worker 之后，而不是在不同的 goroutine 中

	wg.Add(1)
	go func() {
		defer wg.Done()
		// 启动 Matrix 客户端

		// 检查是否需要登陆
		if cfg.Content.Matrix.Token == "" {
			log.Info().Msg("Login Matrix...")
			if err := matrixLogin(ctx, matrix, &cfg.Content.Matrix); err != nil {
				log.Fatal().Err(err).Msg("Failed to login")
				return
			}
			cfg.Save()
		} else {
			log.Info().Msg("Get Token from configuration, skip login")
		}

		for {
			if err := matrix.SyncWithContext(ctx); err != nil {
				if errors.Is(err, context.Canceled) {
					log.Info().Msg("Context canceled, exiting Matrix sync loop")
					return
				}
				if errors.Is(err, mautrix.MUnknownToken) {
					log.Warn().Msg("Token expired, relogin...")
					matrix.AccessToken = ""
					if err := matrixLogin(ctx, matrix, &cfg.Content.Matrix); err != nil {
						log.Fatal().Err(err).Msg("Failed to relogin")
					}
					cfg.Save()
					continue
				}
				if errors.Is(err, mautrix.MInvalidParam) {
					log.Warn().Msg("Username format error, relogin...")
					if err := matrixLogin(ctx, matrix, &cfg.Content.Matrix); err != nil {
						log.Fatal().Err(err).Msg("Failed to relogin")
					}
					cfg.Save()
					continue
				}
				log.Warn().Err(err).Msg("Sync failed")
				return
			}
		}
	}()

	// 检查 Telegram 客户端
	wg.Add(1)
	go func() {
		telegramLogin(ctx, telegram, &wg)
		wg.Done()
	}()
	// 启动 Telegram 客户端
	wg.Add(1)
	go func() {
		defer wg.Done()
		telegram.Start(ctx)
	}()

	dbWg.Wait()
	wg.Wait()
	log.Info().Msg("Shutting down gracefully")
}
