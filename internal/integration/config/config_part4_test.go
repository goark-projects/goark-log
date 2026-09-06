package integration

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "goark.dev/log/internal/testsupport"
)

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
	old := FixedTestTime().Add(-48 * 60 * 60 * 1e9)
	if err := os.Chtimes(expired, old, old); err != nil {
		t.Fatalf("Chtimes(expired) error = %v", err)
	}
	configPath := filepath.Join(dir, "goark-log.yml")
	WriteConfig(t, configPath, fmt.Sprintf(`
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

func TestLoggerContext_whenMonitorIntervalConfigured_shouldReloadChangedFile(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "logs", "monitor.log")
	configPath := filepath.Join(dir, "goark-log.yml")
	writeMonitoredFileConfig(t, configPath, logPath, "error")
	originalInfo, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}

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
	if err := os.Chtimes(configPath, originalInfo.ModTime(), originalInfo.ModTime()); err != nil {
		t.Fatalf("Chtimes() error = %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		logger.Debug("visible after monitor reload")
		if strings.Contains(ReadTextFile(t, logPath), "visible after monitor reload") {
			return
		}
		time.Sleep(30 * time.Millisecond)
	}
	t.Fatalf("monitor reload did not make debug log visible, content=%q", ReadTextFile(t, logPath))
}

func TestNewConfigured_whenPropertiesConfigurationUsed_shouldBuild(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "logs", "properties.log")
	configPath := filepath.Join(dir, "goark-log.properties")
	WriteConfig(t, configPath, fmt.Sprintf(`
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
	if content := ReadTextFile(t, logPath); !strings.Contains(content, "properties config") {
		t.Fatalf("properties config output is wrong: %q", content)
	}
}

func TestLoadOptions_whenDefaultTomlExists_shouldLoad(t *testing.T) {
	dir := t.TempDir()
	confDir := filepath.Join(dir, "conf")
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	path := filepath.Join(confDir, "goark-log.toml")
	if err := os.WriteFile(path, []byte("root.level = \"info\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	options, result, err := LoadOptions(context.Background(), WithConfigWorkingDir(dir))
	if err != nil {
		t.Fatalf("LoadOptions() error = %v", err)
	}
	defer CloseAppenderList(options.Appenders)
	if result.Source != ConfigSourceFile {
		t.Fatalf("ConfigResult.Source = %q, want file", result.Source)
	}
	if got := options.Root.Level; got != slog.LevelInfo {
		t.Fatalf("Root.Level = %v, want info", got)
	}
}

func TestNewConfigured_whenAsyncAppenderRefBlank_shouldReject(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "goark-log.yml")
	WriteConfig(t, configPath, `
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
	if err == nil ||
		!strings.Contains(err.Error(), `async appender "async" appender ref is empty`) {
		t.Fatalf("NewConfiguredHandler() error = %v, want blank async appender ref rejection", err)
	}
}

func configuredFilePatternLayout(t *testing.T, appenders []Appender, name string) *PatternLayout {
	t.Helper()
	for _, appender := range appenders {
		file, ok := appender.(*FileAppender)
		if !ok || file.Name() != name {
			continue
		}
		layout, ok := file.Layout().(*PatternLayout)
		if !ok {
			t.Fatalf("file layout type = %T, want *PatternLayout", file.Layout())
		}
		return layout
	}
	t.Fatalf("file appender %q was not built: %+v", name, appenders)
	return nil
}

func assertFullLayoutOptions(t *testing.T, options LayoutOptions) {
	t.Helper()
	if !options.Compact ||
		!options.EventEOL ||
		!options.Complete ||
		!options.IncludeStacktrace ||
		!options.StacktraceAsString ||
		!options.PropertiesAsList ||
		!options.IncludeNullDelimiter ||
		options.Header != "H" ||
		options.Footer != "F" {
		t.Fatalf("layout options = %+v, want every option populated", options)
	}
}
