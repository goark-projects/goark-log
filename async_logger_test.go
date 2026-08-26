package goarklog

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	events := delegate.Events()
	if !events[len(events)-1].EndOfBatch {
		t.Fatalf("last async logger event should be marked EndOfBatch, events=%+v", events)
	}
}

func TestAsyncLogger_whenAppendAfterClose_shouldReject(t *testing.T) {
	delegate := newRecordingAppender("delegate")
	handler, err := NewHandler(Options{
		Appenders: []Appender{delegate},
		Root:      RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"delegate"}},
		Async: AsyncLoggerOptions{
			Enabled:          true,
			QueueSize:        4,
			BatchSize:        2,
			OverflowStrategy: AsyncOverflowBlock,
		},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	if err := handler.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	record := slog.NewRecord(fixedTestTime(), slog.LevelInfo, "after close", 0)
	if err := handler.Handle(context.Background(), record); err == nil {
		t.Fatalf("Handle() after Close should fail")
	}
}

func TestAsyncLogger_whenReloadChangesQueueSettings_shouldReject(t *testing.T) {
	delegate := newRecordingAppender("delegate")
	handler, err := NewHandler(Options{
		Appenders: []Appender{delegate},
		Root:      RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"delegate"}},
		Async: AsyncLoggerOptions{
			Enabled:          true,
			QueueSize:        8,
			BatchSize:        4,
			OverflowStrategy: AsyncOverflowBlock,
		},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	t.Cleanup(func() {
		_ = handler.Close()
	})

	replacement := newRecordingAppender("replacement")
	err = handler.Reload(Options{
		Appenders: []Appender{replacement},
		Root:      RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"replacement"}},
		Async: AsyncLoggerOptions{
			Enabled:          true,
			QueueSize:        16,
			BatchSize:        4,
			OverflowStrategy: AsyncOverflowBlock,
		},
	})
	if err == nil {
		t.Fatalf("Reload() should reject changed async logger queue settings")
	}
	if !strings.Contains(err.Error(), "async logger queue settings") {
		t.Fatalf("Reload() error = %v, want async logger queue settings error", err)
	}

	NewLogger(handler, "goark.async.logger").Info("old route still active")
	if err := handler.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if !delegate.Contains("old route still active") {
		t.Fatalf("old route should stay active, events=%v", delegate.Events())
	}
	if replacement.Contains("old route still active") {
		t.Fatalf("rejected reload should not install replacement route")
	}
}

func TestAsyncLogger_whenCloseWhileHandleBlocked_shouldUnblockHandle(t *testing.T) {
	delegate := newGatedAppender("delegate")
	t.Cleanup(delegate.releaseGate)
	handler, err := NewHandler(Options{
		Appenders: []Appender{delegate},
		Root:      RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"delegate"}},
		Async: AsyncLoggerOptions{
			Enabled:          true,
			QueueSize:        1,
			BatchSize:        1,
			OverflowStrategy: AsyncOverflowBlock,
		},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	if err := handler.Handle(context.Background(), slog.NewRecord(fixedTestTime(), slog.LevelInfo, "first", 0)); err != nil {
		t.Fatalf("Handle(first) error = %v", err)
	}
	<-delegate.started
	if err := handler.Handle(context.Background(), slog.NewRecord(fixedTestTime(), slog.LevelInfo, "second", 0)); err != nil {
		t.Fatalf("Handle(second) error = %v", err)
	}

	handleDone := make(chan error, 1)
	go func() {
		handleDone <- handler.Handle(context.Background(), slog.NewRecord(fixedTestTime(), slog.LevelInfo, "blocked", 0))
	}()
	closeDone := make(chan error, 1)
	go func() {
		closeDone <- handler.Close()
	}()

	select {
	case err := <-handleDone:
		if err == nil || !strings.Contains(err.Error(), "closed") {
			t.Fatalf("blocked Handle() error = %v, want closed error", err)
		}
	case <-time.After(time.Second):
		t.Fatalf("blocked Handle() was not unblocked by Close()")
	}

	delegate.releaseGate()
	select {
	case err := <-closeDone:
		if err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatalf("Close() did not finish after delegate was released")
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
  waitStrategy: yield
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

func TestNewConfigured_whenAsyncLoggerWaitStrategyInvalid_shouldReject(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "goark-log.yml")
	writeConfig(t, configPath, `
asyncLogger:
  enabled: true
  waitStrategy: park-forever
appenders:
  console:
    type: console
root:
  level: info
  appenderRefs: [console]
`)
	_, _, err := NewConfiguredHandler(context.Background(), WithConfigPath(configPath))
	if err == nil || !strings.Contains(err.Error(), "wait strategy") {
		t.Fatalf("NewConfiguredHandler() error = %v, want wait strategy rejection", err)
	}
}
