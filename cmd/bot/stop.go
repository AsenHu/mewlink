package main

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
)

// 接受系统信号来停止程序

func stopBySig(cancel func()) {
	// 信号处理
	// 1. 创建一个信号通道
	sigChan := make(chan os.Signal, 1)
	// 2. 通知监听的信号
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		// 3. 阻塞等待信号
		sig := <-sigChan
		log.Info().Msgf("Received signal: %v", sig)
		go func() {
			sig := <-sigChan
			log.Info().Msgf("Received second signal: %v, force exiting", sig)
			os.Exit(1)
		}()
		// 4. 关闭程序
		cancel()
	}()
}

func cliDead(cancel func(), dbWg *sync.WaitGroup) {
	cancel()
	time.Sleep(5 * time.Minute)
	log.Info().Msg("Timeout reached, force exiting")
	dbWg.Done()
	os.Exit(1)
}
