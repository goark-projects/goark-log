package goarklog

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"goark.dev/log/internal/callsite"
	"goark.dev/log/internal/logvalue"
)

func TestNativeLogger_whenInfoCalled_shouldDispatchNamedEvent(t *testing.T) {
	appender := newRecordingAppender("memory")
	handler, err := NewHandler(Options{
		Appenders: []Appender{appender},
		Root: RootLogger{
			Level:        slog.LevelInfo,
			AppenderRefs: []string{"memory"},
		},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	defer handler.Close()

	logger, err := NewNativeLogger(handler, "goark.native")
	if err != nil {
		t.Fatalf("NewNativeLogger() error = %v", err)
	}
	ctx := WithContextAttrs(context.Background(), slog.String("trace_id", "trace-1"))
	err = logger.WithGroup("request").WithAttrs(slog.String("component", "api")).
		InfoContext(ctx, "request done", slog.Int("status", 200))
	if err != nil {
		t.Fatalf("InfoContext() error = %v", err)
	}

	events := appender.Events()
	if len(events) != 1 {
		t.Fatalf("event count = %d, want 1", len(events))
	}
	event := events[0]
	if event.Logger != "goark.native" || event.Message != "request done" {
		t.Fatalf("event = %+v", event)
	}
	assertAttrString(t, event, "trace_id", "trace-1")
	assertAttrString(t, event, "request.component", "api")
	assertAttrString(t, event, "request.status", "200")
}

func TestNativeLogger_whenLevelDisabled_shouldSkipEvent(t *testing.T) {
	appender := newRecordingAppender("memory")
	handler, err := NewHandler(Options{
		Appenders: []Appender{appender},
		Root: RootLogger{
			Level:        slog.LevelWarn,
			AppenderRefs: []string{"memory"},
		},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	defer handler.Close()

	logger, err := NewNativeLogger(handler, "goark.native")
	if err != nil {
		t.Fatalf("NewNativeLogger() error = %v", err)
	}
	if logger.Enabled(context.Background(), slog.LevelInfo) {
		t.Fatalf("Enabled(INFO) = true, want false")
	}
	if err := logger.Info("ignored", slog.String("key", "value")); err != nil {
		t.Fatalf("Info() error = %v", err)
	}
	if got := len(appender.Events()); got != 0 {
		t.Fatalf("event count = %d, want 0", got)
	}
}

func TestNativeLogger_whenCallerEnabled_shouldCaptureCallSite(t *testing.T) {
	appender := newRecordingAppender("memory")
	handler, err := NewHandler(Options{
		Appenders: []Appender{appender},
		Root: RootLogger{
			Level:        slog.LevelInfo,
			AppenderRefs: []string{"memory"},
		},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	defer handler.Close()

	logger, err := NewNativeLogger(handler, "goark.native", WithLoggerCaller(true))
	if err != nil {
		t.Fatalf("NewNativeLogger() error = %v", err)
	}
	if err := logger.Info("with caller"); err != nil {
		t.Fatalf("Info() error = %v", err)
	}

	events := appender.Events()
	if len(events) != 1 {
		t.Fatalf("event count = %d, want 1", len(events))
	}
	frame := callsite.FrameFromPC(events[0].PC)
	if !strings.Contains(frame.Method, "TestNativeLogger_whenCallerEnabled") {
		t.Fatalf("caller method = %q, want test method", frame.Method)
	}
}

func assertAttrString(t *testing.T, event Event, key string, want string) {
	t.Helper()
	value, ok := event.Attr(key)
	if !ok {
		t.Fatalf("event attr %q missing, event=%+v", key, event)
	}
	if got := logvalue.String(value); got != want {
		t.Fatalf("event attr %q = %q, want %q", key, got, want)
	}
}
