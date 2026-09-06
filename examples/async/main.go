package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"goark.dev/log"
	"goark.dev/log/examples/internal/exampleutil"
)

func main() {
	logDir, cleanup, err := exampleutil.PrepareLogDir("async")
	if err != nil {
		panic(err)
	}
	defer cleanup()

	logger, handler, _, err := log.NewConfigured(context.Background(),
		log.WithConfigPath(exampleutil.ConfigPath("async-failover.yml")),
	)
	if err != nil {
		panic(err)
	}
	defer handler.Close()

	logger = log.WithName(logger, "goark.demo.async")
	for index := 0; index < 5; index++ {
		logger.Info(
			"queued event",
			slog.Int("index", index),
			slog.Duration("elapsed", time.Duration(index)*time.Millisecond),
		)
	}
	fmt.Println("dropped=" + fmt.Sprint(handler.AsyncDropped()))
	fmt.Println("failed=" + fmt.Sprint(handler.AsyncFailed()))
	fmt.Println("logDir=" + logDir)
}
