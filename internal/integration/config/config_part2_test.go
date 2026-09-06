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

func TestLoadOptions_whenTomlConfigProvided_shouldBuildOptions(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "goark-log.toml")
	logPath := filepath.Join(dir, "app.log")
	WriteConfig(t, configPath, `
monitorInterval = "30s"

[properties]
LOG_DIR = "logs"

[appenders.console]
type = "console"
target = "stderr"

[appenders.console.layout]
type = "pattern"
pattern = "%d %-5p %c : %m%attrs%n"

[appenders.file]
type = "file"
fileName = "`+filepath.ToSlash(logPath)+`"
bufferSize = "0"

[appenders.file.layout]
type = "json"
eventEol = true

[filters.audit]
type = "attr"
key = "channel"
value = "audit"
onMatch = "accept"
onMismatch = "neutral"

[root]
level = "info"
appenderRefs = ["console"]

[loggers."goark.audit"]
level = "debug"
appenderRefs = [{ ref = "file", level = "warn", filters = ["audit"] }]
additivity = false
includeLocation = true
`)

	options, result, err := LoadOptions(context.Background(), WithConfigPath(configPath))
	if err != nil {
		t.Fatalf("LoadOptions() error = %v", err)
	}
	defer CloseAppenderList(options.Appenders)
	if result.Source != ConfigSourceExplicit || result.Path != configPath {
		t.Fatalf("ConfigResult = %+v, want explicit %q", result, configPath)
	}
	if result.MonitorInterval != 30*time.Second {
		t.Fatalf("MonitorInterval = %v, want 30s", result.MonitorInterval)
	}
	if len(options.Appenders) != 2 {
		t.Fatalf("len(Appenders) = %d, want 2", len(options.Appenders))
	}
	if len(options.Filters) != 0 {
		t.Fatalf("len(Filters) = %d, want 0", len(options.Filters))
	}
	if got := options.Root.Level; got != slog.LevelInfo {
		t.Fatalf("Root.Level = %v, want info", got)
	}
	if refs := options.Root.AppenderRefs; len(refs) != 1 || refs[0] != "console" {
		t.Fatalf("Root.AppenderRefs = %v, want [console]", refs)
	}
	if len(options.Loggers) != 1 {
		t.Fatalf("len(Loggers) = %d, want 1", len(options.Loggers))
	}
	logger := options.Loggers[0]
	if logger.Name != "goark.audit" {
		t.Fatalf("Logger.Name = %q, want goark.audit", logger.Name)
	}
	if logger.Level == nil || *logger.Level != slog.LevelDebug {
		t.Fatalf("Logger.Level = %v, want debug", logger.Level)
	}
	if !logger.AdditivitySet || logger.Additivity {
		t.Fatalf(
			"Logger additivity = set:%v value:%v, want set false",
			logger.AdditivitySet,
			logger.Additivity,
		)
	}
	if logger.IncludeLocation == nil || !*logger.IncludeLocation {
		t.Fatalf("Logger.IncludeLocation = %v, want true", logger.IncludeLocation)
	}
	if len(logger.AppenderRefs) != 0 {
		t.Fatalf("Logger.AppenderRefs = %v, want structured refs only", logger.AppenderRefs)
	}
	if len(logger.AppenderRefControls) != 1 {
		t.Fatalf("len(Logger.AppenderRefControls) = %d, want 1", len(logger.AppenderRefControls))
	}
	control := logger.AppenderRefControls[0]
	if control.Ref != "file" {
		t.Fatalf("AppenderRefControls[0].Ref = %q, want file", control.Ref)
	}
	if control.Level == nil || *control.Level != slog.LevelWarn {
		t.Fatalf("AppenderRefControls[0].Level = %v, want warn", control.Level)
	}
	if len(control.Filters) != 1 {
		t.Fatalf("len(AppenderRefControls[0].Filters) = %d, want 1", len(control.Filters))
	}
}

func TestNewConfigured_whenYamlDefinesAsyncFileAndNamedLogger_shouldRouteThroughAsync(
	t *testing.T,
) {
	ctx := context.Background()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "logs", "orm.log")
	configPath := filepath.Join(dir, "goark-log.yml")
	WriteConfig(t, configPath, fmt.Sprintf(`
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
	if strings.Contains(string(content), "hidden root") ||
		!strings.Contains(string(content), "msg=\"sql prepared\"") {
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
	if strings.Contains(string(content), "hidden before reload") ||
		!strings.Contains(string(content), "visible after reload") {
		t.Fatalf("reload output is wrong: %q", string(content))
	}
}

func TestLoadOptions_whenXmlCustomLevelConfigured_shouldRegister(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "goark-log.xml")
	WriteConfig(t, configPath, `
<Configuration>
  <CustomLevels>
    <CustomLevel name="AUDIT_XML" intLevel="5"/>
  </CustomLevels>
  <Appenders>
    <Console name="console"/>
  </Appenders>
  <Loggers>
    <Root level="audit_xml">
      <AppenderRef ref="console"/>
    </Root>
  </Loggers>
</Configuration>
`)

	options, _, err := LoadOptions(ctx, WithConfigPath(configPath))
	if err != nil {
		t.Fatalf("LoadOptions() error = %v", err)
	}
	if options.Root.Level != slog.Level(5) {
		t.Fatalf("root level = %d, want 5", options.Root.Level)
	}
}

func TestLoadOptions_whenPropertiesCustomLevelConfigured_shouldRegister(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "goark-log.properties")
	WriteConfig(t, configPath, `
customLevel.AUDIT_PROP = 7
appender.console.type = console
rootLogger.level = audit_prop
rootLogger.appenderRefs = console
`)

	options, _, err := LoadOptions(ctx, WithConfigPath(configPath))
	if err != nil {
		t.Fatalf("LoadOptions() error = %v", err)
	}
	if options.Root.Level != slog.Level(7) {
		t.Fatalf("root level = %d, want 7", options.Root.Level)
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

func TestNewConfigured_whenAppenderRefBlank_shouldReject(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "goark-log.yml")
	WriteConfig(t, configPath, `
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

func assertConfigSource(t *testing.T, result *ConfigResult, source ConfigSource, path string) {
	t.Helper()
	if result.Source != source || result.Path != filepath.Clean(path) {
		t.Fatalf("result = %+v, want source=%s path=%s", result, source, filepath.Clean(path))
	}
}
