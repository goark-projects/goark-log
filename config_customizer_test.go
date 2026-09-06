package log_test

import (
	"context"
	"log/slog"
	"testing"

	"goark.dev/log"
)

func TestLoadOptions_whenCustomizerConfigured_shouldApplyAfterDefaultLoading(t *testing.T) {
	called := false
	options, result, err := log.LoadOptions(
		context.Background(),
		log.WithDefaultConfigPaths(),
		log.WithOptionsCustomizer(func(_ context.Context, current log.Options, source *log.ConfigResult) (log.Options, error) {
			called = true
			if source.Source != log.ConfigSourceDefault {
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
	if result.Source != log.ConfigSourceDefault || options.Root.Level != slog.LevelDebug {
		t.Fatalf("result/options = %#v/%#v", result, options.Root)
	}
}

func closeAppenders(t *testing.T, appenders []log.Appender) {
	t.Helper()
	for _, appender := range appenders {
		if appender != nil {
			if err := appender.Close(); err != nil {
				t.Fatalf("close appender failed: %v", err)
			}
		}
	}
}
