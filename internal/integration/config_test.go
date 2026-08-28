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

func TestLoadOptions_whenAsyncWaitOptionsConfigured_shouldPopulateRuntimeOptions(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "goark-log.yml")
	writeConfig(t, configPath, `
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
	defer closeAppenderList(options.Appenders)
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
		t.Fatalf("async appender options = batch %d strategy %s options %+v", asyncAppender.BatchSize(), asyncAppender.WaitStrategy(), waitOptions)
	}
}

func TestLoadOptions_whenStructuredLayoutOptionsConfigured_shouldPopulateJSONLayout(t *testing.T) {
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
      type: json
      compact: true
      eventEol: true
      complete: true
      includeStacktrace: true
      stacktraceAsString: true
      propertiesAsList: true
      includeNullDelimiter: true
      header: H
      footer: F
root:
  level: info
  appenderRefs: [file]
`, filepath.ToSlash(logPath))
			},
		},
		{
			name:       "json",
			configName: "goark-log.json",
			content: func(logPath string) string {
				return fmt.Sprintf(`{
  "appenders": {
    "file": {
      "type": "file",
      "fileName": %q,
      "layout": {
        "type": "json",
        "compact": true,
        "eventEol": true,
        "complete": true,
        "includeStacktrace": true,
        "stacktraceAsString": true,
        "propertiesAsList": true,
        "includeNullDelimiter": true,
        "header": "H",
        "footer": "F"
      }
    }
  },
  "root": {"level": "info", "appenderRefs": ["file"]}
}`, filepath.ToSlash(logPath))
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
type = "json"
compact = true
eventEol = true
complete = true
includeStacktrace = true
stacktraceAsString = true
propertiesAsList = true
includeNullDelimiter = true
header = "H"
footer = "F"

[root]
level = "info"
appenderRefs = ["file"]
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
      <JSONLayout compact="true" eventEol="true" complete="true" includeStacktrace="true" stacktraceAsString="true" propertiesAsList="true" includeNullDelimiter="true" header="H" footer="F"/>
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
			name:       "properties",
			configName: "goark-log.properties",
			content: func(logPath string) string {
				return fmt.Sprintf(`
appender.file.type = file
appender.file.fileName = %s
appender.file.layout.type = json
appender.file.layout.compact = true
appender.file.layout.eventEol = true
appender.file.layout.complete = true
appender.file.layout.includeStacktrace = true
appender.file.layout.stacktraceAsString = true
appender.file.layout.propertiesAsList = true
appender.file.layout.includeNullDelimiter = true
appender.file.layout.header = H
appender.file.layout.footer = F
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
			writeConfig(t, configPath, tc.content(logPath))

			options, _, err := LoadOptions(ctx, WithConfigPath(configPath))
			if err != nil {
				t.Fatalf("LoadOptions() error = %v", err)
			}
			defer closeAppenderList(options.Appenders)
			layout := configuredFileJSONLayout(t, options.Appenders, "file")
			assertFullLayoutOptions(t, layout.Options())
		})
	}
}

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
			writeConfig(t, configPath, tc.content(logPath))

			options, _, err := LoadOptions(ctx, WithConfigPath(configPath))
			if err != nil {
				t.Fatalf("LoadOptions() error = %v", err)
			}
			defer closeAppenderList(options.Appenders)
			layout := configuredFilePatternLayout(t, options.Appenders, "file")
			if !layout.Options().DisableANSI {
				t.Fatalf("pattern layout options = %+v, want DisableANSI", layout.Options())
			}
		})
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

func TestNewConfigured_whenJsonTemplateLayoutConfigured_shouldWriteTemplateJson(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "logs", "template.log")
	configPath := filepath.Join(dir, "goark-log.yml")
	writeConfig(t, configPath, fmt.Sprintf(`
appenders:
  file:
    type: file
    fileName: %q
    flushOnWrite: true
    layout:
      type: jsonTemplate
      eventTemplate: '{"message":{"$resolver":"message"},"trace":{"$resolver":"attr","key":"trace_id"}}'
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
	content := readTextFile(t, logPath)
	if !strings.Contains(content, `"message":"json template config"`) ||
		!strings.Contains(content, `"trace":"trace-config"`) {
		t.Fatalf("json template layout output is wrong: %q", content)
	}
}

func TestNewConfigured_whenJsonAppenderFileConfigured_shouldUseDirectJSONWriter(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "logs", "direct.json")
	configPath := filepath.Join(dir, "goark-log.yml")
	writeConfig(t, configPath, fmt.Sprintf(`
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
	content := readTextFile(t, logPath)
	if !strings.Contains(content, `"msg":"direct json config"`) ||
		!strings.Contains(content, `"profile":"bench"`) {
		t.Fatalf("direct JSON config output is wrong: %q", content)
	}
}

func TestLoadOptions_whenYamlCustomLevelsConfigured_shouldRegisterBeforeLevelParsing(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "goark-log.yml")
	writeConfig(t, configPath, `
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

func TestLoadOptions_whenPropertiesCustomLevelConfigured_shouldRegister(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "goark-log.properties")
	writeConfig(t, configPath, `
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

func TestLoadOptions_whenXmlCustomLevelConfigured_shouldRegister(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "goark-log.xml")
	writeConfig(t, configPath, `
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

func TestNewConfigured_whenRollingCronScheduleInvalid_shouldReject(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "goark-log.yml")
	writeConfig(t, configPath, `
appenders:
  rolling:
    type: rollingFile
    fileName: app.log
    rolling:
      policies:
        cron:
          schedule: "0 0 24 * * *"
root:
  level: info
  appenderRefs: [rolling]
`)
	_, _, err := NewConfiguredHandler(context.Background(), WithConfigPath(configPath))
	if err == nil || !strings.Contains(err.Error(), "cron schedule") {
		t.Fatalf("NewConfiguredHandler() error = %v, want cron schedule rejection", err)
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

func TestLoadOptions_whenTomlConfigProvided_shouldBuildOptions(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "goark-log.toml")
	logPath := filepath.Join(dir, "app.log")
	writeConfig(t, configPath, `
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
	defer closeAppenderList(options.Appenders)
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
		t.Fatalf("Logger additivity = set:%v value:%v, want set false", logger.AdditivitySet, logger.Additivity)
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

func TestLoadOptions_whenDefaultTomlExists_shouldLoad(t *testing.T) {
	dir := t.TempDir()
	confDir := filepath.Join(dir, "conf")
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(confDir, "goark-log.toml"), []byte("root.level = \"info\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	options, result, err := LoadOptions(context.Background(), WithConfigWorkingDir(dir))
	if err != nil {
		t.Fatalf("LoadOptions() error = %v", err)
	}
	defer closeAppenderList(options.Appenders)
	if result.Source != ConfigSourceFile {
		t.Fatalf("ConfigResult.Source = %q, want file", result.Source)
	}
	if got := options.Root.Level; got != slog.LevelInfo {
		t.Fatalf("Root.Level = %v, want info", got)
	}
}

func TestLoadOptions_whenTomlHasUnknownField_shouldReject(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "goark-log.toml")
	writeConfig(t, configPath, `
[root]
level = "info"
unknown = true
`)
	_, _, err := LoadOptions(context.Background(), WithConfigPath(configPath))
	if err == nil {
		t.Fatalf("LoadOptions() should reject unknown TOML fields")
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
