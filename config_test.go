package goarklog

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadOptions_whenMultipleSourcesAvailable_shouldUsePriority(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	explicitPath := writeLevelConfig(t, dir, "explicit.yml", "error")
	envPath := writeLevelConfig(t, dir, "env.yml", "warn")
	bootPath := writeLevelConfig(t, dir, "boot.yml", "debug")
	confDir := filepath.Join(dir, "conf")
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	writeLevelConfig(t, confDir, "goark-log.yml", "info")

	t.Setenv(EnvConfigPath, envPath)
	options, result, err := LoadOptions(ctx,
		WithConfigPath(explicitPath),
		WithBootPropertyResolver(PropertyMap{"goark.log.config": bootPath}),
		WithConfigWorkingDir(dir),
	)
	if err != nil {
		t.Fatalf("LoadOptions(explicit) error = %v", err)
	}
	assertConfigSource(t, result, ConfigSourceExplicit, explicitPath)
	assertRootLevel(t, options, slog.LevelError)

	options, result, err = LoadOptions(ctx,
		WithBootPropertyResolver(PropertyMap{"goark.log.config": bootPath}),
		WithConfigWorkingDir(dir),
	)
	if err != nil {
		t.Fatalf("LoadOptions(env) error = %v", err)
	}
	assertConfigSource(t, result, ConfigSourceEnv, envPath)
	assertRootLevel(t, options, slog.LevelWarn)

	t.Setenv(EnvConfigPath, "")
	options, result, err = LoadOptions(ctx,
		WithBootPropertyResolver(PropertyMap{"goark.log.config": bootPath}),
		WithConfigWorkingDir(dir),
	)
	if err != nil {
		t.Fatalf("LoadOptions(boot) error = %v", err)
	}
	assertConfigSource(t, result, ConfigSourceBoot, bootPath)
	assertRootLevel(t, options, slog.LevelDebug)

	options, result, err = LoadOptions(ctx, WithConfigWorkingDir(dir))
	if err != nil {
		t.Fatalf("LoadOptions(file) error = %v", err)
	}
	assertConfigSource(t, result, ConfigSourceFile, filepath.Join(confDir, "goark-log.yml"))
	assertRootLevel(t, options, slog.LevelInfo)

	options, result, err = LoadOptions(ctx, WithConfigWorkingDir(filepath.Join(dir, "empty")))
	if err != nil {
		t.Fatalf("LoadOptions(default) error = %v", err)
	}
	if result.Source != ConfigSourceDefault || result.Path != "" {
		t.Fatalf("default result = %+v", result)
	}
	assertRootLevel(t, options, slog.LevelInfo)
}

func TestNewConfigured_whenYamlDefinesAsyncFileAndNamedLogger_shouldRouteThroughAsync(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "logs", "orm.log")
	configPath := filepath.Join(dir, "goark-log.yml")
	writeConfig(t, configPath, fmt.Sprintf(`
appenders:
  file:
    type: file
    fileName: %q
    layout:
      type: text
  async:
    type: async
    appenderRefs: [file]
    queueSize: 8
    overflowStrategy: block
    waitStrategy: sleep
root:
  level: error
  appenderRefs: [async]
loggers:
  goark.orm:
    level: debug
    appenderRefs: [async]
    additivity: false
`, filepath.ToSlash(logPath)))

	_, handler, result, err := NewConfigured(ctx, WithConfigPath(configPath))
	if err != nil {
		t.Fatalf("NewConfigured() error = %v", err)
	}
	assertConfigSource(t, result, ConfigSourceExplicit, configPath)
	root := NewLogger(handler, "goark")
	root.Info("hidden root")
	orm := NewLogger(handler, "goark.orm.mapper")
	orm.Debug("sql prepared")
	if err := handler.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if strings.Contains(string(content), "hidden root") || !strings.Contains(string(content), "msg=\"sql prepared\"") {
		t.Fatalf("log content routing is wrong: %q", string(content))
	}
}

