package goarklog_test

import (
	"context"
	"log/slog"
	"testing"

	goarklog "goark.dev/log"
)

func TestLoadOptions_whenCustomizerConfigured_shouldApplyAfterDefaultLoading(t *testing.T) {
	called := false
	options, result, err := goarklog.LoadOptions(
		context.Background(),
		goarklog.WithDefaultConfigPaths(),
		goarklog.WithOptionsCustomizer(func(_ context.Context, current goarklog.Options, source *goarklog.ConfigResult) (goarklog.Options, error) {
			called = true
			if source.Source != goarklog.ConfigSourceDefault {
				t.Fatalf("source = %q, want default", source.Source)
			}
			current.Root.Level = slog.LevelDebug
			return current, nil
		}),
	)
	if err != nil {
		t.Fatalf("LoadOptions() error = %v", err)
	}
	defer closeAppenders(t, options.Appenders)
	if !called {
		t.Fatal("customizer was not called")
	}
	if result.Source != goarklog.ConfigSourceDefault || options.Root.Level != slog.LevelDebug {
		t.Fatalf("result/options = %#v/%#v", result, options.Root)
	}
}

func closeAppenders(t *testing.T, appenders []goarklog.Appender) {
	t.Helper()
	for _, appender := range appenders {
		if appender != nil {
			if err := appender.Close(); err != nil {
				t.Fatalf("close appender failed: %v", err)
			}
		}
	}
}
