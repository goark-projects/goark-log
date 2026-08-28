package integration

import (
	"bytes"
	"log/slog"
	"strings"
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
