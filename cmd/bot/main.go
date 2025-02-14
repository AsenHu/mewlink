package main

import (
	"context"
	"errors"
	"flag"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/AsenHu/mewlink/internal/config"
	"github.com/AsenHu/mewlink/internal/database"
	"github.com/AsenHu/mewlink/internal/worker"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"maunium.net/go/mautrix"
)

var CONFIG_PATH string

func init() {
	flag.StringVar(&CONFIG_PATH, "c", "config.json", "Path to the configuration file")
}

func main() {
	// 强制关闭
	/*
		go func() {
			forceClose := make(chan os.Signal, 1)
			signal.Notify(forceClose, syscall.SIGINT)
			for i := 0; i < 8; i++ {
				<-forceClose
			}
			log.Fatal().Msg("Force closed")
		}()
		//*/

	flag.Parse()

	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// 准备配置
	cfg := config.NewConfig(CONFIG_PATH)
	err := cfg.Load()
	if errors.Is(err, os.ErrNotExist) {
		if err := cfg.Save(); err != nil {
			log.Fatal().Err(err).Msg("Failed to create configuration file")
		}
		log.Fatal().Msg("Please edit the configuration file first")
	} else if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}
	if cfg.Content.ServedUser == "@user:example.com" {
		log.Fatal().Msg("Please edit the configuration file first")
	}

	// 日志
	//cfg.Content.LogLevel = zerolog.DebugLevel
	zerolog.SetGlobalLevel(cfg.Content.LogLevel)

	// 准备上下文
	// 一般来说，调用 syncCancel 后，不会再处理新的事件，旧的事件会继续处理，这足以保证正常关闭
	// 但是，如果有必要，可以在某些地方调用 cancel，以做到比较快速的关闭
	ctx, cancel := context.WithCancel(context.Background())
	syncCtx, syncCancel := context.WithCancel(context.Background())
	// 这个是给 Worker 用的，如果 Worker 遇到错误，将会调用 errCancel，主程序会在收到信号后准备关闭操作
	workerCtx, errCancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	// 信号处理
	// 1. 创建一个信号通道
	sigintChan := make(chan os.Signal, 6)
	sigtermChan := make(chan os.Signal, 1)
	// 2. 通知监听的信号
	signal.Notify(sigintChan, syscall.SIGINT)
	signal.Notify(sigtermChan, syscall.SIGTERM)

	// 准备数据库
	// 之后的代码中，不可以再使用 Fatal，因为这可能损坏数据库
	db, err := database.NewDataBase(cfg.Content.DataBase)
	if err != nil {
		log.Error().Err(err).Msg("Failed to load/init database")
		if err = db.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close database")
		}
		os.Exit(1)
	}

	// 准备 Worker
	worker := &worker.Worker{
		DataBase:  db,
		Config:    cfg,
		WaitGroup: &wg,
		Context:   ctx,
		StopProc:  errCancel,
	}

	// 准备 Matrix 客户端
	setMatrixClient(worker)

	// 准备 Telegram 客户端
	setTelegramClient(worker)

	// 启动客户端必须在准备好 Worker 之后，而不是在不同的 goroutine 中
	wg.Add(2)

	// 启动 Matrix 客户端
	go func() {
		defer wg.Done()
		// 启动 Matrix 客户端

		// 检查是否需要登陆
		if worker.Matrix.AccessToken == "" {
			if err := matrixLogin(worker); err != nil {
				log.Error().Err(err).Msg("Failed to login")
				errCancel()
				return
			}
			cfg.Save()
		}

		for {
			if err := worker.Matrix.SyncWithContext(syncCtx); err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				if errors.Is(err, mautrix.MUnknownToken) {
					log.Warn().Msg("Token expired, relogin...")
					worker.Matrix.AccessToken = ""
					if err := matrixLogin(worker); err != nil {
						log.Error().Err(err).Msg("Failed to relogin")
						errCancel()
						return
					}
					cfg.Save()
					continue
				}
				if errors.Is(err, mautrix.MInvalidParam) {
					log.Warn().Msg("Username format error, relogin...")
					if err := matrixLogin(worker); err != nil {
						log.Error().Err(err).Msg("Failed to relogin")
						errCancel()
						return
					}
					cfg.Save()
					continue
				}
				log.Error().Err(err).Msg("Sync failed")
				errCancel()
				return
			}
		}
	}()

	// 检查 Telegram 客户端
	checkTelegramClient(worker)

	// 启动 Telegram 客户端
	go func() {
		worker.Telegram.Start(syncCtx)
		wg.Done()
	}()

	log.Info().Msg("MewLink is running")

	/* 处理关闭信号
	   有两种正常关闭方式：
	   1. 用户按下 Ctrl+C (SIGINT)
	   2. 用户发送 SIGTERM
	   和两种异常关闭方式：
	   1. 用户按下 Ctrl+C (SIGINT) 之后多次按下 Ctrl+C
	   2. Worker 发送了 StopProc 信号
	*/

	exitCode := 0
	select {
	case <-sigintChan:
		log.Info().Msg("Received SIGINT, stopping")
		syncCancel()
		// 处理 Ctrl+C 之后多次按下 Ctrl+C
		go func() {
			<-sigintChan
			log.Warn().Msg("Do not press Ctrl+C again")
			<-sigintChan
			log.Warn().Msg("Next time press Ctrl+C will stop the unfinished process")
			<-sigintChan
			exitCode = 1
			cancel()
			log.Warn().Msg("Stop signal sent")
			<-sigintChan
			log.Warn().Msg("Force stop may result in database corruption")
			<-sigintChan
			log.Warn().Msg("Next time press Ctrl+C will force close database")
			log.Warn().Msg("It may cause unexpected data to be included in the database")
			log.Warn().Msg("WE WARNED YOU, IF YOU REALLY WANT TO STOP, PRESS Ctrl+C AGAIN")
			<-sigintChan
			go func() {
				err := db.Close()
				if err != nil {
					log.Fatal().Err(err).Msg("Failed to close database")
				}
				log.Warn().Msg("Database force closed")
				os.Exit(1)
			}()
			log.Warn().Msg("Trying to force close database, please wait for a while")
			log.Warn().Msg("Next time press Ctrl+C will force close program")
			log.Warn().Msg("It may cause you to be unable to open the database again")
			os.Exit(1)
		}()
	case <-sigtermChan:
		log.Info().Msg("Received SIGTERM, stopping")
		syncCancel()
	case <-workerCtx.Done():
		log.Warn().Msg("Worker wants to stop")
		exitCode = 1
		syncCancel()
	}

	// 处理 goroutine 死活不退出的情况
	go func() {
		// 等待 1 分钟
		<-time.After(1 * time.Minute)
		exitCode = 1
		log.Warn().Msg("Some process is still running, try to force stop all goroutines")
		cancel()
		// 等待 2 分钟
		<-time.After(2 * time.Minute)
		log.Warn().Msg("Some process is still running, try to force close database")
		go func() {
			// 等待 4 分钟
			<-time.After(4 * time.Minute)
			log.Fatal().Msg("Force close database failed")
		}()
		err := db.Close()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to close database")
		}
		log.Warn().Msg("Database force closed")
		os.Exit(1)
	}()

	wg.Wait()
	err = db.Close()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to close database")
	}
	if exitCode != 0 {
		log.Warn().
			Int("ExitCode", exitCode).
			Msg("MewLink stopped with error")
		os.Exit(exitCode)
	}
	log.Info().Msg("MewLink stopped gracefully")
}
