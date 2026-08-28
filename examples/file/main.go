package main

import (
	"context"
	"fmt"
	"log/slog"

	goarklog "goark.dev/log"
	"goark.dev/log/examples/internal/exampleutil"
)

func main() {
	logDir, cleanup, err := exampleutil.PrepareLogDir("file")
	if err != nil {
		panic(err)
	}
	defer cleanup()

	logger, handler, result, err := goarklog.NewConfigured(context.Background(),
		goarklog.WithConfigPath(exampleutil.ConfigPath("complete-json-file.yml")),
	)
	if err != nil {
		panic(err)
	}
	defer handler.Close()

	logger = goarklog.WithName(logger, "goark.demo.file")
	logger.Info("complete JSON file stream is ready", slog.String("source", string(result.Source)))
	fmt.Println("logDir=" + logDir)
}
