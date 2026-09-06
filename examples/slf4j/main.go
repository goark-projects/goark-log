package main

import (
	"context"
	"log/slog"
	"time"

	"goark.dev/log"
	"goark.dev/log/examples/internal/exampleutil"
)

func main() {
	handler, _, err := log.NewConfiguredHandler(context.Background(),
		log.WithConfigPath(exampleutil.ConfigPath("container-json.yml")),
	)
	if err != nil {
		panic(err)
	}
	defer handler.Close()

	logger, err := log.NewNativeLogger(handler, "goark.demo.slf4j",
		log.WithLoggerMessageFactory(log.ParameterizedMessageFactory{}),
	)
	if err != nil {
		panic(err)
	}

	ctx := log.WithContextAttrs(context.Background(), slog.String("trace_id", "trace-slf4j-1"))
	_ = logger.AtInfo().
		WithContext(ctx).
		WithString("user", "alice").
		WithInt("status", 200).
		Logf("user {} finished request in {}", "alice", 8*time.Millisecond)

	slogLogger := logger.Slog().WithGroup("request")
	slogLogger.InfoContext(
		ctx,
		"standard slog interop",
		slog.String("method", "GET"),
		slog.Int("status", 200),
	)
}