func TestNewConfigured_whenLog4jStyleConfigurationWrapperUsed_shouldBuildGoYamlExperience(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	logDir := filepath.ToSlash(filepath.Join(dir, "logs"))
	configPath := filepath.Join(dir, "goark-log.yml")
	writeConfig(t, configPath, fmt.Sprintf(`
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
	writeConfig(t, configPath, fmt.Sprintf(`{
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
	if content := readTextFile(t, logPath); !strings.Contains(content, "json config") {
		t.Fatalf("json config output is wrong: %q", content)
	}
}

func TestNewConfigured_whenXmlConfigurationUsed_shouldBuild(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "logs", "xml.log")
	configPath := filepath.Join(dir, "goark-log.xml")
	writeConfig(t, configPath, fmt.Sprintf(`
<Configuration status="warn" monitorInterval="0">
  <Appenders>
    <File name="file" fileName="%s" flushOnWrite="true">
      <TextLayout/>
    </File>
  </Appenders>
  <Loggers>
    <Root level="info">
      <AppenderRef ref="file"/>
    </Root>
  </Loggers>
</Configuration>
`, filepath.ToSlash(logPath)))

	logger, handler, result, err := NewConfigured(ctx, WithConfigPath(configPath))
	if err != nil {
		t.Fatalf("NewConfigured() error = %v", err)
	}
	if result.MonitorInterval != 0 {
		t.Fatalf("MonitorInterval = %v, want disabled", result.MonitorInterval)
	}
	logger.Info("xml config")
	if err := handler.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if content := readTextFile(t, logPath); !strings.Contains(content, "xml config") {
		t.Fatalf("xml config output is wrong: %q", content)
	}
}

func TestNewConfigured_whenPropertiesConfigurationUsed_shouldBuild(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "logs", "properties.log")
	configPath := filepath.Join(dir, "goark-log.properties")
	writeConfig(t, configPath, fmt.Sprintf(`
status = warn
monitorInterval = 0
appender.file.type = file
appender.file.fileName = %s
appender.file.flushOnWrite = true
appender.file.layout.type = text
rootLogger.level = info
rootLogger.appenderRefs = file
`, filepath.ToSlash(logPath)))

	logger, handler, result, err := NewConfigured(ctx, WithConfigPath(configPath))
	if err != nil {
		t.Fatalf("NewConfigured() error = %v", err)
	}
	if result.MonitorInterval != 0 {
		t.Fatalf("MonitorInterval = %v, want disabled", result.MonitorInterval)
	}
	logger.Info("properties config")
	if err := handler.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if content := readTextFile(t, logPath); !strings.Contains(content, "properties config") {
		t.Fatalf("properties config output is wrong: %q", content)
	}
}

func TestNewConfigured_whenRollingPoliciesAndStrategyUsed_shouldBuildLog4jStyleYaml(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	logDir := filepath.ToSlash(filepath.Join(dir, "logs"))
	archiveDir := filepath.Join(dir, "logs", "archive")
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	expired := filepath.Join(archiveDir, "expired.log.gz")
	if err := os.WriteFile(expired, []byte("expired"), 0o644); err != nil {
		t.Fatalf("WriteFile(expired) error = %v", err)
	}
	old := fixedTestTime().Add(-48 * 60 * 60 * 1e9)
	if err := os.Chtimes(expired, old, old); err != nil {
		t.Fatalf("Chtimes(expired) error = %v", err)
	}
	configPath := filepath.Join(dir, "goark-log.yml")
	writeConfig(t, configPath, fmt.Sprintf(`
configuration:
  properties:
    LOG_DIR: %q
  appenders:
    rolling:
      type: rollingFile
      fileName: "${prop:LOG_DIR}/app.log"
      layout:
        type: text
      rolling:
        filePattern: "${prop:LOG_DIR}/archive/app-%%d{yyyyMMdd}-%%i.log.gz"
        policies:
          size:
            size: 120
          time:
            interval: daily
            modulate: true
          startup:
            enabled: true
        strategy:
          max: 2
          compression:
            gzip: true
            async: true
          delete:
            basePath: "${prop:LOG_DIR}/archive"
            maxDepth: 1
            ifFileName:
              glob: "*.log.gz"
            ifLastModified:
              age: 24h
            async: true
  root:
    level: info
    appenderRefs: [rolling]
`, logDir))

	logger, handler, _, err := NewConfigured(ctx, WithConfigPath(configPath))
	if err != nil {
		t.Fatalf("NewConfigured() error = %v", err)
	}
	for index := 0; index < 3; index++ {
		logger.Info("strategy " + strings.Repeat("x", 80))
	}
	if err := handler.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if _, err := os.Stat(expired); !os.IsNotExist(err) {
		t.Fatalf("expired archive should be deleted by strategy action, stat error = %v", err)
	}
	archives, err := filepath.Glob(filepath.Join(archiveDir, "app-*.gz"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(archives) == 0 || len(archives) > 2 {
		t.Fatalf("strategy retained archives = %d, want 1..2: %v", len(archives), archives)
	}
}

func TestConfigReloader_whenConfigChanges_shouldSwapHandlerOptions(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "logs", "reload.log")
	configPath := filepath.Join(dir, "goark-log.yml")
	writeFileConfig(t, configPath, logPath, "error")

	logger, handler, _, err := NewConfigured(ctx, WithConfigPath(configPath))
	if err != nil {
		t.Fatalf("NewConfigured() error = %v", err)
	}
	logger.Info("hidden before reload")

	writeFileConfig(t, configPath, logPath, "debug")
	reloader, err := NewConfigReloader(handler, WithConfigPath(configPath))
	if err != nil {
		t.Fatalf("NewConfigReloader() error = %v", err)
	}
	if _, err := reloader.Reload(ctx); err != nil {
		t.Fatalf("Reload() error = %v", err)
	}
	logger.Debug("visible after reload")
	if err := handler.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if strings.Contains(string(content), "hidden before reload") || !strings.Contains(string(content), "visible after reload") {
		t.Fatalf("reload output is wrong: %q", string(content))
	}
}

func TestLoggerContext_whenMonitorIntervalConfigured_shouldReloadChangedFile(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "logs", "monitor.log")
	configPath := filepath.Join(dir, "goark-log.yml")
	writeMonitoredFileConfig(t, configPath, logPath, "error")

	context, result, err := NewConfiguredLoggerContext(ctx, WithConfigPath(configPath))
	if err != nil {
		t.Fatalf("NewConfiguredLoggerContext() error = %v", err)
	}
	defer context.Close()
	if result.MonitorInterval <= 0 {
		t.Fatalf("MonitorInterval = %v, want enabled", result.MonitorInterval)
	}
	logger := context.Logger("goark.monitor")
	logger.Debug("hidden before monitor reload")

	writeMonitoredFileConfig(t, configPath, logPath, "debug")
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		logger.Debug("visible after monitor reload")
		if strings.Contains(readTextFile(t, logPath), "visible after monitor reload") {
			return
		}
		time.Sleep(30 * time.Millisecond)
	}
	t.Fatalf("monitor reload did not make debug log visible, content=%q", readTextFile(t, logPath))
}

