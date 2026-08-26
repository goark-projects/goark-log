package goarklog

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
)

func TestNativeLogger_whenRootIncludeLocationEnabled_shouldCaptureCallSite(t *testing.T) {
	appender := newRecordingAppender("memory")
	handler, err := NewHandler(Options{
		Appenders: []Appender{appender},
		Root: RootLogger{
			Level:           slog.LevelInfo,
			AppenderRefs:    []string{"memory"},
			IncludeLocation: true,
		},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	defer handler.Close()

	logger, err := NewNativeLogger(handler, "goark.location")
	if err != nil {
		t.Fatalf("NewNativeLogger() error = %v", err)
	}
	if err := logger.Info("root location"); err != nil {
		t.Fatalf("Info() error = %v", err)
	}

	assertRecordingAppenderCaller(t, appender, "TestNativeLogger_whenRootIncludeLocationEnabled")
}

func TestNativeLogger_whenAppenderRefIncludeLocationDisabled_shouldClearOnlyThatAppender(t *testing.T) {
	locationAppender := newRecordingAppender("location")
	plainAppender := newRecordingAppender("plain")
	handler, err := NewHandler(Options{
		Appenders: []Appender{locationAppender, plainAppender},
		Root: RootLogger{
			Level:           slog.LevelInfo,
			AppenderRefs:    []string{"location"},
			IncludeLocation: true,
			AppenderRefControls: []AppenderRef{
				NewAppenderRef("plain", WithAppenderRefLocation(false)),
			},
		},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	defer handler.Close()

	logger, err := NewNativeLogger(handler, "goark.location")
	if err != nil {
		t.Fatalf("NewNativeLogger() error = %v", err)
	}
	if err := logger.Info("split location"); err != nil {
		t.Fatalf("Info() error = %v", err)
	}

	assertRecordingAppenderCaller(t, locationAppender, "TestNativeLogger_whenAppenderRefIncludeLocationDisabled")
	events := plainAppender.Events()
	if len(events) != 1 {
		t.Fatalf("plain event count = %d, want 1", len(events))
	}
	if events[0].PC != 0 {
		t.Fatalf("plain appender PC = %d, want 0", events[0].PC)
	}
}

func TestLoadOptions_whenIncludeLocationConfigured_shouldPopulateRootLoggerAndAppenderRef(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "goark-log.yml")
	writeConfig(t, configPath, `
appenders:
  console:
    type: console
root:
  level: info
  includeLocation: true
  appenderRefs:
    - ref: console
      includeLocation: false
loggers:
  goark.orm:
    level: debug
    includeLocation: true
    appenderRefs:
      - ref: console
        includeLocation: true
    additivity: false
`)
	options, _, err := LoadOptions(context.Background(), WithConfigPath(configPath))
	if err != nil {
		t.Fatalf("LoadOptions() error = %v", err)
	}
	if !options.Root.IncludeLocation {
		t.Fatalf("root IncludeLocation = false, want true")
	}
	if len(options.Root.AppenderRefControls) != 1 || options.Root.AppenderRefControls[0].IncludeLocation == nil || *options.Root.AppenderRefControls[0].IncludeLocation {
		t.Fatalf("root appender ref includeLocation = %+v, want explicit false", options.Root.AppenderRefControls)
	}
	if len(options.Loggers) != 1 || options.Loggers[0].IncludeLocation == nil || !*options.Loggers[0].IncludeLocation {
		t.Fatalf("logger includeLocation = %+v, want explicit true", options.Loggers)
	}
	if len(options.Loggers[0].AppenderRefControls) != 1 || options.Loggers[0].AppenderRefControls[0].IncludeLocation == nil || !*options.Loggers[0].AppenderRefControls[0].IncludeLocation {
		t.Fatalf("logger appender ref includeLocation = %+v, want explicit true", options.Loggers[0].AppenderRefControls)
	}
}

func TestLoadOptions_whenPropertiesAppenderRefIncludeLocationConfigured_shouldPopulateControls(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "goark-log.properties")
	writeConfig(t, configPath, `
appender.console.type=console
rootLogger.level=info
rootLogger.includeLocation=true
rootLogger.appenderRef.console.ref=console
rootLogger.appenderRef.console.includeLocation=false
logger.orm.name=goark.orm
logger.orm.level=debug
logger.orm.includeLocation=true
logger.orm.appenderRef.console.ref=console
logger.orm.appenderRef.console.includeLocation=true
logger.orm.additivity=false
`)
	options, _, err := LoadOptions(context.Background(), WithConfigPath(configPath))
	if err != nil {
		t.Fatalf("LoadOptions() error = %v", err)
	}
	if !options.Root.IncludeLocation {
		t.Fatalf("properties root IncludeLocation = false, want true")
	}
	if len(options.Root.AppenderRefControls) != 1 || options.Root.AppenderRefControls[0].IncludeLocation == nil || *options.Root.AppenderRefControls[0].IncludeLocation {
		t.Fatalf("properties root appender refs = %+v, want explicit false", options.Root.AppenderRefControls)
	}
	if len(options.Loggers) != 1 || options.Loggers[0].IncludeLocation == nil || !*options.Loggers[0].IncludeLocation {
		t.Fatalf("properties logger includeLocation = %+v, want explicit true", options.Loggers)
	}
}

func TestNewConfigured_whenPropertyShorthandLookupUsed_shouldExpandProperty(t *testing.T) {
	dir := t.TempDir()
	logDir := filepath.ToSlash(filepath.Join(dir, "logs"))
	configPath := filepath.Join(dir, "goark-log.yml")
	writeConfig(t, configPath, `
properties:
  LOG_DIR: "`+logDir+`"
  LOG_PATTERN: "%p %c %m%n"
appenders:
  file:
    type: file
    fileName: "${LOG_DIR}/lookup.log"
    layout:
      type: pattern
      pattern: "${LOG_PATTERN}"
root:
  level: info
  appenderRefs: [file]
`)
	logger, handler, _, err := NewConfigured(context.Background(), WithConfigPath(configPath))
	if err != nil {
		t.Fatalf("NewConfigured() error = %v", err)
	}
	logger.Info("property shorthand")
	if err := handler.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	content := readTextFile(t, filepath.Join(dir, "logs", "lookup.log"))
	if !strings.Contains(content, "INFO goark property shorthand") {
		t.Fatalf("property shorthand output is wrong: %q", content)
	}
}

func TestNewConfigured_whenPropertiesBoolInvalid_shouldReportFieldAndValue(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "goark-log.properties")
	writeConfig(t, configPath, `
appender.console.type=console
rootLogger.level=info
rootLogger.includeLocation=maybe
rootLogger.appenderRefs=console
`)
	_, _, err := NewConfiguredHandler(context.Background(), WithConfigPath(configPath))
	if err == nil {
		t.Fatalf("NewConfiguredHandler() should reject invalid boolean")
	}
	for _, want := range []string{"rootLogger.includeLocation", `"maybe"`, "invalid boolean"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error should contain %q, got %v", want, err)
		}
	}
}

func assertRecordingAppenderCaller(t *testing.T, appender *recordingAppender, methodPrefix string) {
	t.Helper()
	events := appender.Events()
	if len(events) != 1 {
		t.Fatalf("event count = %d, want 1", len(events))
	}
	frame := callerFrameFromPC(events[0].PC)
	if !strings.Contains(frame.method, methodPrefix) {
		t.Fatalf("caller method = %q, want %q", frame.method, methodPrefix)
	}
}
