package integration

import (
	"context"
	"log/slog"
	"sync/atomic"
	"testing"
)

func TestHandler_whenGlobalAcceptFilterMatches_shouldBypassLoggerLevel(t *testing.T) {
	appender := newRecordingAppender("memory")
	acceptAudit, err := NewAttrFilter("kind", "audit",
		WithFilterOnMatch(FilterAccept),
		WithFilterOnMismatch(FilterNeutral),
	)
	if err != nil {
		t.Fatalf("NewAttrFilter() error = %v", err)
	}
	handler, err := NewHandler(Options{
		Appenders: []Appender{appender},
		Filters:   []Filter{acceptAudit},
		Root:      RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"memory"}},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	logger, err := NewNativeLogger(handler, "goark.global")
	if err != nil {
		t.Fatalf("NewNativeLogger() error = %v", err)
	}

	if err := logger.Debug("debug accepted", slog.String("kind", "audit")); err != nil {
		t.Fatalf("Debug() error = %v", err)
	}
	if err := logger.Debug("debug neutral", slog.String("kind", "business")); err != nil {
		t.Fatalf("Debug() error = %v", err)
	}

	if !appender.Contains("debug accepted") || appender.Contains("debug neutral") {
		t.Fatalf("events = %+v, want only globally accepted debug event", appender.Events())
	}
}

func TestHandler_whenGlobalFilterExists_shouldKeepSlogDebugReachable(t *testing.T) {
	appender := newRecordingAppender("memory")
	acceptAudit, err := NewAttrFilter("kind", "audit",
		WithFilterOnMatch(FilterAccept),
		WithFilterOnMismatch(FilterNeutral),
	)
	if err != nil {
		t.Fatalf("NewAttrFilter() error = %v", err)
	}
	handler, err := NewHandler(Options{
		Appenders: []Appender{appender},
		Filters:   []Filter{acceptAudit},
		Root:      RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"memory"}},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	logger := NewLogger(handler, "goark.global")

	if !logger.Enabled(context.Background(), slog.LevelDebug) {
		t.Fatalf("Enabled(DEBUG) = false, want true when global filters are configured")
	}
	logger.Debug("slog accepted", slog.String("kind", "audit"))
	logger.Debug("slog neutral", slog.String("kind", "business"))

	if !appender.Contains("slog accepted") || appender.Contains("slog neutral") {
		t.Fatalf("events = %+v, want only globally accepted slog debug event", appender.Events())
	}
}

func TestHandler_whenGlobalDenyFilterMatches_shouldShortCircuitRouteFilters(t *testing.T) {
	appender := newRecordingAppender("memory")
	denyHealth, err := NewRegexFilter("health",
		WithRegexOnMatch(FilterDeny),
		WithRegexOnMismatch(FilterNeutral),
	)
	if err != nil {
		t.Fatalf("NewRegexFilter() error = %v", err)
	}
	var routeFilterCalls atomic.Int32
	routeFilter := FilterFunc(func(context.Context, Event) FilterDecision {
		routeFilterCalls.Add(1)
		return FilterNeutral
	})
	handler, err := NewHandler(Options{
		Appenders: []Appender{appender},
		Filters:   []Filter{denyHealth},
		Root: RootLogger{
			Level:        slog.LevelDebug,
			AppenderRefs: []string{"memory"},
			Filters:      []Filter{routeFilter},
		},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	logger, err := NewNativeLogger(handler, "goark.global")
	if err != nil {
		t.Fatalf("NewNativeLogger() error = %v", err)
	}

	if err := logger.Info("health probe"); err != nil {
		t.Fatalf("Info() error = %v", err)
	}
	if err := logger.Info("request done"); err != nil {
		t.Fatalf("Info() error = %v", err)
	}

	if appender.Contains("health probe") || !appender.Contains("request done") {
		t.Fatalf("events = %+v, want global deny to drop only health probe", appender.Events())
	}
	if got := routeFilterCalls.Load(); got != 1 {
		t.Fatalf("route filter calls = %d, want only the non-denied event to reach route filters", got)
	}
}

func TestHandler_whenGlobalAcceptRunsAsync_shouldSurviveConsumerLevelCheck(t *testing.T) {
	appender := newRecordingAppender("memory")
	acceptAudit, err := NewAttrFilter("kind", "audit",
		WithFilterOnMatch(FilterAccept),
		WithFilterOnMismatch(FilterNeutral),
	)
	if err != nil {
		t.Fatalf("NewAttrFilter() error = %v", err)
	}
	handler, err := NewHandler(Options{
		Appenders: []Appender{appender},
		Filters:   []Filter{acceptAudit},
		Root:      RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"memory"}},
		Async: AsyncLoggerOptions{
			Enabled:   true,
			QueueSize: 8,
			BatchSize: 4,
		},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	logger, err := NewNativeLogger(handler, "goark.global.async")
	if err != nil {
		t.Fatalf("NewNativeLogger() error = %v", err)
	}

	if err := logger.Debug("async accepted", slog.String("kind", "audit")); err != nil {
		t.Fatalf("Debug() error = %v", err)
	}
	if err := handler.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if !appender.Contains("async accepted") {
		t.Fatalf("events = %+v, want async accepted event", appender.Events())
	}
}
