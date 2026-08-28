package main

import (
	"context"
	"log/slog"
	"time"

	goarklog "goark.dev/log"
	"goark.dev/log/examples/internal/exampleutil"
)

func main() {
	handler, _, err := goarklog.NewConfiguredHandler(context.Background(),
		goarklog.WithConfigPath(exampleutil.ConfigPath("container-json.yml")),
	)
	if err != nil {
		panic(err)
	}
	defer handler.Close()

	logger, err := goarklog.NewNativeLogger(handler, "goark.demo.slf4j",
		goarklog.WithLoggerMessageFactory(goarklog.ParameterizedMessageFactory{}),
	)
	if err != nil {
		panic(err)
	}

	ctx := goarklog.WithContextAttrs(context.Background(), slog.String("trace_id", "trace-slf4j-1"))
	_ = logger.AtInfo().
		WithContext(ctx).
		WithString("user", "alice").
		WithInt("status", 200).
		Logf("user {} finished request in {}", "alice", 8*time.Millisecond)

	slogLogger := logger.Slog().WithGroup("request")
	slogLogger.InfoContext(ctx, "standard slog interop", slog.String("method", "GET"), slog.Int("status", 200))
}
