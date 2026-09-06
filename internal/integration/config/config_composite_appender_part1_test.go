package integration

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	. "goark.dev/log/internal/testsupport"
)

func TestNewConfigured_whenCompositeAppenderHasFilter_shouldApplyBeforeDelegation(t *testing.T) {
	tests := []struct {
		name     string
		register func(*PluginRegistry, *testing.T)
		config   func(string) string
	}{
		{
			name: "failover",
			register: func(registry *PluginRegistry, t *testing.T) {
				t.Helper()
				factory := func(config AppenderBuildConfig) (Appender, error) {
					return NewFailingAppender(config.Name), nil
				}
				if err := registry.RegisterAppender("failing", factory); err != nil {
					t.Fatalf("RegisterAppender() error = %v", err)
				}
			},
			config: func(logPath string) string {
				return fmt.Sprintf(`
filters:
  deny-all:
    type: deny
appenders:
  primary:
    type: failing
  file:
    type: file
    fileName: %q
    layout:
      type: text
  guarded:
    type: failover
    primary: primary
    failovers: [file]
    filters: [deny-all]
root:
  level: info
  appenderRefs: [guarded]
`, filepath.ToSlash(logPath))
			},
		},
		{
			name: "routing",
			config: func(logPath string) string {
				return fmt.Sprintf(`
filters:
  deny-all:
    type: deny
appenders:
  file:
    type: file
    fileName: %q
    layout:
      type: text
  guarded:
    type: routing
    defaultRoute: file
    filters: [deny-all]
root:
  level: info
  appenderRefs: [guarded]
`, filepath.ToSlash(logPath))
			},
		},
		{
			name: "rewrite",
			config: func(logPath string) string {
				return fmt.Sprintf(`
filters:
  deny-all:
    type: deny
appenders:
  file:
    type: file
    fileName: %q
    layout:
      type: text
  guarded:
    type: rewrite
    appenderRefs: [file]
    filters: [deny-all]
    rewrite:
      attrs:
        tenant: core
root:
  level: info
  appenderRefs: [guarded]
`, filepath.ToSlash(logPath))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewPluginRegistry()
			if tt.register != nil {
				tt.register(registry, t)
			}
			dir := t.TempDir()
			logPath := filepath.Join(dir, tt.name+".log")
			configPath := filepath.Join(dir, "goark-log.yml")
			WriteConfig(t, configPath, tt.config(logPath))

			logger, handler, _, err := NewConfigured(
				context.Background(),
				WithConfigPath(configPath),
				WithPluginRegistry(registry),
			)
			if err != nil {
				t.Fatalf("NewConfigured() error = %v", err)
			}
			logger.Info("blocked composite event")
			if err := handler.Close(); err != nil {
				t.Fatalf("Close() error = %v", err)
			}
			if content := ReadTextFile(t, logPath); strings.Contains(
				content,
				"blocked composite event",
			) {
				t.Fatalf("composite appender filter was ignored, content = %q", content)
			}
		})
	}
}

func TestNewConfigured_whenYamlRewriteAppenderConfigured_shouldRewriteAttrs(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "rewrite.log")
	configPath := filepath.Join(dir, "goark-log.yml")
	WriteConfig(t, configPath, fmt.Sprintf(`
appenders:
  file:
    type: file
    fileName: %q
    layout:
      type: text
  rewrite:
    type: rewrite
    appenderRefs: [file]
    rewrite:
      attrs:
        tenant: core
      remove: [secret]
root:
  level: info
  appenderRefs: [rewrite]
`, filepath.ToSlash(logPath)))

	logger, handler, _, err := NewConfigured(context.Background(), WithConfigPath(configPath))
	if err != nil {
		t.Fatalf("NewConfigured() error = %v", err)
	}
	logger.Info("rewrite configured", slog.String("secret", "raw"))
	if err := handler.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	content := ReadTextFile(t, logPath)
	if !strings.Contains(content, "tenant=core") || strings.Contains(content, "secret=raw") {
		t.Fatalf("rewrite content = %q, want tenant added and secret removed", content)
	}
}

func TestLoadOptions_whenXMLAsyncAppenderBatchSizeConfigured_shouldApply(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "async.log")
	configPath := filepath.Join(dir, "goark-log.xml")
	WriteConfig(t, configPath, fmt.Sprintf(`
<Configuration>
  <Appenders>
    <File name="file" fileName="%s">
      <TextLayout/>
    </File>
    <Async name="async" queueSize="8" batchSize="3">
      <AppenderRef ref="file"/>
    </Async>
  </Appenders>
  <Loggers>
    <Root level="info">
      <AppenderRef ref="async"/>
    </Root>
  </Loggers>
</Configuration>
`, filepath.ToSlash(logPath)))

	options, _, err := LoadOptions(context.Background(), WithConfigPath(configPath))
	if err != nil {
		t.Fatalf("LoadOptions() error = %v", err)
	}
	defer CloseAppenderList(options.Appenders)
	asyncAppender := findAsyncAppender(options.Appenders, "async")
	if asyncAppender == nil {
		t.Fatalf("async appender was not built: %+v", options.Appenders)
	}
	if asyncAppender.BatchSize() != 3 {
		t.Fatalf("async appender batchSize = %d, want 3", asyncAppender.BatchSize())
	}
}

func findAsyncAppender(appenders []Appender, name string) *AsyncAppender {
	for _, appender := range appenders {
		if asyncAppender, ok := appender.(*AsyncAppender); ok && asyncAppender.Name() == name {
			return asyncAppender
		}
	}
	return nil
}
