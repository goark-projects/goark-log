package integration

import (
	"log/slog"
	"sync"
	"testing"

	. "goark.dev/log/internal/testsupport"
)

func TestLoggerContext_whenLevelChanges_shouldApplyAtomicallyAndRestoreInheritance(t *testing.T) {
	memory := NewRecordingAppender("memory")
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
	assertLoggerConfiguration(
		t,
		context.LoggerConfigurations(),
		"service",
		slog.LevelWarn,
		slog.LevelWarn,
	)
}

func TestLoggerContext_whenLevelChangesDuringWrites_shouldRemainConcurrentSafe(t *testing.T) {
	memory := NewRecordingAppender("memory")
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

func TestLoggerContext_whenParentLevelChanges_shouldUpdateConfiguredDescendants(t *testing.T) {
	memory := NewRecordingAppender("memory")
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
	assertInheritedLoggerConfiguration(
		t,
		context.LoggerConfigurations(),
		"service.worker",
		slog.LevelWarn,
	)
	context.Logger("service.worker").Info("disabled")
	if len(memory.Events()) != 0 {
		t.Fatal("configured descendant should inherit dynamic parent level")
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

func assertLoggerConfiguration(
	t *testing.T,
	configurations []LoggerConfiguration,
	name string,
	configured slog.Level,
	effective slog.Level,
) {
	t.Helper()
	for _, configuration := range configurations {
		if configuration.Name != name {
			continue
		}
		if configuration.ConfiguredLevel == nil || *configuration.ConfiguredLevel != configured ||
			configuration.EffectiveLevel != effective {
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
