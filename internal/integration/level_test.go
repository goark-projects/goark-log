package integration

import (
	"bytes"
	"log/slog"
	"strings"
	"sync"
	"testing"
)

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

func TestLevelRegistry_whenBuiltInLog4jLevelsParsed_shouldSupportThresholdNames(t *testing.T) {
	tests := []struct {
		name  string
		level slog.Level
	}{
		{name: "ALL", level: LevelAll},
		{name: "TRACE", level: LevelTrace},
		{name: "FATAL", level: LevelFatal},
		{name: "OFF", level: LevelOff},
	}
	for _, test := range tests {
		parsed, err := ParseLevel(test.name)
		if err != nil {
			t.Fatalf("ParseLevel(%s) error = %v", test.name, err)
		}
		if parsed != test.level {
			t.Fatalf("ParseLevel(%s) = %d, want %d", test.name, parsed, test.level)
		}
		if got := LevelName(test.level); got != test.name {
			t.Fatalf("LevelName(%d) = %q, want %q", test.level, got, test.name)
		}
	}
}

func TestLoggerContext_whenLevelChanges_shouldApplyAtomicallyAndRestoreInheritance(t *testing.T) {
	memory := newRecordingAppender("memory")
	configured := slog.LevelWarn
	context, err := NewLoggerContext(Options{
		Appenders: []Appender{memory},
		Root:      RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"memory"}},
		Loggers:   []LoggerRule{{Name: "service", Level: &configured}},
	})
	if err != nil {
		t.Fatalf("NewLoggerContext() error = %v", err)
	}
	defer context.Close()

	debug := slog.LevelDebug
	if err := context.SetLevel("service.worker", &debug); err != nil {
		t.Fatalf("SetLevel() error = %v", err)
	}
	context.Logger("service.worker").Debug("dynamic")
	if len(memory.Events()) != 1 {
		t.Fatalf("events = %d, want 1", len(memory.Events()))
	}

	configurations := context.LoggerConfigurations()
	assertLoggerConfiguration(t, configurations, "service.worker", slog.LevelDebug, slog.LevelDebug)
	if err := context.SetLevel("service.worker", nil); err != nil {
		t.Fatalf("restore level error = %v", err)
	}
	context.Logger("service.worker").Info("inherited")
	if len(memory.Events()) != 1 {
		t.Fatal("restored logger should inherit WARN from service")
	}
	assertLoggerConfiguration(t, context.LoggerConfigurations(), "service", slog.LevelWarn, slog.LevelWarn)
}

func TestLoggerContext_whenRootLevelChanges_shouldUpdateEffectiveLevel(t *testing.T) {
	memory := newRecordingAppender("memory")
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
	assertLoggerConfiguration(t, context.LoggerConfigurations(), "ROOT", slog.LevelError, slog.LevelError)
	assertInheritedLoggerConfiguration(t, context.LoggerConfigurations(), "service", slog.LevelError)
	context.Logger("service.worker").Info("disabled")
	if len(memory.Events()) != 0 {
		t.Fatal("named logger should inherit dynamic root level")
	}
	if err := context.SetLevel("", &errorLevel); err == nil {
		t.Fatal("empty logger name should fail")
	}
}

func TestLoggerContext_whenParentLevelChanges_shouldUpdateConfiguredDescendants(t *testing.T) {
	memory := newRecordingAppender("memory")
	context, err := NewLoggerContext(Options{
		Appenders: []Appender{memory},
		Root:      RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"memory"}},
		Loggers:   []LoggerRule{{Name: "service.worker"}},
	})
	if err != nil {
		t.Fatalf("NewLoggerContext() error = %v", err)
	}
	defer context.Close()

	warn := slog.LevelWarn
	if err := context.SetLevel("service", &warn); err != nil {
		t.Fatalf("SetLevel(service) error = %v", err)
	}
	assertInheritedLoggerConfiguration(t, context.LoggerConfigurations(), "service.worker", slog.LevelWarn)
	context.Logger("service.worker").Info("disabled")
	if len(memory.Events()) != 0 {
		t.Fatal("configured descendant should inherit dynamic parent level")
	}
}

func TestLoggerContext_whenLevelIsNotConfigured_shouldInheritNearestConfiguredParent(t *testing.T) {
	memory := newRecordingAppender("memory")
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

	assertInheritedLoggerConfiguration(t, context.LoggerConfigurations(), "service.worker", slog.LevelWarn)
	context.Logger("service.worker.task").Info("disabled")
	if len(memory.Events()) != 0 {
		t.Fatal("named logger should inherit nearest configured parent level")
	}
}

