package goarklog

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestHandler_whenDefaultPatternUsed_shouldRenderSpringBootStyleLine(t *testing.T) {
	var out bytes.Buffer
	handler, err := NewHandler(Options{
		Appenders: []Appender{NewConsoleAppender(WithConsoleWriter(&out))},
		Root: RootLogger{
			Level:        slog.LevelInfo,
			AppenderRefs: []string{"console"},
		},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	record := slog.NewRecord(time.Date(2026, 8, 25, 10, 15, 30, 123000000, time.FixedZone("CST", 8*3600)), slog.LevelInfo, "service started", 0)
	record.AddAttrs(slog.String("profile", "dev"))
	if err := NewLogger(handler, "goark.boot").Handler().Handle(context.Background(), record); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	line := out.String()
	if !strings.HasPrefix(line, "2026-08-25T10:15:30.123+08:00  INFO ") {
		t.Fatalf("line should start with Spring Boot timestamp and level, got %q", line)
	}
	for _, want := range []string{" --- [main] goark.boot : service started", "profile=dev\n"} {
		if !strings.Contains(line, want) {
			t.Fatalf("line should contain %q, got %q", want, line)
		}
	}
}

func TestHandler_whenNamedLoggerLevelIsLower_shouldUseMostSpecificRule(t *testing.T) {
	var rootOut bytes.Buffer
	var ormOut bytes.Buffer
	debug := slog.LevelDebug
	handler, err := NewHandler(Options{
		Appenders: []Appender{
			NewConsoleAppender(WithConsoleName("root"), WithConsoleWriter(&rootOut), WithConsoleLayout(TextLayout{})),
			NewConsoleAppender(WithConsoleName("orm"), WithConsoleWriter(&ormOut), WithConsoleLayout(TextLayout{})),
		},
		Root: RootLogger{Level: slog.LevelWarn, AppenderRefs: []string{"root"}},
		Loggers: []LoggerRule{
			{
				Name:          "goark.orm",
				Level:         &debug,
				AppenderRefs:  []string{"orm"},
				Additivity:    false,
				AdditivitySet: true,
			},
		},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	logger := NewLogger(handler, "goark.orm.mapper")
	logger.Debug("sql prepared", slog.String("statement", "FindByID"))
	logger.Info("sql done")
	slog.New(handler).Info("root info hidden")
	slog.New(handler).Warn("root warn visible")

	if rootOut.String() == "" || strings.Contains(rootOut.String(), "root info hidden") {
		t.Fatalf("root output should only contain warn, got %q", rootOut.String())
	}
	if strings.Contains(rootOut.String(), "sql prepared") {
		t.Fatalf("non-additive named logger should not write root appender, got %q", rootOut.String())
	}
	if !strings.Contains(ormOut.String(), "sql prepared") || !strings.Contains(ormOut.String(), "statement=FindByID") {
		t.Fatalf("named output should contain debug SQL line, got %q", ormOut.String())
	}
}

func TestHandler_whenAppenderRefControlConfigured_shouldApplyLevelAndFiltersPerAppender(t *testing.T) {
	allAppender := newRecordingAppender("all")
	errorAppender := newRecordingAppender("errors")
	auditAppender := newRecordingAppender("audit")
	auditOnly, err := NewAttrFilter("kind", "audit",
		WithFilterOnMatch(FilterAccept),
		WithFilterOnMismatch(FilterDeny),
	)
	if err != nil {
		t.Fatalf("NewAttrFilter() error = %v", err)
	}
	handler, err := NewHandler(Options{
		Appenders: []Appender{allAppender, errorAppender, auditAppender},
		Root: RootLogger{
			Level:        slog.LevelDebug,
			AppenderRefs: []string{"all"},
			AppenderRefControls: []AppenderRef{
				NewAppenderRef("errors", WithAppenderRefLevel(slog.LevelError)),
				NewAppenderRef("audit", WithAppenderRefFilters(auditOnly)),
			},
		},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	logger := NewLogger(handler, "goark.audit")
	logger.Info("business event", slog.String("kind", "biz"))
	logger.Info("audit event", slog.String("kind", "audit"))
	logger.Error("error event")

	if !allAppender.Contains("business event") ||
		!allAppender.Contains("audit event") ||
		!allAppender.Contains("error event") {
		t.Fatalf("all appender events = %+v, want every event", allAppender.Events())
	}
	if errorAppender.Contains("business event") ||
		errorAppender.Contains("audit event") ||
		!errorAppender.Contains("error event") {
		t.Fatalf("error appender events = %+v, want only error event", errorAppender.Events())
	}
	if auditAppender.Contains("business event") ||
		!auditAppender.Contains("audit event") ||
		auditAppender.Contains("error event") {
		t.Fatalf("audit appender events = %+v, want only audit event", auditAppender.Events())
	}
}

func TestHandler_whenUsedConcurrently_shouldKeepLinesComplete(t *testing.T) {
	var out bytes.Buffer
	handler, err := NewHandler(Options{
		Appenders: []Appender{NewConsoleAppender(WithConsoleWriter(&out), WithConsoleLayout(TextLayout{}))},
		Root:      RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"console"}},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	logger := NewLogger(handler, "goark.concurrent")
	var wait sync.WaitGroup
	for index := 0; index < 64; index++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			logger.Info("event", slog.Int("index", index))
		}(index)
	}
	wait.Wait()

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 64 {
		t.Fatalf("expected 64 complete lines, got %d: %q", len(lines), out.String())
	}
	for _, line := range lines {
		if !strings.Contains(line, "msg=event") || !strings.Contains(line, "logger=goark.concurrent") {
			t.Fatalf("line is incomplete: %q", line)
		}
	}
}

func TestNewHandler_whenAppenderRefMissing_shouldRejectConfig(t *testing.T) {
	_, err := NewHandler(Options{
		Appenders: []Appender{NewConsoleAppender()},
		Root:      RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"missing"}},
	})
	if err == nil {
		t.Fatalf("NewHandler() should reject missing appender ref")
	}
}
