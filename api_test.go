package goarklog_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	goarklog "goark.dev/goark-log"
)

func TestPublicAPICompile(t *testing.T) {
	var out bytes.Buffer
	handler, err := goarklog.NewHandler(goarklog.Options{
		Appenders: []goarklog.Appender{
			goarklog.NewConsoleAppender(
				goarklog.WithConsoleWriter(&out),
				goarklog.WithConsoleLayout(goarklog.TextLayout{}),
			),
		},
		Root: goarklog.RootLogger{
			Level:        slog.LevelInfo,
			AppenderRefs: []string{"console"},
		},
		Loggers: []goarklog.LoggerRule{
			{
				Name:  "goark.orm",
				Level: levelPtr(slog.LevelDebug),
			},
		},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	defer handler.Close()

	logger := goarklog.NewLogger(handler, "goark.orm")
	logger.Debug("sql prepared")
	logger.Info("sql done")

	var _ func(context.Context, ...goarklog.ConfigLoadOption) (*slog.Logger, *goarklog.Handler, *goarklog.ConfigResult, error) = goarklog.NewConfigured
	var _ func(context.Context, ...goarklog.ConfigLoadOption) (*goarklog.Handler, *goarklog.ConfigResult, error) = goarklog.NewConfiguredHandler
	var _ func(context.Context, ...goarklog.ConfigLoadOption) (*goarklog.Handler, *goarklog.ConfigResult, error) = goarklog.ConfigureDefault
	var _ func(context.Context, ...goarklog.ConfigLoadOption) (goarklog.Options, *goarklog.ConfigResult, error) = goarklog.LoadOptions
	var _ goarklog.ConfigLoadOption = goarklog.WithConfigPath("conf/goark-log.yml")
}

func levelPtr(level slog.Level) *slog.Level {
	return &level
}