func TestLoggerContext_whenLevelChangesDuringWrites_shouldRemainConcurrentSafe(t *testing.T) {
	memory := newRecordingAppender("memory")
	context, err := NewLoggerContext(Options{
		Appenders: []Appender{memory},
		Root:      RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"memory"}},
	})
	if err != nil {
		t.Fatalf("NewLoggerContext() error = %v", err)
	}
	defer context.Close()

	var wait sync.WaitGroup
	wait.Add(2)
	go func() {
		defer wait.Done()
		for index := 0; index < 1_000; index++ {
			level := slog.LevelDebug
			if index%2 == 0 {
				level = slog.LevelError
			}
			if setErr := context.SetLevel("service", &level); setErr != nil {
				t.Errorf("SetLevel() error = %v", setErr)
				return
			}
		}
	}()
	go func() {
		defer wait.Done()
		logger := context.Logger("service.worker")
		for index := 0; index < 1_000; index++ {
			logger.Info("concurrent", "index", index)
		}
	}()
	wait.Wait()
}

func TestLoggerContext_whenReloadAfterClose_shouldRejectAndCloseReplacement(t *testing.T) {
	initial := newRecordingAppender("initial")
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

	replacement := newRecordingAppender("replacement")
	err = context.Reload(Options{
		Appenders: []Appender{replacement},
		Root:      RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"replacement"}},
	})
	if err == nil || !strings.Contains(err.Error(), "closed") {
		t.Fatalf("Reload() error = %v, want closed error", err)
	}
	replacement.mu.Lock()
	closeCount := replacement.closeCount
	replacement.mu.Unlock()
	if closeCount != 1 {
		t.Fatalf("replacement close count = %d, want 1", closeCount)
	}
	level := slog.LevelDebug
	if err := context.SetLevel("service", &level); err == nil || !strings.Contains(err.Error(), "closed") {
		t.Fatalf("SetLevel() error = %v, want closed error", err)
	}
	if err := context.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
	initial.mu.Lock()
	initialCloseCount := initial.closeCount
	initial.mu.Unlock()
	if initialCloseCount != 1 {
		t.Fatalf("initial close count = %d, want 1", initialCloseCount)
	}
}

func assertLoggerConfiguration(t *testing.T, configurations []LoggerConfiguration, name string, configured slog.Level, effective slog.Level) {
	t.Helper()
	for _, configuration := range configurations {
		if configuration.Name != name {
			continue
		}
		if configuration.ConfiguredLevel == nil || *configuration.ConfiguredLevel != configured || configuration.EffectiveLevel != effective {
			t.Fatalf("configuration %q = %#v", name, configuration)
		}
		return
	}
	t.Fatalf("configuration %q not found: %#v", name, configurations)
}

func assertInheritedLoggerConfiguration(t *testing.T, configurations []LoggerConfiguration, name string, effective slog.Level) {
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

func TestRegisterLevel_whenDefaultRegistryUsed_shouldAffectParseLevelAndLayouts(t *testing.T) {
	level := slog.Level(10)
	if err := RegisterLevel("ALERT", level); err != nil {
		t.Fatalf("RegisterLevel() error = %v", err)
	}
	parsed, err := ParseLevel("alert")
	if err != nil {
		t.Fatalf("ParseLevel() error = %v", err)
	}
	if parsed != level {
		t.Fatalf("parsed level = %d, want %d", parsed, level)
	}
	if got := LevelName(level); got != "ALERT" {
		t.Fatalf("LevelName() = %q, want ALERT", got)
	}
}

func TestLevelRegistry_whenNameInvalid_shouldReject(t *testing.T) {
	registry := NewLevelRegistry()
	if err := registry.Register("bad level", slog.Level(2)); err == nil {
		t.Fatalf("Register() error = nil, want whitespace rejection")
	}
	if err := registry.Register("123", slog.Level(2)); err == nil {
		t.Fatalf("Register() error = nil, want numeric name rejection")
	}
}

func TestLevelRegistry_whenUnknownValue_shouldFallbackToStandardRange(t *testing.T) {
	registry := NewLevelRegistry()
	if got := registry.Name(slog.Level(2)); got != "INFO" {
		t.Fatalf("Name(2) = %q, want INFO", got)
	}
}

func TestNativeLogger_whenFatalUsed_shouldWriteFatalLevelName(t *testing.T) {
	var out bytes.Buffer
	handler, err := NewHandler(Options{
		Appenders: []Appender{NewConsoleAppender(WithConsoleWriter(&out), WithConsoleLayout(TextLayout{}))},
		Root:      RootLogger{Level: slog.LevelError, AppenderRefs: []string{"console"}},
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
	if got := out.String(); !strings.Contains(got, "level=FATAL") || !strings.Contains(got, "fatal event") {
		t.Fatalf("fatal output is wrong: %q", got)
	}
}
