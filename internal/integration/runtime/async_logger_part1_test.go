package integration

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "goark.dev/log/internal/testsupport"
)

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

	if err := handler.Handle(
		context.Background(),
		slog.NewRecord(FixedTestTime(), slog.LevelInfo, "first", 0),
	); err != nil {
		t.Fatalf("Handle(first) error = %v", err)
	}
	<-delegate.started
	if err := handler.Handle(
		context.Background(),
		slog.NewRecord(FixedTestTime(), slog.LevelInfo, "second", 0),
	); err != nil {
		t.Fatalf("Handle(second) error = %v", err)
	}

	handleDone := make(chan error, 1)
	go func() {
		record := slog.NewRecord(FixedTestTime(), slog.LevelInfo, "blocked", 0)
		handleDone <- handler.Handle(context.Background(), record)
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

func TestAsyncLogger_whenErrorHandlerConfigured_shouldReceiveWriteFailure(t *testing.T) {
	errCh := make(chan error, 1)
	handler, err := NewHandler(Options{
		Appenders: []Appender{NewFailingAppender("broken")},
		Root:      RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"broken"}},
		Async: AsyncLoggerOptions{
			Enabled:          true,
			QueueSize:        8,
			BatchSize:        2,
			OverflowStrategy: AsyncOverflowBlock,
			ErrorHandler: AsyncErrorHandlerFunc(func(_ context.Context, err error, event Event) {
				if event.Message == "broken event" {
					errCh <- err
				}
			}),
		},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	logger := NewLogger(handler, "goark.async.logger")
	logger.Info("broken event")
	if err := handler.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	select {
	case err := <-errCh:
		if err == nil || !strings.Contains(err.Error(), "forced failure") {
			t.Fatalf("async logger error = %v, want forced failure", err)
		}
	default:
		t.Fatalf("async logger error handler was not called")
	}
	if handler.AsyncFailed() != 1 {
		t.Fatalf("AsyncFailed() = %d, want 1", handler.AsyncFailed())
	}
}

func TestNewConfigured_whenAsyncLoggerPropertiesConfigured_shouldBuildOptions(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "goark-log.properties")
	WriteConfig(t, configPath, `
asyncLogger.enabled = true
asyncLogger.queueSize = 7
asyncLogger.batchSize = 16
asyncLogger.overflowStrategy = discard
asyncLogger.waitStrategy = timeout
asyncLogger.includeLocation = true
appender.console.type = console
rootLogger.level = info
rootLogger.appenderRefs = console
`)
	_, handler, _, err := NewConfigured(context.Background(), WithConfigPath(configPath))
	if err != nil {
		t.Fatalf("NewConfigured() error = %v", err)
	}
	defer handler.Close()
	asyncOptions := handler.AsyncOptions()
	if !asyncOptions.Enabled {
		t.Fatalf("async logger is disabled")
	}
	if asyncOptions.QueueSize != 8 || asyncOptions.BatchSize != 8 {
		t.Fatalf("async options = %+v, want normalized queue/batch size 8", asyncOptions)
	}
	if asyncOptions.OverflowStrategy != AsyncOverflowDrop {
		t.Fatalf("OverflowStrategy = %q, want %q", asyncOptions.OverflowStrategy, AsyncOverflowDrop)
	}
	if asyncOptions.WaitStrategy != AsyncWaitBlock {
		t.Fatalf("WaitStrategy = %q, want %q", asyncOptions.WaitStrategy, AsyncWaitBlock)
	}
	if !asyncOptions.IncludeLocation {
		t.Fatalf("IncludeLocation = false, want true")
	}
}

func TestAsyncLogger_whenCloseCalled_shouldDrainQueuedEvents(t *testing.T) {
	delegate := NewRecordingAppender("delegate")
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

func TestNewConfigured_whenAsyncLoggerStrategyInvalid_shouldReject(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "goark-log.yml")
	WriteConfig(t, configPath, `
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
	WriteConfig(t, configPath, `
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
