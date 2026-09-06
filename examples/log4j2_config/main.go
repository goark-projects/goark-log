package main

import (
	"context"
	"fmt"
	"log/slog"

	"goark.dev/log"
	"goark.dev/log/examples/internal/exampleutil"
)

func main() {
	logDir, cleanup, err := exampleutil.PrepareLogDir("log4j2")
	if err != nil {
		panic(err)
	}
	defer cleanup()

	logger, handler, result, err := log.NewConfigured(context.Background(),
		log.WithConfigPath(exampleutil.ConfigPath("log4j2-service.xml")),
	)
	if err != nil {
		panic(err)
	}
	defer handler.Close()

	logger = log.WithName(logger, "goark.demo.http")
	logger.Info("tenant routed request",
		slog.String("tenant", "tenant-a"),
		slog.String("path", "/orders"),
		slog.String("password", "should-be-removed"),
	)
	logger.Info("GET /health")

	sqlLogger := log.WithName(logger, "goark.demo.sql")
	sqlLogger.Debug("debug SQL event", slog.String("tenant", "tenant-b"))

	fmt.Println("source=" + string(result.Source))
	fmt.Println("logDir=" + logDir)
}
