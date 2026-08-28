package main

import (
	"context"
	"log/slog"

	goarklog "goark.dev/log"
	"goark.dev/log/examples/internal/exampleutil"
)

func main() {
	handler, result, err := goarklog.ConfigureDefault(context.Background(),
		goarklog.WithConfigPath(exampleutil.ConfigPath("basic-console.yml")),
	)
	if err != nil {
		panic(err)
	}
	defer handler.Close()

	logger := goarklog.WithName(slog.Default(), "goark.demo.console")
	logger.Info("console logging is ready", slog.String("source", string(result.Source)))
}
