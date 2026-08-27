package goarklog

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"testing"
)

var errNilContextObserved = errors.New("nil context observed")

func TestAppenders_whenNilContext_shouldUseBackground(t *testing.T) {
	event := testEvent("nil context", fixedTestTime())

	t.Run("console", func(t *testing.T) {
		var out bytes.Buffer
		appender := NewConsoleAppender(
			WithConsoleWriter(&out),
			WithConsoleLayout(TextLayout{}),
		)
		if err := appender.Append(nil, event); err != nil {
			t.Fatalf("Append(nil) error = %v", err)
		}
		if out.Len() == 0 {
			t.Fatalf("Append(nil) should write event")
		}
	})

	t.Run("file", func(t *testing.T) {
		appender, err := NewFileAppender(
			filepath.Join(t.TempDir(), "app.log"),
			WithFileLayout(TextLayout{}),
		)
		if err != nil {
			t.Fatalf("NewFileAppender() error = %v", err)
		}
		defer appender.Close()
		if err := appender.Append(nil, event); err != nil {
			t.Fatalf("Append(nil) error = %v", err)
		}
	})

	t.Run("rolling", func(t *testing.T) {
		appender, err := NewRollingFileAppender(
			filepath.Join(t.TempDir(), "app.log"),
			WithRollingFileLayout(TextLayout{}),
		)
		if err != nil {
			t.Fatalf("NewRollingFileAppender() error = %v", err)
		}
		defer appender.Close()
		if err := appender.Append(nil, event); err != nil {
			t.Fatalf("Append(nil) error = %v", err)
		}
	})

	t.Run("json", func(t *testing.T) {
		var out bytes.Buffer
		appender := NewJSONAppender(WithJSONAppenderWriter(&out))
		if err := appender.Append(nil, event); err != nil {
			t.Fatalf("Append(nil) error = %v", err)
		}
		if out.Len() == 0 {
			t.Fatalf("Append(nil) should write event")
		}
	})
}

func TestDelegatingAppenders_whenNilContext_shouldUseBackground(t *testing.T) {
	event := testEvent("nil context", fixedTestTime())

	t.Run("filtered", func(t *testing.T) {
		delegate := &contextCheckingAppender{name: "filtered"}
		filter := contextCheckingFilter{}
		appender, err := NewFilteredAppender(delegate, filter)
		if err != nil {
			t.Fatalf("NewFilteredAppender() error = %v", err)
		}
		if err := appender.Append(nil, event); err != nil {
			t.Fatalf("Append(nil) error = %v", err)
		}
		if !delegate.called {
			t.Fatalf("delegate should be called")
		}
	})

	t.Run("failover", func(t *testing.T) {
		primary := &contextCheckingAppender{name: "primary"}
		failover := &contextCheckingAppender{name: "failover"}
		appender, err := NewFailoverAppender(primary, []Appender{failover})
		if err != nil {
			t.Fatalf("NewFailoverAppender() error = %v", err)
		}
		if err := appender.Append(nil, event); err != nil {
			t.Fatalf("Append(nil) error = %v", err)
		}
		if !primary.called || failover.called {
			t.Fatalf("primary called = %t, failover called = %t", primary.called, failover.called)
		}
	})

	t.Run("routing", func(t *testing.T) {
		route := &contextCheckingAppender{name: "route"}
		appender, err := NewRoutingAppender(map[string]Appender{"hot": route}, WithRoutingKeyFunc(func(ctx context.Context, _ Event) string {
			if ctx == nil {
				return "nil"
			}
			return "hot"
		}))
		if err != nil {
			t.Fatalf("NewRoutingAppender() error = %v", err)
		}
		if err := appender.Append(nil, event); err != nil {
			t.Fatalf("Append(nil) error = %v", err)
		}
		if !route.called {
			t.Fatalf("route should be called")
		}
	})

	t.Run("rewrite", func(t *testing.T) {
		delegate := &contextCheckingAppender{name: "rewrite"}
		appender, err := NewRewriteAppender(delegate, func(ctx context.Context, event Event) (Event, error) {
			if ctx == nil {
				return Event{}, errNilContextObserved
			}
			return event, nil
		})
		if err != nil {
			t.Fatalf("NewRewriteAppender() error = %v", err)
		}
		if err := appender.Append(nil, event); err != nil {
			t.Fatalf("Append(nil) error = %v", err)
		}
		if !delegate.called {
			t.Fatalf("delegate should be called")
		}
	})
}

func TestJSONAppender_whenFixedAttrFastPathAfterClose_shouldRejectAndNotWrite(t *testing.T) {
	var out bytes.Buffer
	appender := NewJSONAppender(WithJSONAppenderWriter(&out))
	handler, err := NewHandler(Options{
		Appenders: []Appender{appender},
		Root:      RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"json"}},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	logger, err := NewNativeLogger(handler, "goark.closed")
	if err != nil {
		t.Fatalf("NewNativeLogger() error = %v", err)
	}
	if err := handler.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := logger.LogAttrs3(context.Background(), slog.LevelInfo, "event",
		slog.String("profile", "bench"),
		slog.Int("index", 7),
		slog.Bool("closed", true),
	); err == nil {
		t.Fatalf("LogAttrs3() should reject closed JSON appender")
	}
	if out.Len() != 0 {
		t.Fatalf("closed JSON appender should not write, got %q", out.String())
	}
}

type contextCheckingAppender struct {
	name   string
	called bool
}

func (a *contextCheckingAppender) Name() string {
	return a.name
}

func (a *contextCheckingAppender) Append(ctx context.Context, _ Event) error {
	if ctx == nil {
		return errNilContextObserved
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	a.called = true
	return nil
}

func (a *contextCheckingAppender) Close() error {
	return nil
}

type contextCheckingFilter struct{}

func (contextCheckingFilter) Decide(ctx context.Context, _ Event) FilterDecision {
	if ctx == nil {
		return FilterDeny
	}
	return FilterNeutral
}
