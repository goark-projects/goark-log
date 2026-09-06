package integration

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	. "goark.dev/log/internal/testsupport"
)

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
      <JSONLayout compact="true" eventEol="true" complete="true"
          includeStacktrace="true" stacktraceAsString="true"
          propertiesAsList="true" includeNullDelimiter="true"
          header="H" footer="F"/>
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
			WriteConfig(t, configPath, tc.content(logPath))

			options, _, err := LoadOptions(ctx, WithConfigPath(configPath))
			if err != nil {
				t.Fatalf("LoadOptions() error = %v", err)
			}
			defer CloseAppenderList(options.Appenders)
			layout := configuredFileJSONLayout(t, options.Appenders, "file")
			assertFullLayoutOptions(t, layout.Options())
		})
	}
}

func TestNewConfigured_whenXmlConfigurationUsed_shouldBuild(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "logs", "xml.log")
	configPath := filepath.Join(dir, "goark-log.xml")
	WriteConfig(t, configPath, fmt.Sprintf(`
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
	if content := ReadTextFile(t, logPath); !strings.Contains(content, "xml config") {
		t.Fatalf("xml config output is wrong: %q", content)
	}
}

func TestNewConfigured_whenAppenderFilterRefMissing_shouldCloseBuiltAppender(t *testing.T) {
	registry := NewPluginRegistry()
	var built *RecordingAppender
	factory := func(config AppenderBuildConfig) (Appender, error) {
		built = NewRecordingAppender(config.Name)
		return built, nil
	}
	if err := registry.RegisterAppender("recording", factory); err != nil {
		t.Fatalf("RegisterAppender() error = %v", err)
	}
	configPath := filepath.Join(t.TempDir(), "goark-log.yml")
	WriteConfig(t, configPath, `
appenders:
  recording:
    type: recording
    filters: [missing]
root:
  level: info
  appenderRefs: [recording]
`)

	_, _, err := NewConfiguredHandler(
		context.Background(),
		WithConfigPath(configPath),
		WithPluginRegistry(registry),
	)
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

func TestNewConfigured_whenRollingCronScheduleInvalid_shouldReject(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "goark-log.yml")
	WriteConfig(t, configPath, `
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

func TestNewConfigured_whenFilterRefBlank_shouldReject(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "goark-log.yml")
	WriteConfig(t, configPath, `
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

func TestNewConfigured_whenAppenderRefMissing_shouldReject(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "goark-log.yml")
	WriteConfig(t, configPath, `
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

func TestLoadOptions_whenTomlHasUnknownField_shouldReject(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "goark-log.toml")
	WriteConfig(t, configPath, `
[root]
level = "info"
unknown = true
`)
	_, _, err := LoadOptions(context.Background(), WithConfigPath(configPath))
	if err == nil {
		t.Fatalf("LoadOptions() should reject unknown TOML fields")
	}
}
