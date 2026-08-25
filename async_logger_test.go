package goarklog

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAsyncLogger_whenCloseCalled_shouldDrainQueuedEvents(t *testing.T) {
	delegate := newRecordingAppender("delegate")
	handler, err := NewHandler(Options{
		Appenders: []Appender{delegate},
		Root:      RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"delegate"}},
		Async: AsyncLoggerOptions{
			Enabled:          true,
			QueueSize:        16,
			BatchSize:        4,
			OverflowStrategy: AsyncOverflowBlock,
		},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	logger := NewLogger(handler, "goark.async.logger")
	for index := 0; index < 12; index++ {
		logger.Info("queued event")
	}
	if err := handler.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if got := len(delegate.Events()); got != 12 {
		t.Fatalf("delegate event count = %d, want 12", got)
	}
}

func TestAsyncLogger_whenDropDebugQueueFull_shouldDropDebugOnly(t *testing.T) {
	delegate := newGatedAppender("delegate")
	t.Cleanup(delegate.releaseGate)
	handler, err := NewHandler(Options{
		Appenders: []Appender{delegate},
		Root:      RootLogger{Level: slog.LevelDebug, AppenderRefs: []string{"delegate"}},
		Async: AsyncLoggerOptions{
			Enabled:          true,
			QueueSize:        1,
			BatchSize:        1,
			OverflowStrategy: AsyncOverflowDropDebug,
		},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	logger := NewLogger(handler, "goark.async.logger")
	logger.Info("first")
	<-delegate.started
	logger.Info("second")
	logger.Debug("debug dropped")
	if handler.AsyncDropped() != 1 {
		t.Fatalf("AsyncDropped() = %d, want 1", handler.AsyncDropped())
	}
	delegate.releaseGate()
	if err := handler.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if delegate.Contains("debug dropped") {
		t.Fatalf("debug event should be dropped, events=%v", delegate.Events())
	}
}

func TestNewConfigured_whenAsyncLoggerYamlEnabled_shouldDrainFileOnClose(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "logs", "async-logger.log")
	configPath := filepath.Join(dir, "goark-log.yml")
	writeConfig(t, configPath, `
asyncLogger:
  enabled: true
  queueSize: 32
  batchSize: 8
  overflowStrategy: block
appenders:
  file:
    type: file
    fileName: "`+filepath.ToSlash(logPath)+`"
    layout:
      type: text
root:
  level: info
  appenderRefs: [file]
`)
	logger, handler, _, err := NewConfigured(context.Background(), WithConfigPath(configPath))
	if err != nil {
		t.Fatalf("NewConfigured() error = %v", err)
	}
	for index := 0; index < 10; index++ {
		logger.Info("yaml async logger")
	}
	if err := handler.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if got := strings.Count(string(content), "yaml async logger"); got != 10 {
		t.Fatalf("written event count = %d, want 10: %q", got, string(content))
	}
}

func TestNewConfigured_whenAsyncLoggerStrategyInvalid_shouldReject(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "goark-log.yml")
	writeConfig(t, configPath, `
asyncLogger:
  enabled: true
  overflowStrategy: never-block
appenders:
  console:
    type: console
root:
  level: info
  appenderRefs: [console]
`)
	_, _, err := NewConfiguredHandler(context.Background(), WithConfigPath(configPath))
	if err == nil {
		t.Fatalf("NewConfiguredHandler() should reject invalid async logger strategy")
	}
}
