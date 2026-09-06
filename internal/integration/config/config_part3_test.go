package integration

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "goark.dev/log/internal/testsupport"
)

func TestLoadOptions_whenPatternLayoutOptionsConfigured_shouldPopulateANSIControl(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name       string
		configName string
		content    func(logPath string) string
	}{
		{
			name:       "yaml",
			configName: "goark-log.yml",
			content: func(logPath string) string {
				return fmt.Sprintf(`
appenders:
  file:
    type: file
    fileName: %q
    layout:
      type: pattern
      pattern: "%%style{%%m}{red}%%n"
      disableAnsi: true
root:
  level: info
  appenderRefs: [file]
`, filepath.ToSlash(logPath))
			},
		},
		{
			name:       "xml",
			configName: "goark-log.xml",
			content: func(logPath string) string {
				return fmt.Sprintf(`
<Configuration>
  <Appenders>
    <File name="file" fileName="%s">
      <PatternLayout pattern="%%style{%%m}{red}%%n" disableAnsi="true"/>
    </File>
  </Appenders>
  <Loggers>
    <Root level="info">
      <AppenderRef ref="file"/>
    </Root>
  </Loggers>
</Configuration>
`, filepath.ToSlash(logPath))
			},
		},
		{
			name:       "toml",
			configName: "goark-log.toml",
			content: func(logPath string) string {
				return fmt.Sprintf(`
[appenders.file]
type = "file"
fileName = %q

[appenders.file.layout]
type = "pattern"
pattern = "%%style{%%m}{red}%%n"
disableAnsi = true

[root]
level = "info"
appenderRefs = ["file"]
`, filepath.ToSlash(logPath))
			},
		},
		{
			name:       "properties",
			configName: "goark-log.properties",
			content: func(logPath string) string {
				return fmt.Sprintf(`
appender.file.type = file
appender.file.fileName = %s
appender.file.layout.type = pattern
appender.file.layout.pattern = %%style{%%m}{red}%%n
appender.file.layout.disableAnsi = true
rootLogger.level = info
rootLogger.appenderRefs = file
`, filepath.ToSlash(logPath))
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			logPath := filepath.Join(dir, "logs", "app.log")
			configPath := filepath.Join(dir, tc.configName)
			WriteConfig(t, configPath, tc.content(logPath))

			options, _, err := LoadOptions(ctx, WithConfigPath(configPath))
			if err != nil {
				t.Fatalf("LoadOptions() error = %v", err)
			}
			defer CloseAppenderList(options.Appenders)
			layout := configuredFilePatternLayout(t, options.Appenders, "file")
			if !layout.Options().DisableANSI {
				t.Fatalf("pattern layout options = %+v, want DisableANSI", layout.Options())
			}
		})
	}
}

func TestLoadOptions_whenAsyncWaitOptionsConfigured_shouldPopulateRuntimeOptions(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "goark-log.yml")
	WriteConfig(t, configPath, `
asyncLogger:
  enabled: true
  queueSize: 16
  batchSize: 4
  waitStrategy: sleep
  waitRetries: 17
  sleepTime: 250us
  timeout: 3ms
appenders:
  console:
    type: console
  async:
    type: async
    appenderRefs: [console]
    batchSize: 2
    waitStrategy: timeout
    waitRetries: 9
    sleepTime: 1ms
    timeout: 5ms
root:
  level: info
  appenderRefs: [async]
`)
	options, _, err := LoadOptions(ctx, WithConfigPath(configPath))
	if err != nil {
		t.Fatalf("LoadOptions() error = %v", err)
	}
	defer CloseAppenderList(options.Appenders)
	if !options.Async.Enabled ||
		options.Async.WaitStrategy != AsyncWaitSleep ||
		options.Async.WaitOptions.Retries != 17 ||
		options.Async.WaitOptions.SleepTime != 250*time.Microsecond ||
		options.Async.WaitOptions.Timeout != 3*time.Millisecond {
		t.Fatalf("async logger options = %+v", options.Async)
	}
	var asyncAppender *AsyncAppender
	for _, appender := range options.Appenders {
		if candidate, ok := appender.(*AsyncAppender); ok && candidate.Name() == "async" {
			asyncAppender = candidate
			break
		}
	}
	if asyncAppender == nil {
		t.Fatalf("async appender was not built: %+v", options.Appenders)
	}
	waitOptions := asyncAppender.WaitOptions()
	if asyncAppender.BatchSize() != 2 ||
		asyncAppender.WaitStrategy() != AsyncWaitBlock ||
		waitOptions.Retries != 9 ||
		waitOptions.SleepTime != time.Millisecond ||
		waitOptions.Timeout != 5*time.Millisecond {
		t.Fatalf(
			"async appender options = batch %d strategy %s options %+v",
			asyncAppender.BatchSize(),
			asyncAppender.WaitStrategy(),
			waitOptions,
		)
	}
}

func TestNewConfigured_whenJsonTemplateLayoutConfigured_shouldWriteTemplateJson(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "logs", "template.log")
	configPath := filepath.Join(dir, "goark-log.yml")
	WriteConfig(t, configPath, fmt.Sprintf(`
appenders:
  file:
    type: file
    fileName: %q
    flushOnWrite: true
    layout:
      type: jsonTemplate
      eventTemplate: >-
        {"message":{"$resolver":"message"},
        "trace":{"$resolver":"attr","key":"trace_id"}}
root:
  level: info
  appenderRefs: [file]
`, filepath.ToSlash(logPath)))

	logger, handler, _, err := NewConfigured(ctx, WithConfigPath(configPath))
	if err != nil {
		t.Fatalf("NewConfigured() error = %v", err)
	}
	logger.Info("json template config", slog.String("trace_id", "trace-config"))
	if err := handler.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	content := ReadTextFile(t, logPath)
	if !strings.Contains(content, `"message":"json template config"`) ||
		!strings.Contains(content, `"trace":"trace-config"`) {
		t.Fatalf("json template layout output is wrong: %q", content)
	}
}

func TestLoadOptions_whenYamlCustomLevelsConfigured_shouldRegisterBeforeLevelParsing(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "goark-log.yml")
	WriteConfig(t, configPath, `
customLevels:
  AUDIT_YAML: 6
appenders:
  console:
    type: console
root:
  level: audit_yaml
  appenderRefs: [console]
`)

	options, _, err := LoadOptions(ctx, WithConfigPath(configPath))
	if err != nil {
		t.Fatalf("LoadOptions() error = %v", err)
	}
	if options.Root.Level != slog.Level(6) {
		t.Fatalf("root level = %d, want 6", options.Root.Level)
	}
	if got := LevelName(slog.Level(6)); got != "AUDIT_YAML" {
		t.Fatalf("LevelName(6) = %q, want AUDIT_YAML", got)
	}
}

func TestNewConfigured_whenConfigurationWrapperMixedWithTopLevel_shouldReject(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "goark-log.yml")
	WriteConfig(t, configPath, `
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
		t.Fatalf(
			"NewConfiguredHandler() should reject mixed configuration wrapper and top-level fields",
		)
	}
}

func writeMonitoredFileConfig(t *testing.T, configPath string, logPath string, level string) {
	t.Helper()
	WriteConfig(t, configPath, fmt.Sprintf(`
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

func writeFileConfig(t *testing.T, configPath string, logPath string, level string) {
	t.Helper()
	WriteConfig(t, configPath, fmt.Sprintf(`
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

func assertRootLevel(t *testing.T, options Options, level slog.Level) {
	t.Helper()
	if options.Root.Level != level {
		t.Fatalf("root level = %v, want %v", options.Root.Level, level)
	}
}
