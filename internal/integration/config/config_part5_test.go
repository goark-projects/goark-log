package integration

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "goark.dev/log/internal/testsupport"
)

func TestNewConfigured_whenYamlAppenderRefControlsConfigured_shouldApplyPerAppender(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	allPath := filepath.Join(dir, "logs", "all.log")
	errorPath := filepath.Join(dir, "logs", "errors.log")
	auditPath := filepath.Join(dir, "logs", "audit.log")
	configPath := filepath.Join(dir, "goark-log.yml")
	WriteConfig(t, configPath, fmt.Sprintf(`
filters:
  audit-only:
    type: attr
    key: kind
    value: audit
    onMatch: accept
    onMismatch: deny
appenders:
  all:
    type: file
    fileName: %q
    layout:
      type: text
  errors:
    type: file
    fileName: %q
    layout:
      type: text
  audit:
    type: file
    fileName: %q
    layout:
      type: text
root:
  level: debug
  appenderRefs:
    - all
    - ref: errors
      level: error
    - ref: audit
      filters: [audit-only]
`, filepath.ToSlash(allPath), filepath.ToSlash(errorPath), filepath.ToSlash(auditPath)))

	logger, handler, _, err := NewConfigured(ctx, WithConfigPath(configPath))
	if err != nil {
		t.Fatalf("NewConfigured() error = %v", err)
	}
	logger.Info("business event", slog.String("kind", "biz"))
	logger.Info("audit event", slog.String("kind", "audit"))
	logger.Error("error event")
	if err := handler.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	allContent := ReadTextFile(t, allPath)
	if !strings.Contains(allContent, "business event") ||
		!strings.Contains(allContent, "audit event") ||
		!strings.Contains(allContent, "error event") {
		t.Fatalf("all appender content = %q, want every event", allContent)
	}
	errorContent := ReadTextFile(t, errorPath)
	if strings.Contains(errorContent, "business event") ||
		strings.Contains(errorContent, "audit event") ||
		!strings.Contains(errorContent, "error event") {
		t.Fatalf("error appender content = %q, want only error event", errorContent)
	}
	auditContent := ReadTextFile(t, auditPath)
	if strings.Contains(auditContent, "business event") ||
		!strings.Contains(auditContent, "audit event") ||
		strings.Contains(auditContent, "error event") {
		t.Fatalf("audit appender content = %q, want only audit event", auditContent)
	}
}

func TestNewConfigured_whenLog4jStyleConfigurationWrapperUsed_shouldBuildGoYamlExperience(
	t *testing.T,
) {
	ctx := context.Background()
	dir := t.TempDir()
	logDir := filepath.ToSlash(filepath.Join(dir, "logs"))
	configPath := filepath.Join(dir, "goark-log.yml")
	WriteConfig(t, configPath, fmt.Sprintf(`
configuration:
  status: warn
  properties:
    LOG_DIR: %q
    LOG_PATTERN: "%%d{yyyy-MM-dd HH:mm:ss.SSS} %%5p %%c %%X{trace_id} %%m%%n"
  asyncLogger:
    enabled: true
    queueSize: 32
    batchSize: 8
    overflowStrategy: block
  filters:
    keep-info:
      type: threshold
      level: info
  appenders:
    rolling:
      type: rollingFile
      fileName: "${prop:LOG_DIR}/app.log"
      bufferSize: 64KiB
      layout:
        type: pattern
        pattern: "${prop:LOG_PATTERN}"
      rolling:
        filePattern: "${prop:LOG_DIR}/archive/app-%%d{yyyyMMdd}-%%i.log.gz"
        maxSize: 1MiB
        maxBackups: 7
  root:
    level: debug
    appenderRefs: [rolling]
    filters: [keep-info]
  loggers:
    goark.orm:
      level: debug
      appenderRefs: [rolling]
      additivity: false
`, logDir))

	_, handler, result, err := NewConfigured(ctx, WithConfigPath(configPath))
	if err != nil {
		t.Fatalf("NewConfigured() error = %v", err)
	}
	assertConfigSource(t, result, ConfigSourceExplicit, configPath)
	root := NewLogger(handler, "goark")
	root.Debug("hidden by root filter")
	orm := NewLogger(handler, "goark.orm.mapper")
	orm.Info("sql done", slog.String("trace_id", "trace-42"))
	if err := handler.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "logs", "app.log"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if strings.Contains(string(content), "hidden by root filter") ||
		!strings.Contains(string(content), "INFO goark.orm.mapper trace-42 sql done") {
		t.Fatalf("configuration wrapper output is wrong: %q", string(content))
	}
}