func TestNewConfigured_whenYamlAppenderRefControlsConfigured_shouldApplyPerAppender(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	allPath := filepath.Join(dir, "logs", "all.log")
	errorPath := filepath.Join(dir, "logs", "errors.log")
	auditPath := filepath.Join(dir, "logs", "audit.log")
	configPath := filepath.Join(dir, "goark-log.yml")
	writeConfig(t, configPath, fmt.Sprintf(`
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

	allContent := readTextFile(t, allPath)
	if !strings.Contains(allContent, "business event") ||
		!strings.Contains(allContent, "audit event") ||
		!strings.Contains(allContent, "error event") {
		t.Fatalf("all appender content = %q, want every event", allContent)
	}
	errorContent := readTextFile(t, errorPath)
	if strings.Contains(errorContent, "business event") ||
		strings.Contains(errorContent, "audit event") ||
		!strings.Contains(errorContent, "error event") {
		t.Fatalf("error appender content = %q, want only error event", errorContent)
	}
	auditContent := readTextFile(t, auditPath)
	if strings.Contains(auditContent, "business event") ||
		!strings.Contains(auditContent, "audit event") ||
		strings.Contains(auditContent, "error event") {
		t.Fatalf("audit appender content = %q, want only audit event", auditContent)
	}
}

func TestNewConfigured_whenAppenderRefMissing_shouldReject(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "goark-log.yml")
	writeConfig(t, configPath, `
appenders:
  console:
    type: console
root:
  level: info
  appenderRefs: [missing]
`)
	_, _, err := NewConfiguredHandler(context.Background(), WithConfigPath(configPath))
	if err == nil {
		t.Fatalf("NewConfiguredHandler() should reject missing appender ref")
	}
}

func TestNewConfigured_whenAppenderRefFilterMissing_shouldReject(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "goark-log.yml")
	writeConfig(t, configPath, `
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

func TestNewConfigured_whenRollingFileIndexUnsupported_shouldReject(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "goark-log.yml")
	writeConfig(t, configPath, `
appenders:
  rolling:
    type: rollingFile
    fileName: app.log
    rolling:
      maxSize: 1MiB
      strategy:
        fileIndex: min
root:
  level: info
  appenderRefs: [rolling]
`)
	_, _, err := NewConfiguredHandler(context.Background(), WithConfigPath(configPath))
	if err == nil || !strings.Contains(err.Error(), "fileIndex") {
		t.Fatalf("NewConfiguredHandler() error = %v, want fileIndex rejection", err)
	}
}

func TestNewConfigured_whenAppenderRefBlank_shouldReject(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "goark-log.yml")
	writeConfig(t, configPath, `
appenders:
  console:
    type: console
root:
  level: info
  appenderRefs: [""]
`)
	_, _, err := NewConfiguredHandler(context.Background(), WithConfigPath(configPath))
	if err == nil || !strings.Contains(err.Error(), "appender ref is empty") {
		t.Fatalf("NewConfiguredHandler() error = %v, want blank appender ref rejection", err)
	}
}

func TestNewConfigured_whenAsyncAppenderRefBlank_shouldReject(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "goark-log.yml")
	writeConfig(t, configPath, `
appenders:
  console:
    type: console
  async:
    type: async
    appenderRefs: [""]
root:
  level: info
  appenderRefs: [async]
`)
	_, _, err := NewConfiguredHandler(context.Background(), WithConfigPath(configPath))
	if err == nil || !strings.Contains(err.Error(), `async appender "async" appender ref is empty`) {
		t.Fatalf("NewConfiguredHandler() error = %v, want blank async appender ref rejection", err)
	}
}

