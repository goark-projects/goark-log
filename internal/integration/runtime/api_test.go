package integration

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"goark.dev/log"
)

func TestPublicAPICompile(t *testing.T) {
	var out bytes.Buffer
	handler, err := log.NewHandler(log.Options{
		Appenders: []log.Appender{
			log.NewConsoleAppender(
				log.WithConsoleWriter(&out),
				log.WithConsoleLayout(log.TextLayout{}),
			),
		},
		Root: log.RootLogger{
			Level:        slog.LevelInfo,
			AppenderRefs: []string{"console"},
		},
		Loggers: []log.LoggerRule{
			{
				Name:  "goark.orm",
				Level: levelPtr(slog.LevelDebug),
			},
		},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	defer handler.Close()

	logger := log.NewLogger(handler, "goark.orm")
	logger.Debug("sql prepared")
	logger.Info("sql done")

	type newConfiguredFunc func(
		context.Context,
		...log.ConfigLoadOption,
	) (*slog.Logger, *log.Handler, *log.ConfigResult, error)
	type newHandlerFunc func(
		context.Context,
		...log.ConfigLoadOption,
	) (*log.Handler, *log.ConfigResult, error)
	type loadOptionsFunc func(
		context.Context,
		...log.ConfigLoadOption,
	) (log.Options, *log.ConfigResult, error)
	var _ newConfiguredFunc = log.NewConfigured
	var _ newHandlerFunc = log.NewConfiguredHandler
	var _ newHandlerFunc = log.ConfigureDefault
	var _ loadOptionsFunc = log.LoadOptions
	var _ log.ConfigLoadOption = log.WithConfigPath("conf/goark-log.yml")
	var _ log.ConfigLoadOption = log.WithPluginRegistry(log.NewPluginRegistry())
	var _ log.LoggerContextOption = log.WithLoggerContextStatus(log.NewStatusLogger())
	var _ *log.PluginRegistry = log.DefaultPluginRegistry()
}

func TestPublicAPI_whenPatternLayoutOptionsUsed_shouldCompileAndRender(t *testing.T) {
	layout, err := log.NewPatternLayoutWithOptions(
		"%style{%m}{red}%n",
		log.LayoutOptions{DisableANSI: true},
	)
	if err != nil {
		t.Fatalf("NewPatternLayoutWithOptions() error = %v", err)
	}
	var _ log.Layout = layout

	var out bytes.Buffer
	event := log.Event{Level: slog.LevelInfo, Logger: "goark.api", Message: "plain"}
	if err := layout.Format(&out, event); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	if out.String() != "plain\n" {
		t.Fatalf("pattern output = %q, want plain text", out.String())
	}
}

func TestPublicAPI_whenPluginBuildConfigUsed_shouldExposeStableFields(t *testing.T) {
	registry := log.NewPluginRegistry()
	var capturedLayout log.LayoutBuildConfig
	layoutFactory := func(config log.LayoutBuildConfig) (log.Layout, error) {
		capturedLayout = config
		return log.TextLayout{}, nil
	}
	if err := registry.RegisterLayout("apiCapture", layoutFactory); err != nil {
		t.Fatalf("RegisterLayout() error = %v", err)
	}

	var capturedAppender log.AppenderBuildConfig
	appenderFactory := func(config log.AppenderBuildConfig) (log.Appender, error) {
		capturedAppender = config
		return log.NewConsoleAppender(
			log.WithConsoleName(config.Name),
			log.WithConsoleWriter(io.Discard),
		), nil
	}
	if err := registry.RegisterAppender("apiCapture", appenderFactory); err != nil {
		t.Fatalf("RegisterAppender() error = %v", err)
	}

	configPath := filepath.Join(t.TempDir(), "goark-log.yml")
	content := []byte(`
appenders:
  custom:
    type: apiCapture
    target: stdout
    url: https://logs.example.invalid
    method: POST
    address: 127.0.0.1:1514
    network: tcp
    facility: LOCAL0
    appName: api-test
    connectTimeout: 100ms
    writeTimeout: 200ms
    fileName: logs/app.log
    bufferSize: 64KiB
    flushOnWrite: true
    append: false
    createOnDemand: true
    filePermissions: "0640"
    layout:
      type: apiCapture
      pattern: "%m%n"
      disableAnsi: true
    rolling:
      filePattern: logs/archive/app-%i.log.gz
      policies:
        time:
          interval: daily
          modulate: true
        startup:
          enabled: true
      strategy:
        max: 3
        maxAge: 7d
        fileIndex: max
        directWrite: true
        compression:
          gzip: true
          async: true
        delete:
          basePath: logs/archive
          maxDepth: 1
          ifFileName:
            glob: "*.gz"
          ifLastModified:
            age: 7d
    rewrite:
      attrs:
        service: api
      remove: [password]
root:
  level: info
  appenderRefs: [custom]
`)
	if err := os.WriteFile(configPath, content, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	handler, _, err := log.NewConfiguredHandler(context.Background(),
		log.WithConfigPath(configPath),
		log.WithPluginRegistry(registry),
	)
	if err != nil {
		t.Fatalf("NewConfiguredHandler() error = %v", err)
	}
	defer handler.Close()

	if capturedLayout.Pattern != "%m%n" || !capturedLayout.Options.DisableANSI {
		t.Fatalf("layout build config = %+v, want pattern and DisableANSI", capturedLayout)
	}
	if capturedAppender.Name != "custom" ||
		capturedAppender.Target != "stdout" ||
		capturedAppender.URL != "https://logs.example.invalid" ||
		capturedAppender.Method != "POST" ||
		capturedAppender.Address != "127.0.0.1:1514" ||
		capturedAppender.Network != "tcp" ||
		capturedAppender.Facility != "LOCAL0" ||
		capturedAppender.AppName != "api-test" ||
		capturedAppender.ConnectTimeout != "100ms" ||
		capturedAppender.WriteTimeout != "200ms" ||
		capturedAppender.FileName != "logs/app.log" ||
		capturedAppender.BufferSize != "64KiB" ||
		!capturedAppender.FlushOnWrite ||
		capturedAppender.Append == nil ||
		*capturedAppender.Append ||
		!capturedAppender.CreateOnDemand ||
		capturedAppender.FilePermissions != "0640" {
		t.Fatalf("appender build config = %+v, want stable scalar fields", capturedAppender)
	}
	if capturedAppender.Rolling.FilePattern != "logs/archive/app-%i.log.gz" ||
		capturedAppender.Rolling.Interval != "daily" ||
		capturedAppender.Rolling.TimeModulate == nil ||
		!*capturedAppender.Rolling.TimeModulate ||
		!capturedAppender.Rolling.OnStartup ||
		capturedAppender.Rolling.MaxBackups == nil ||
		*capturedAppender.Rolling.MaxBackups != 3 ||
		capturedAppender.Rolling.MaxAge != "7d" ||
		capturedAppender.Rolling.FileIndex != "max" ||
		!capturedAppender.Rolling.DirectWrite ||
		!capturedAppender.Rolling.Gzip ||
		!capturedAppender.Rolling.AsyncActions ||
		len(capturedAppender.Rolling.DeleteActions) != 1 {
		t.Fatalf("rolling build config = %+v, want stable rolling fields", capturedAppender.Rolling)
	}
	deleteAction := capturedAppender.Rolling.DeleteActions[0]
	if deleteAction.BasePath != "logs/archive" ||
		deleteAction.MaxDepth != 1 ||
		deleteAction.Glob != "*.gz" ||
		deleteAction.MaxAge != "7d" {
		t.Fatalf(
			"rolling delete build config = %+v, want stable delete action fields",
			deleteAction,
		)
	}
	if capturedAppender.Rewrite.Attrs["service"] != "api" ||
		len(capturedAppender.Rewrite.RemoveAttrs) != 1 ||
		capturedAppender.Rewrite.RemoveAttrs[0] != "password" {
		t.Fatalf(
			"rewrite build config = %+v, want attrs and removal keys",
			capturedAppender.Rewrite,
		)
	}
}

func levelPtr(level slog.Level) *slog.Level {
	return &level
}
