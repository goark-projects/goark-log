package goarklog

import (
	"log/slog"
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
