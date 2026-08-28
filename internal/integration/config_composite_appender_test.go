package integration

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewConfigured_whenYamlFailoverAppenderConfigured_shouldUseFailover(t *testing.T) {
	registry := NewPluginRegistry()
	if err := registry.RegisterAppender("failing", func(config AppenderBuildConfig) (Appender, error) {
		return failingAppender{name: config.Name}, nil
	}); err != nil {
		t.Fatalf("RegisterAppender() error = %v", err)
	}
	dir := t.TempDir()
	backupPath := filepath.Join(dir, "backup.log")
	configPath := filepath.Join(dir, "goark-log.yml")
	writeConfig(t, configPath, fmt.Sprintf(`
appenders:
  primary:
    type: failing
  backup:
    type: file
    fileName: %q
    layout:
      type: text
  safe:
    type: failover
    primary: primary
    failovers: [backup]
root:
  level: info
  appenderRefs: [safe]
`, filepath.ToSlash(backupPath)))

	logger, handler, _, err := NewConfigured(context.Background(), WithConfigPath(configPath), WithPluginRegistry(registry))
	if err != nil {
		t.Fatalf("NewConfigured() error = %v", err)
	}
	logger.Info("failover configured")
	if err := handler.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if content := readTextFile(t, backupPath); !strings.Contains(content, "failover configured") {
		t.Fatalf("backup content = %q, want failover event", content)
	}
}

func TestNewConfigured_whenYamlRoutingAppenderConfigured_shouldRouteByAttr(t *testing.T) {
	dir := t.TempDir()
	auditPath := filepath.Join(dir, "audit.log")
	defaultPath := filepath.Join(dir, "default.log")
	configPath := filepath.Join(dir, "goark-log.yml")
	writeConfig(t, configPath, fmt.Sprintf(`
appenders:
  audit:
    type: file
    fileName: %q
    layout:
      type: text
  general:
    type: file
    fileName: %q
    layout:
      type: text
  router:
    type: routing
    routeKey: kind
    defaultRoute: general
    routes:
      audit: audit
root:
  level: info
  appenderRefs: [router]
`, filepath.ToSlash(auditPath), filepath.ToSlash(defaultPath)))

	logger, handler, _, err := NewConfigured(context.Background(), WithConfigPath(configPath))
	if err != nil {
		t.Fatalf("NewConfigured() error = %v", err)
	}
	logger.Info("audit configured", slog.String("kind", "audit"))
	logger.Info("default configured", slog.String("kind", "business"))
	if err := handler.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	auditContent := readTextFile(t, auditPath)
	defaultContent := readTextFile(t, defaultPath)
	if !strings.Contains(auditContent, "audit configured") || strings.Contains(auditContent, "default configured") {
		t.Fatalf("audit content = %q, want only audit event", auditContent)
	}
	if !strings.Contains(defaultContent, "default configured") || strings.Contains(defaultContent, "audit configured") {
		t.Fatalf("default content = %q, want only default event", defaultContent)
	}
}

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
				if err := registry.RegisterAppender("failing", func(config AppenderBuildConfig) (Appender, error) {
					return failingAppender{name: config.Name}, nil
				}); err != nil {
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
			writeConfig(t, configPath, tt.config(logPath))

			logger, handler, _, err := NewConfigured(context.Background(), WithConfigPath(configPath), WithPluginRegistry(registry))
			if err != nil {
				t.Fatalf("NewConfigured() error = %v", err)
			}
			logger.Info("blocked composite event")
			if err := handler.Close(); err != nil {
				t.Fatalf("Close() error = %v", err)
			}
			if content := readTextFile(t, logPath); strings.Contains(content, "blocked composite event") {
				t.Fatalf("composite appender filter was ignored, content = %q", content)
			}
		})
	}
}

