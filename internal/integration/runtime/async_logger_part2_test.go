package integration

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"goark.dev/log/internal/callsite"
	. "goark.dev/log/internal/testsupport"
)

func TestAsyncLogger_whenReloadChangesQueueSettings_shouldReject(t *testing.T) {
	delegate := NewRecordingAppender("delegate")
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

	replacement := NewRecordingAppender("replacement")
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

func TestNewConfigured_whenAsyncLoggerYamlEnabled_shouldDrainFileOnClose(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "logs", "async-logger.log")
	configPath := filepath.Join(dir, "goark-log.yml")
	WriteConfig(t, configPath, `
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

func TestAsyncLogger_whenIncludeLocationEnabled_shouldCaptureNativeCaller(t *testing.T) {
	delegate := NewRecordingAppender("delegate")
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
	frame := callsite.FrameFromPC(events[0].PC)
	if !strings.Contains(frame.Method, "TestAsyncLogger_whenIncludeLocationEnabled") {
		t.Fatalf("caller method = %q, want test method", frame.Method)
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

func TestAsyncLogger_whenAppendAfterClose_shouldReject(t *testing.T) {
	delegate := NewRecordingAppender("delegate")
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

	record := slog.NewRecord(FixedTestTime(), slog.LevelInfo, "after close", 0)
	if err := handler.Handle(context.Background(), record); err == nil {
		t.Fatalf("Handle() after Close should fail")
	}
}
