package goarklog

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandler_whenRootThresholdFilterDeniesDebug_shouldDropBeforeAppender(t *testing.T) {
	var out bytes.Buffer
	handler, err := NewHandler(Options{
		Appenders: []Appender{NewConsoleAppender(WithConsoleWriter(&out), WithConsoleLayout(TextLayout{}))},
		Root: RootLogger{
			Level:        slog.LevelDebug,
			AppenderRefs: []string{"console"},
			Filters:      []Filter{NewThresholdFilter(slog.LevelInfo)},
		},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	logger := NewLogger(handler, "goark.filter")
	logger.Debug("hidden debug")
	logger.Info("visible info")

	if strings.Contains(out.String(), "hidden debug") || !strings.Contains(out.String(), "visible info") {
		t.Fatalf("root threshold filter output is wrong: %q", out.String())
	}
}

func TestHandler_whenLoggerRegexFilterDeniesMessage_shouldKeepOtherEvents(t *testing.T) {
	var out bytes.Buffer
	denyHealth, err := NewRegexFilter("health",
		WithRegexOnMatch(FilterDeny),
		WithRegexOnMismatch(FilterNeutral),
	)
	if err != nil {
		t.Fatalf("NewRegexFilter() error = %v", err)
	}
	handler, err := NewHandler(Options{
		Appenders: []Appender{NewConsoleAppender(WithConsoleWriter(&out), WithConsoleLayout(TextLayout{}))},
		Root:      RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"console"}},
		Loggers: []LoggerRule{
			{
				Name:          "goark.web",
				Level:         levelPointer(slog.LevelInfo),
				Filters:       []Filter{denyHealth},
				Additivity:    true,
				AdditivitySet: true,
			},
		},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	logger := NewLogger(handler, "goark.web")
	logger.Info("health probe")
	logger.Info("request done")

	if strings.Contains(out.String(), "health probe") || !strings.Contains(out.String(), "request done") {
		t.Fatalf("logger regex filter output is wrong: %q", out.String())
	}
}

func TestFilteredAppender_whenAttrFilterDenies_shouldOnlyAffectWrappedAppender(t *testing.T) {
	var filteredOut bytes.Buffer
	var plainOut bytes.Buffer
	denyAudit, err := NewAttrFilter("kind", "audit",
		WithFilterOnMatch(FilterDeny),
		WithFilterOnMismatch(FilterNeutral),
	)
	if err != nil {
		t.Fatalf("NewAttrFilter() error = %v", err)
	}
	filtered, err := NewFilteredAppender(
		NewConsoleAppender(WithConsoleName("filtered"), WithConsoleWriter(&filteredOut), WithConsoleLayout(TextLayout{})),
		denyAudit,
	)
	if err != nil {
		t.Fatalf("NewFilteredAppender() error = %v", err)
	}
	handler, err := NewHandler(Options{
		Appenders: []Appender{
			filtered,
			NewConsoleAppender(WithConsoleName("plain"), WithConsoleWriter(&plainOut), WithConsoleLayout(TextLayout{})),
		},
		Root: RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"filtered", "plain"}},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	logger := NewLogger(handler, "goark.filter")
	logger.Info("audit event", slog.String("kind", "audit"))
	logger.Info("business event", slog.String("kind", "biz"))

	if strings.Contains(filteredOut.String(), "audit event") || !strings.Contains(filteredOut.String(), "business event") {
		t.Fatalf("filtered appender output is wrong: %q", filteredOut.String())
	}
	if !strings.Contains(plainOut.String(), "audit event") || !strings.Contains(plainOut.String(), "business event") {
		t.Fatalf("plain appender output is wrong: %q", plainOut.String())
	}
}

func TestNewConfigured_whenYamlFiltersConfigured_shouldApplyAndValidateRefs(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "logs", "filter.log")
	configPath := filepath.Join(dir, "goark-log.yml")
	writeConfig(t, configPath, `
filters:
  drop-debug:
    type: threshold
    level: info
  drop-health:
    type: regex
    field: message
    pattern: "health"
    onMatch: deny
    onMismatch: neutral
filterRefs: [drop-debug]
appenders:
  file:
    type: file
    fileName: "`+filepath.ToSlash(logPath)+`"
    layout:
      type: text
    filters: [drop-health]
root:
  level: debug
  appenderRefs: [file]
`)
	logger, handler, _, err := NewConfigured(context.Background(), WithConfigPath(configPath))
	if err != nil {
		t.Fatalf("NewConfigured() error = %v", err)
	}
	logger.Debug("hidden debug")
	logger.Info("health probe")
	logger.Info("visible info")
	if err := handler.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	contentBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	content := string(contentBytes)
	if strings.Contains(content, "hidden debug") ||
		strings.Contains(content, "health probe") ||
		!strings.Contains(content, "visible info") {
		t.Fatalf("yaml filter output is wrong: %q", content)
	}
}

func TestNewConfigured_whenFilterRefMissing_shouldReject(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "goark-log.yml")
	writeConfig(t, configPath, `
appenders:
  console:
    type: console
    filters: [missing]
root:
  level: info
  appenderRefs: [console]
`)
	_, _, err := NewConfiguredHandler(context.Background(), WithConfigPath(configPath))
	if err == nil {
		t.Fatalf("NewConfiguredHandler() should reject missing filter ref")
	}
}

func TestNewConfigured_whenFilterDecisionInvalid_shouldReject(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "goark-log.yml")
	writeConfig(t, configPath, `
filters:
  bad:
    type: threshold
    level: info
    onMatch: pass
appenders:
  console:
    type: console
root:
  level: info
  appenderRefs: [console]
  filters: [bad]
`)
	_, _, err := NewConfiguredHandler(context.Background(), WithConfigPath(configPath))
	if err == nil {
		t.Fatalf("NewConfiguredHandler() should reject invalid filter decision")
	}
}