func TestNewConfigured_whenFilterRefBlank_shouldReject(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "goark-log.yml")
	writeConfig(t, configPath, `
appenders:
  console:
    type: console
root:
  level: info
  appenderRefs: [console]
  filters: [""]
`)
	_, _, err := NewConfiguredHandler(context.Background(), WithConfigPath(configPath))
	if err == nil || !strings.Contains(err.Error(), "filter ref is empty") {
		t.Fatalf("NewConfiguredHandler() error = %v, want blank filter ref rejection", err)
	}
}

func TestNewConfigured_whenAppenderFilterRefMissing_shouldCloseBuiltAppender(t *testing.T) {
	registry := NewPluginRegistry()
	var built *recordingAppender
	if err := registry.RegisterAppender("recording", func(config AppenderBuildConfig) (Appender, error) {
		built = newRecordingAppender(config.Name)
		return built, nil
	}); err != nil {
		t.Fatalf("RegisterAppender() error = %v", err)
	}
	configPath := filepath.Join(t.TempDir(), "goark-log.yml")
	writeConfig(t, configPath, `
appenders:
  recording:
    type: recording
    filters: [missing]
root:
  level: info
  appenderRefs: [recording]
`)

	_, _, err := NewConfiguredHandler(context.Background(), WithConfigPath(configPath), WithPluginRegistry(registry))
	if err == nil {
		t.Fatalf("NewConfiguredHandler() should reject missing appender filter ref")
	}
	if built == nil {
		t.Fatalf("recording appender should have been built before filter validation failed")
	}
	if built.CloseCount() != 1 {
		t.Fatalf("built CloseCount() = %d, want 1", built.CloseCount())
	}
}

