package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	goarklog "goark.dev/log"
	"goark.dev/log/examples/internal/exampleutil"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	logDir, cleanup, err := exampleutil.PrepareLogDir("production")
	if err != nil {
		return err
	}
	defer cleanup()

	loggerContext, result, err := goarklog.NewConfiguredLoggerContext(context.Background(),
		goarklog.WithConfigPath(exampleutil.ConfigPath("production-service.yml")),
	)
	if err != nil {
		return err
	}
	defer loggerContext.Close()

	requestLogger := loggerContext.Logger("goark.demo.http")
	auditLogger := loggerContext.Logger("goark.audit")
	nativeLogger, err := goarklog.NewNativeLogger(loggerContext.Handler(), "goark.demo.sql")
	if err != nil {
		return err
	}

	requestCtx := goarklog.WithContextAttrs(context.Background(),
		slog.String("trace_id", "trace-prod-1"),
		slog.String("tenant", "tenant-a"),
		slog.String("component", "api"),
	)
	requestCtx = goarklog.WithThreadName(requestCtx, "http-worker-1")
	requestCtx = goarklog.WithContextStack(requestCtx, "request", "checkout")
	requestCtx = goarklog.WithMarker(requestCtx, goarklog.NewMarker("HTTP"))

	requestLogger.InfoContext(requestCtx, "request completed",
		slog.String("method", "POST"),
		slog.String("path", "/orders"),
		slog.Int("status", 201),
		slog.Duration("elapsed", 18*time.Millisecond),
	)
	requestLogger.InfoContext(requestCtx, "GET /health")

	auditCtx := goarklog.WithMarker(requestCtx, goarklog.NewMarker("AUDIT"))
	auditLogger.InfoContext(auditCtx, "order approved",
		slog.String("principal", "alice"),
		slog.String("action", "approve"),
		slog.String("resource", "order:1001"),
	)

	err = errors.New("dial tcp 10.0.0.10:5432: connect: connection refused")
	_ = nativeLogger.AtError().
		WithContext(requestCtx).
		WithErrorStack(err).
		WithString("query", "select * from orders where id=?").
		Log("database request failed")

	fmt.Println("source=" + string(result.Source))
	fmt.Println("logDir=" + logDir)
	return nil
}
