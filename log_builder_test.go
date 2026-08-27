package goarklog

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"goark.dev/log/internal/logvalue"
)

func TestLogBuilder_whenFluentEventBuilt_shouldDispatchCompleteEvent(t *testing.T) {
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

	logger, err := NewNativeLogger(handler, "goark.builder")
	if err != nil {
		t.Fatalf("NewNativeLogger() error = %v", err)
	}
	boom := errors.New("boom")
	err = logger.AtInfo().
		WithContext(WithContextAttrs(context.Background(), slog.String("traceId", "trace-1"))).
		WithMarker(NewMarker("AUDIT")).
		WithError(boom).
		WithGroup("request").
		WithString("path", "/login").
		WithInt("status", 200).
		Log("accepted")
	if err != nil {
		t.Fatalf("Log() error = %v", err)
	}

	events := appender.Events()
	if len(events) != 1 {
		t.Fatalf("event count = %d, want 1", len(events))
	}
	event := events[0]
	if event.Message != "accepted" || event.Logger != "goark.builder" {
		t.Fatalf("event = %+v, want builder event", event)
	}
	if event.Marker == nil || event.Marker.Name != "AUDIT" {
		t.Fatalf("marker = %+v, want AUDIT", event.Marker)
	}
	if event.Throwable == nil || event.Throwable.Message != "boom" {
		t.Fatalf("throwable = %+v, want boom", event.Throwable)
	}
	assertAttrString(t, event, "traceId", "trace-1")
	assertAttrString(t, event, "request.path", "/login")
	assertAttrString(t, event, "request.status", "200")
}

func TestLogBuilder_whenLevelDisabled_shouldAvoidAttrsAndSkipEvent(t *testing.T) {
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

	logger, err := NewNativeLogger(handler, "goark.builder")
	if err != nil {
		t.Fatalf("NewNativeLogger() error = %v", err)
	}
	if logger.AtInfo().Enabled() {
		t.Fatalf("AtInfo().Enabled() = true, want false")
	}
	if err := logger.AtInfo().WithAny("payload", map[string]any{"x": 1}).Log("ignored"); err != nil {
		t.Fatalf("Log() error = %v", err)
	}
	if len(appender.Events()) != 0 {
		t.Fatalf("events = %+v, want none", appender.Events())
	}
}

func TestLogBuilder_whenMessageHasAttrs_shouldAppendMessageAttrs(t *testing.T) {
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

	logger, err := NewNativeLogger(handler, "goark.builder")
	if err != nil {
		t.Fatalf("NewNativeLogger() error = %v", err)
	}
	message := NewStructuredDataMessage("audit@32473", "login", "accepted", slog.String("user", "alice"))
	if err := logger.AtInfo().WithString("traceId", "trace-1").LogMessage(message); err != nil {
		t.Fatalf("LogMessage() error = %v", err)
	}

	events := appender.Events()
	if len(events) != 1 {
		t.Fatalf("event count = %d, want 1", len(events))
	}
	assertAttrString(t, events[0], "traceId", "trace-1")
	assertAttrString(t, events[0], StructuredDataIDAttrKey, "audit@32473")
	assertAttrString(t, events[0], StructuredDataTypeAttrKey, "login")
	assertAttrString(t, events[0], "user", "alice")
}

func TestLogBuilder_whenLogfCalled_shouldUseParameterizedMessage(t *testing.T) {
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

	logger, err := NewNativeLogger(handler, "goark.builder")
	if err != nil {
		t.Fatalf("NewNativeLogger() error = %v", err)
	}
	if err := logger.AtInfo().Logf("user {} status {}", "alice", 200); err != nil {
		t.Fatalf("Logf() error = %v", err)
	}
	if got := appender.Events()[0].Message; got != "user alice status 200" {
		t.Fatalf("message = %q, want parameterized message", got)
	}
}

func TestLogBuilder_whenMessageFactoryConfigured_shouldUseFactory(t *testing.T) {
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

	logger, err := NewNativeLogger(handler, "goark.builder",
		WithLoggerMessageFactory(MessageFactoryFunc(func(pattern string, args ...any) Message {
			return NewSimpleMessage(pattern + ":" + logvalue.String(slog.AnyValue(args[0])))
		})),
	)
	if err != nil {
		t.Fatalf("NewNativeLogger() error = %v", err)
	}
	if err := logger.AtInfo().Logf("factory", "value"); err != nil {
		t.Fatalf("Logf() error = %v", err)
	}
	if got := appender.Events()[0].Message; got != "factory:value" {
		t.Fatalf("message = %q, want factory:value", got)
	}
}