func TestNewConfigured_whenConfigurationWrapperMixedWithTopLevel_shouldReject(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "goark-log.yml")
	writeConfig(t, configPath, `
configuration:
  appenders:
    console:
      type: console
  root:
    level: info
    appenderRefs: [console]
appenders:
  other:
    type: console
`)
	_, _, err := NewConfiguredHandler(context.Background(), WithConfigPath(configPath))
	if err == nil {
		t.Fatalf("NewConfiguredHandler() should reject mixed configuration wrapper and top-level fields")
	}
}

func TestLoadOptions_whenDefaultTomlExists_shouldRejectUnsupportedFormat(t *testing.T) {
	dir := t.TempDir()
	confDir := filepath.Join(dir, "conf")
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(confDir, "goark-log.toml"), []byte("root.level = \"info\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	_, _, err := LoadOptions(context.Background(), WithConfigWorkingDir(dir))
	if err == nil {
		t.Fatalf("LoadOptions() should reject unsupported TOML config in default search path")
	}
}

func TestParseByteSizeAndRollingInterval(t *testing.T) {
	size, err := ParseByteSize("1.5MiB")
	if err != nil {
		t.Fatalf("ParseByteSize() error = %v", err)
	}
	if size != 1572864 {
		t.Fatalf("ParseByteSize() = %d, want 1572864", size)
	}
	interval, err := ParseRollingInterval("daily")
	if err != nil {
		t.Fatalf("ParseRollingInterval() error = %v", err)
	}
	if interval != 24*60*60*1e9 {
		t.Fatalf("ParseRollingInterval() = %v, want 24h", interval)
	}
}

func writeLevelConfig(t *testing.T, dir string, name string, level string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	writeConfig(t, path, fmt.Sprintf(`
appenders:
  console:
    type: console
root:
  level: %s
  appenderRefs: [console]
`, level))
	return path
}

func writeFileConfig(t *testing.T, configPath string, logPath string, level string) {
	t.Helper()
	writeConfig(t, configPath, fmt.Sprintf(`
appenders:
  file:
    type: file
    fileName: %q
    layout:
      type: text
root:
  level: %s
  appenderRefs: [file]
`, filepath.ToSlash(logPath), level))
}

func writeMonitoredFileConfig(t *testing.T, configPath string, logPath string, level string) {
	t.Helper()
	writeConfig(t, configPath, fmt.Sprintf(`
monitorInterval: 20ms
appenders:
  file:
    type: file
    fileName: %q
    flushOnWrite: true
    layout:
      type: text
root:
  level: %s
  appenderRefs: [file]
`, filepath.ToSlash(logPath), level))
}

func writeConfig(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func readTextFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	return string(content)
}

func assertConfigSource(t *testing.T, result *ConfigResult, source ConfigSource, path string) {
	t.Helper()
	if result.Source != source || result.Path != filepath.Clean(path) {
		t.Fatalf("result = %+v, want source=%s path=%s", result, source, filepath.Clean(path))
	}
}

func assertRootLevel(t *testing.T, options Options, level slog.Level) {
	t.Helper()
	if options.Root.Level != level {
		t.Fatalf("root level = %v, want %v", options.Root.Level, level)
	}
}
