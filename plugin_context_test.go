package goarklog

import (
	"bytes"
	"context"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestPluginRegistry_whenCustomPluginsConfigured_shouldBuildThroughYaml(t *testing.T) {
	var captured *pluginCaptureAppender
	registry := NewPluginRegistry()
	if err := registry.RegisterLookup("test", func(key string) (string, bool) {
		if key == "prefix" {
			return "PLUGIN", true
		}
		return "", false
	}); err != nil {
		t.Fatalf("RegisterLookup() error = %v", err)
	}
	if err := registry.RegisterLayout("prefix", func(config LayoutBuildConfig) (Layout, error) {
		return prefixLayout{prefix: config.Pattern}, nil
	}); err != nil {
		t.Fatalf("RegisterLayout() error = %v", err)
	}
	if err := registry.RegisterFilter("denyContains", func(config FilterBuildConfig) (Filter, error) {
		return FilterFunc(func(_ context.Context, event Event) FilterDecision {
			if strings.Contains(event.Message, config.Pattern) {
				return FilterDeny
			}
			return FilterNeutral
		}), nil
	}); err != nil {
		t.Fatalf("RegisterFilter() error = %v", err)
	}
	if err := registry.RegisterAppender("capture", func(config AppenderBuildConfig) (Appender, error) {
		captured = &pluginCaptureAppender{name: config.Name, layout: config.Layout}
		return captured, nil
	}); err != nil {
		t.Fatalf("RegisterAppender() error = %v", err)
	}

	configPath := filepath.Join(t.TempDir(), "goark-log.yml")
	writeConfig(t, configPath, `
filters:
  drop:
    type: denyContains
    pattern: "drop"
appenders:
  capture:
    type: capture
    layout:
      type: prefix
      pattern: "${test:prefix}"
    filters: [drop]
root:
  level: info
  appenderRefs: [capture]
`)
	logger, handler, _, err := NewConfigured(context.Background(), WithConfigPath(configPath), WithPluginRegistry(registry))
	if err != nil {
		t.Fatalf("NewConfigured() error = %v", err)
	}
	logger.Info("drop this")
	logger.Info("keep this")
	if err := handler.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if captured == nil {
		t.Fatalf("custom appender was not built")
	}
	if captured.String() != "PLUGIN:keep this\n" {
		t.Fatalf("captured output = %q", captured.String())
	}
}

func TestLoggerContext_whenReloadFails_shouldReportStatus(t *testing.T) {
	var out bytes.Buffer
	status := NewStatusLogger(
		WithStatusLevel(slog.LevelDebug),
		WithStatusWriter(&out),
		WithStatusBufferSize(8),
	)
	context, err := NewLoggerContext(DefaultOptions(), WithLoggerContextStatus(status))
	if err != nil {
		t.Fatalf("NewLoggerContext() error = %v", err)
	}
	err = context.Reload(Options{
		Appenders: []Appender{NewConsoleAppender()},
		Root:      RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"missing"}},
	})
	if err == nil {
		t.Fatalf("Reload() should reject missing appender")
	}
	events := status.Events()
	if len(events) == 0 || events[len(events)-1].Level != slog.LevelError {
		t.Fatalf("status events should contain reload error, got %+v", events)
	}
	if !strings.Contains(out.String(), "reload logger context failed") {
		t.Fatalf("status output should contain reload failure, got %q", out.String())
	}
	if err := context.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

type prefixLayout struct {
	prefix string
}

func (l prefixLayout) Format(buf *bytes.Buffer, event Event) error {
	buf.WriteString(l.prefix)
	buf.WriteByte(':')
	buf.WriteString(event.Message)
	buf.WriteByte('\n')
	return nil
}

type pluginCaptureAppender struct {
	name   string
	layout Layout
	mu     sync.Mutex
	out    bytes.Buffer
}

func (a *pluginCaptureAppender) Name() string {
	return a.name
}

func (a *pluginCaptureAppender) Append(ctx context.Context, event Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := a.layout.Format(&buf, event); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	_, err := a.out.Write(buf.Bytes())
	return err
}

func (a *pluginCaptureAppender) Close() error {
	return nil
}

func (a *pluginCaptureAppender) String() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.out.String()
}
