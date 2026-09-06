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
	logDir, cleanup, err := exampleutil.PrepareLogDir("rolling")
	if err != nil {
		panic(err)
	}
	defer cleanup()

	handler, result, err := log.NewConfiguredHandler(context.Background(),
		log.WithConfigPath(exampleutil.ConfigPath("production-service.yml")),
	)
	if err != nil {
		panic(err)
	}
	defer handler.Close()

	logger, err := log.NewNativeLogger(handler, "goark.demo.sql")
	if err != nil {
		panic(err)
	}
	_ = logger.LogAttrs3(context.Background(), slog.LevelInfo, "query finished",
		slog.String("tenant", "tenant-a"),
		slog.Duration("elapsed", 12*time.Millisecond),
		slog.Int("rows", 3),
	)
	fmt.Println("source=" + string(result.Source))
	fmt.Println("logDir=" + logDir)
}