func TestNewConfigured_whenYamlRewriteAppenderConfigured_shouldRewriteAttrs(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "rewrite.log")
	configPath := filepath.Join(dir, "goark-log.yml")
	writeConfig(t, configPath, fmt.Sprintf(`
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

	content := readTextFile(t, logPath)
	if !strings.Contains(content, "tenant=core") || strings.Contains(content, "secret=raw") {
		t.Fatalf("rewrite content = %q, want tenant added and secret removed", content)
	}
}

func TestNewConfigured_whenPropertiesCompositeAppendersConfigured_shouldBuild(t *testing.T) {
	dir := t.TempDir()
	auditPath := filepath.Join(dir, "audit.log")
	defaultPath := filepath.Join(dir, "default.log")
	configPath := filepath.Join(dir, "goark-log.properties")
	writeConfig(t, configPath, fmt.Sprintf(`
appender.audit.type=file
appender.audit.fileName=%s
appender.audit.layout.type=text
appender.general.type=file
appender.general.fileName=%s
appender.general.layout.type=text
appender.router.type=routing
appender.router.routeKey=kind
appender.router.defaultRoute=general
appender.router.routes.audit=audit
rootLogger.level=info
rootLogger.appenderRefs=router
`, filepath.ToSlash(auditPath), filepath.ToSlash(defaultPath)))

	logger, handler, _, err := NewConfigured(context.Background(), WithConfigPath(configPath))
	if err != nil {
		t.Fatalf("NewConfigured() error = %v", err)
	}
	logger.Info("properties audit", slog.String("kind", "audit"))
	logger.Info("properties default")
	if err := handler.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if content := readTextFile(t, auditPath); !strings.Contains(content, "properties audit") || strings.Contains(content, "properties default") {
		t.Fatalf("properties audit content = %q, want only audit event", content)
	}
	if content := readTextFile(t, defaultPath); !strings.Contains(content, "properties default") || strings.Contains(content, "properties audit") {
		t.Fatalf("properties default content = %q, want only default event", content)
	}
}

func TestLoadOptions_whenPropertiesAsyncAppenderBatchSizeConfigured_shouldApply(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "async.log")
	configPath := filepath.Join(dir, "goark-log.properties")
	writeConfig(t, configPath, fmt.Sprintf(`
appender.file.type=file
appender.file.fileName=%s
appender.file.layout.type=text
appender.async.type=async
appender.async.appenderRefs=file
appender.async.queueSize=8
appender.async.batchSize=3
rootLogger.level=info
rootLogger.appenderRefs=async
`, filepath.ToSlash(logPath)))

	options, _, err := LoadOptions(context.Background(), WithConfigPath(configPath))
	if err != nil {
		t.Fatalf("LoadOptions() error = %v", err)
	}
	defer closeAppenderList(options.Appenders)
	asyncAppender := findAsyncAppender(options.Appenders, "async")
	if asyncAppender == nil {
		t.Fatalf("async appender was not built: %+v", options.Appenders)
	}
	if asyncAppender.BatchSize() != 3 {
		t.Fatalf("async appender batchSize = %d, want 3", asyncAppender.BatchSize())
	}
}

func TestNewConfigured_whenXmlCompositeAppendersConfigured_shouldBuild(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "rewrite.log")
	configPath := filepath.Join(dir, "goark-log.xml")
	writeConfig(t, configPath, fmt.Sprintf(`
<Configuration>
  <Appenders>
    <File name="file" fileName="%s">
      <TextLayout/>
    </File>
    <Rewrite name="rewrite">
      <AppenderRef ref="file"/>
      <KeyValuePair key="tenant" value="xml"/>
      <Remove key="secret"/>
    </Rewrite>
  </Appenders>
  <Loggers>
    <Root level="info">
      <AppenderRef ref="rewrite"/>
    </Root>
  </Loggers>
</Configuration>
`, filepath.ToSlash(logPath)))

	logger, handler, _, err := NewConfigured(context.Background(), WithConfigPath(configPath))
	if err != nil {
		t.Fatalf("NewConfigured() error = %v", err)
	}
	logger.Info("xml rewrite", slog.String("secret", "raw"))
	if err := handler.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	content := readTextFile(t, logPath)
	if !strings.Contains(content, "tenant=xml") || strings.Contains(content, "secret=raw") {
		t.Fatalf("xml rewrite content = %q, want tenant added and secret removed", content)
	}
}

func TestLoadOptions_whenXMLAsyncAppenderBatchSizeConfigured_shouldApply(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "async.log")
	configPath := filepath.Join(dir, "goark-log.xml")
	writeConfig(t, configPath, fmt.Sprintf(`
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
	defer closeAppenderList(options.Appenders)
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
