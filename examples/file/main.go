package main

import (
	"context"
	"fmt"
	"log/slog"

	"goark.dev/log"
	"goark.dev/log/examples/internal/exampleutil"
)

func main() {
	logDir, cleanup, err := exampleutil.PrepareLogDir("file")
	if err != nil {
		panic(err)
	}
	defer cleanup()

	logger, handler, result, err := log.NewConfigured(context.Background(),
		log.WithConfigPath(exampleutil.ConfigPath("complete-json-file.yml")),
	)
	if err != nil {
		panic(err)
	}
	defer handler.Close()

	logger = log.WithName(logger, "goark.demo.file")
	logger.Info("complete JSON file stream is ready", slog.String("source", string(result.Source)))
	fmt.Println("logDir=" + logDir)
}
