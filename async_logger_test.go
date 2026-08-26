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

func TestAsyncLogger_whenErrorHandlerConfigured_shouldReceiveWriteFailure(t *testing.T) {
	errCh := make(chan error, 1)
	handler, err := NewHandler(Options{
		Appenders: []Appender{failingAppender{name: "broken"}},
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

func TestAsyncLogger_whenIncludeLocationEnabled_shouldCaptureNativeCaller(t *testing.T) {
	delegate := newRecordingAppender("delegate")
	handler, err := NewHandler(Options{
		Appenders: []Appender{delegate},
		Root:      RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"delegate"}},
		Async: AsyncLoggerOptions{
			Enabled:         true,
			QueueSize:       8,
			BatchSize:       2,
			IncludeLocation: true,
		},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	logger, err := NewNativeLogger(handler, "goark.async.location")
	if err != nil {
		t.Fatalf("NewNativeLogger() error = %v", err)
	}
	if err := logger.Info("with async location"); err != nil {
		t.Fatalf("Info() error = %v", err)
	}
	if err := handler.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	events := delegate.Events()
	if len(events) != 1 {
		t.Fatalf("event count = %d, want 1", len(events))
	}
	frame := callerFrameFromPC(events[0].PC)
	if !strings.Contains(frame.method, "TestAsyncLogger_whenIncludeLocationEnabled") {
		t.Fatalf("caller method = %q, want test method", frame.method)
	}
}

func TestAsyncLoggerOptions_whenAliasesUsed_shouldNormalize(t *testing.T) {
	options, err := normalizeAsyncLoggerOptions(AsyncLoggerOptions{
		Enabled:          true,
		QueueSize:        7,
		BatchSize:        16,
		OverflowStrategy: "discard-debug",
		WaitStrategy:     "busy-spin",
		IncludeLocation:  true,
	})
	if err != nil {
		t.Fatalf("normalizeAsyncLoggerOptions() error = %v", err)
	}
	if options.QueueSize != 8 {
		t.Fatalf("QueueSize = %d, want normalized power of two 8", options.QueueSize)
	}
	if options.BatchSize != 8 {
		t.Fatalf("BatchSize = %d, want capped queue size 8", options.BatchSize)
	}
	if options.OverflowStrategy != AsyncOverflowDropDebug {
		t.Fatalf("OverflowStrategy = %q, want %q", options.OverflowStrategy, AsyncOverflowDropDebug)
	}
	if options.WaitStrategy != AsyncWaitSpin {
		t.Fatalf("WaitStrategy = %q, want %q", options.WaitStrategy, AsyncWaitSpin)
	}
	if !options.IncludeLocation {
		t.Fatalf("IncludeLocation = false, want true")
	}
}

func TestNewConfigured_whenAsyncLoggerPropertiesConfigured_shouldBuildOptions(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "goark-log.properties")
	writeConfig(t, configPath, `
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
	if handler.async == nil {
		t.Fatalf("async logger is nil")
	}
	if handler.async.options.QueueSize != 8 || handler.async.options.BatchSize != 8 {
		t.Fatalf("async options = %+v, want normalized queue/batch size 8", handler.async.options)
	}
	if handler.async.options.OverflowStrategy != AsyncOverflowDrop {
		t.Fatalf("OverflowStrategy = %q, want %q", handler.async.options.OverflowStrategy, AsyncOverflowDrop)
	}
	if handler.async.options.WaitStrategy != AsyncWaitBlock {
		t.Fatalf("WaitStrategy = %q, want %q", handler.async.options.WaitStrategy, AsyncWaitBlock)
	}
	if !handler.async.options.IncludeLocation {
		t.Fatalf("IncludeLocation = false, want true")
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
