package main

import (
	"context"
	"log/slog"

	"goark.dev/log"
	"goark.dev/log/examples/internal/exampleutil"
)

func main() {
	handler, result, err := log.ConfigureDefault(context.Background(),
		log.WithConfigPath(exampleutil.ConfigPath("basic-console.yml")),
	)
	if err != nil {
		panic(err)
	}
	defer handler.Close()

	logger := log.WithName(slog.Default(), "goark.demo.console")
	logger.Info("console logging is ready", slog.String("source", string(result.Source)))
}
