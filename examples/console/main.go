package main

import (
	"log/slog"

	goarklog "goark.dev/goark-log"
)

func main() {
	logger, handler := goarklog.NewDefault()
	defer handler.Close()

	logger = goarklog.WithName(logger, "goark.boot")
	logger.Info("service started", slog.String("profile", "dev"))
}