func TestNewConfigured_whenJsonConfigurationUsed_shouldBuild(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "logs", "json.log")
	configPath := filepath.Join(dir, "goark-log.json")
	WriteConfig(t, configPath, fmt.Sprintf(`{
  "configuration": {
    "monitorInterval": "0",
    "appenders": {
      "file": {
        "type": "file",
        "fileName": %q,
        "flushOnWrite": true,
        "layout": {"type": "text"}
      }
    },
    "root": {
      "level": "info",
      "appenderRefs": ["file"]
    }
  }
}`, filepath.ToSlash(logPath)))

	logger, handler, result, err := NewConfigured(ctx, WithConfigPath(configPath))
	if err != nil {
		t.Fatalf("NewConfigured() error = %v", err)
	}
	if result.MonitorInterval != 0 {
		t.Fatalf("MonitorInterval = %v, want disabled", result.MonitorInterval)
	}
	logger.Info("json config")
	if err := handler.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if content := ReadTextFile(t, logPath); !strings.Contains(content, "json config") {
		t.Fatalf("json config output is wrong: %q", content)
	}
}

func TestNewConfigured_whenJsonAppenderFileConfigured_shouldUseDirectJSONWriter(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "logs", "direct.json")
	configPath := filepath.Join(dir, "goark-log.yml")
	WriteConfig(t, configPath, fmt.Sprintf(`
appenders:
  json:
    type: json
    fileName: %q
    bufferSize: 64KiB
    flushOnWrite: true
root:
  level: info
  appenderRefs: [json]
`, filepath.ToSlash(logPath)))

	logger, handler, _, err := NewConfigured(ctx, WithConfigPath(configPath))
	if err != nil {
		t.Fatalf("NewConfigured() error = %v", err)
	}
	logger.Info("direct json config", slog.String("profile", "bench"))
	if err := handler.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	content := ReadTextFile(t, logPath)
	if !strings.Contains(content, `"msg":"direct json config"`) ||
		!strings.Contains(content, `"profile":"bench"`) {
		t.Fatalf("direct JSON config output is wrong: %q", content)
	}
}

func TestNewConfigured_whenRollingFileIndexUnsupported_shouldReject(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "goark-log.yml")
	WriteConfig(t, configPath, `
appenders:
  rolling:
    type: rollingFile
    fileName: app.log
    rolling:
      maxSize: 1MiB
      strategy:
        fileIndex: middle
root:
  level: info
  appenderRefs: [rolling]
`)
	_, _, err := NewConfiguredHandler(context.Background(), WithConfigPath(configPath))
	if err == nil || !strings.Contains(err.Error(), "fileIndex") {
		t.Fatalf("NewConfiguredHandler() error = %v, want fileIndex rejection", err)
	}
}

func TestNewConfigured_whenAppenderRefFilterMissing_shouldReject(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "goark-log.yml")
	WriteConfig(t, configPath, `
appenders:
  console:
    type: console
root:
  level: info
  appenderRefs:
    - ref: console
      filters: [missing]
`)
	_, _, err := NewConfiguredHandler(context.Background(), WithConfigPath(configPath))
	if err == nil || !strings.Contains(err.Error(), "filter") {
		t.Fatalf("NewConfiguredHandler() error = %v, want appender ref filter rejection", err)
	}
}

func configuredFileJSONLayout(t *testing.T, appenders []Appender, name string) JSONLayout {
	t.Helper()
	for _, appender := range appenders {
		file, ok := appender.(*FileAppender)
		if !ok || file.Name() != name {
			continue
		}
		layout, ok := file.Layout().(JSONLayout)
		if !ok {
			t.Fatalf("file layout type = %T, want JSONLayout", file.Layout())
		}
		return layout
	}
	t.Fatalf("file appender %q was not built: %+v", name, appenders)
	return JSONLayout{}
}

func writeLevelConfig(t *testing.T, dir string, name string, level string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	WriteConfig(t, path, fmt.Sprintf(`
appenders:
  console:
    type: console
root:
  level: %s
  appenderRefs: [console]
`, level))
	return path
}
