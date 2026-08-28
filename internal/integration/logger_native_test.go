package integration

import (
	"bytes"
	"context"
	"encoding/json"
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

func TestNativeLogger_whenDirectJSONFastPathUsed_shouldKeepBoundAttrs(t *testing.T) {
	var out bytes.Buffer
	handler, err := NewHandler(Options{
		Appenders: []Appender{
			NewJSONAppender(WithJSONAppenderWriter(&out)),
		},
		Root: RootLogger{
			Level:        slog.LevelInfo,
			AppenderRefs: []string{"json"},
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
	bound := logger.WithAttrs(slog.String("service", "billing"))
	if err := bound.LogAttrs(context.Background(), slog.LevelInfo, "request done", slog.Int("status", 200)); err != nil {
		t.Fatalf("LogAttrs() error = %v", err)
	}
	grouped := logger.WithGroup("request").WithAttrs(slog.String("service", "billing"))
	if err := grouped.LogAttrs3(context.Background(), slog.LevelInfo, "request done",
		slog.String("method", "GET"),
		slog.Int("status", 200),
		slog.Bool("ok", true),
	); err != nil {
		t.Fatalf("LogAttrs3() error = %v", err)
	}

	decoder := json.NewDecoder(&out)
	first := decodeJSONLogLine(t, decoder)
	if first["service"] != "billing" || first["status"] != float64(200) {
		t.Fatalf("first event = %+v, want bound service and status", first)
	}
	second := decodeJSONLogLine(t, decoder)
	if second["request.service"] != "billing" ||
		second["request.method"] != "GET" ||
		second["request.status"] != float64(200) ||
		second["request.ok"] != true {
		t.Fatalf("second event = %+v, want grouped bound attrs and call attrs", second)
	}
}

func TestNativeLoggerSlog_whenGroupsBound_shouldPreserveInteropAttrs(t *testing.T) {
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
	logger.WithGroup("request").
		WithAttrs(slog.String("service", "billing")).
		Slog().
		Info("via slog", slog.Int("status", 200))

	events := appender.Events()
	if len(events) != 1 {
		t.Fatalf("event count = %d, want 1", len(events))
	}
	event := events[0]
	if event.Logger != "goark.native" {
		t.Fatalf("event logger = %q, want goark.native", event.Logger)
	}
	assertAttrString(t, event, "request.service", "billing")
	assertAttrString(t, event, "request.status", "200")
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

func decodeJSONLogLine(t *testing.T, decoder *json.Decoder) map[string]any {
	t.Helper()
	var fields map[string]any
	if err := decoder.Decode(&fields); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	return fields
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
