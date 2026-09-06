package integration

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	. "goark.dev/log/internal/testsupport"
)

func TestLoggerContext_whenReloadAfterClose_shouldRejectAndCloseReplacement(t *testing.T) {
	initial := NewRecordingAppender("initial")
	context, err := NewLoggerContext(Options{
		Appenders: []Appender{initial},
		Root:      RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"initial"}},
	})
	if err != nil {
		t.Fatalf("NewLoggerContext() error = %v", err)
	}
	if err := context.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	replacement := NewRecordingAppender("replacement")
	err = context.Reload(Options{
		Appenders: []Appender{replacement},
		Root:      RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"replacement"}},
	})
	if err == nil || !strings.Contains(err.Error(), "closed") {
		t.Fatalf("Reload() error = %v, want closed error", err)
	}
	closeCount := replacement.CloseCount()
	if closeCount != 1 {
		t.Fatalf("replacement close count = %d, want 1", closeCount)
	}
	level := slog.LevelDebug
	if err := context.SetLevel("service", &level); err == nil ||
		!strings.Contains(err.Error(), "closed") {
		t.Fatalf("SetLevel() error = %v, want closed error", err)
	}
	if err := context.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
	initialCloseCount := initial.CloseCount()
	if initialCloseCount != 1 {
		t.Fatalf("initial close count = %d, want 1", initialCloseCount)
	}
}

func TestLoggerContext_whenRootLevelChanges_shouldUpdateEffectiveLevel(t *testing.T) {
	memory := NewRecordingAppender("memory")
	context, err := NewLoggerContext(Options{
		Appenders: []Appender{memory},
		Root:      RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"memory"}},
		Loggers:   []LoggerRule{{Name: "service"}},
	})
	if err != nil {
		t.Fatalf("NewLoggerContext() error = %v", err)
	}
	defer context.Close()

	errorLevel := slog.LevelError
	if err := context.SetLevel("root", &errorLevel); err != nil {
		t.Fatalf("SetLevel(root) error = %v", err)
	}
	assertLoggerConfiguration(
		t,
		context.LoggerConfigurations(),
		"ROOT",
		slog.LevelError,
		slog.LevelError,
	)
	assertInheritedLoggerConfiguration(
		t,
		context.LoggerConfigurations(),
		"service",
		slog.LevelError,
	)
	context.Logger("service.worker").Info("disabled")
	if len(memory.Events()) != 0 {
		t.Fatal("named logger should inherit dynamic root level")
	}
	if err := context.SetLevel("", &errorLevel); err == nil {
		t.Fatal("empty logger name should fail")
	}
}

func TestLoggerContext_whenLevelIsNotConfigured_shouldInheritNearestConfiguredParent(t *testing.T) {
	memory := NewRecordingAppender("memory")
	warn := slog.LevelWarn
	context, err := NewLoggerContext(Options{
		Appenders: []Appender{memory},
		Root:      RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"memory"}},
		Loggers: []LoggerRule{
			{Name: "service", Level: &warn},
			{Name: "service.worker"},
		},
	})
	if err != nil {
		t.Fatalf("NewLoggerContext() error = %v", err)
	}
	defer context.Close()

	assertInheritedLoggerConfiguration(
		t,
		context.LoggerConfigurations(),
		"service.worker",
		slog.LevelWarn,
	)
	context.Logger("service.worker.task").Info("disabled")
	if len(memory.Events()) != 0 {
		t.Fatal("named logger should inherit nearest configured parent level")
	}
}

func TestNativeLogger_whenFatalUsed_shouldWriteFatalLevelName(t *testing.T) {
	var out bytes.Buffer
	handler, err := NewHandler(Options{
		Appenders: []Appender{
			NewConsoleAppender(WithConsoleWriter(&out), WithConsoleLayout(TextLayout{})),
		},
		Root: RootLogger{Level: slog.LevelError, AppenderRefs: []string{"console"}},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	logger, err := NewNativeLogger(handler, "goark.level")
	if err != nil {
		t.Fatalf("NewNativeLogger() error = %v", err)
	}
	if err := logger.AtFatal().Log("fatal event"); err != nil {
		t.Fatalf("AtFatal().Log() error = %v", err)
	}
	if got := out.String(); !strings.Contains(got, "level=FATAL") ||
		!strings.Contains(got, "fatal event") {
		t.Fatalf("fatal output is wrong: %q", got)
	}
}
func TestLevelRegistry_whenCustomLevelRegistered_shouldParseAndFormatName(t *testing.T) {
	registry := NewLevelRegistry()
	level := slog.Level(6)

	if err := registry.Register("NOTICE", level); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	parsed, err := registry.Parse("notice")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if parsed != level {
		t.Fatalf("parsed level = %d, want %d", parsed, level)
	}
	if got := registry.Name(level); got != "NOTICE" {
		t.Fatalf("Name() = %q, want NOTICE", got)
	}
}

func assertInheritedLoggerConfiguration(
	t *testing.T,
	configurations []LoggerConfiguration,
	name string,
	effective slog.Level,
) {
	t.Helper()
	for _, configuration := range configurations {
		if configuration.Name != name {
			continue
		}
		if configuration.ConfiguredLevel != nil || configuration.EffectiveLevel != effective {
			t.Fatalf("configuration %q = %#v", name, configuration)
		}
		return
	}
	t.Fatalf("configuration %q not found: %#v", name, configurations)
}

func TestLevelRegistry_whenUnknownValue_shouldFallbackToStandardRange(t *testing.T) {
	registry := NewLevelRegistry()
	if got := registry.Name(slog.Level(2)); got != "INFO" {
		t.Fatalf("Name(2) = %q, want INFO", got)
	}
}
