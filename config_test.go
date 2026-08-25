package goarklog

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func writeConfig(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
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
