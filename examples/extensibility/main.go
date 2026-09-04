package main

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/bytedance/sonic"

	goarklog "goark.dev/log"
)

func main() {
	registry := goarklog.NewPluginRegistry()
	plugins := goarklog.NewPluginSet(
		goarklog.WithPluginLookup("tenant", tenantLookup),
		goarklog.WithPluginJSONTemplateResolver("constant", buildConstantResolver),
	)
	if err := registry.RegisterPlugins(plugins); err != nil {
		panic(err)
	}

	layout, err := goarklog.NewJSONTemplateLayout(`{
  "timestamp": {"$resolver": "timestamp", "format": "RFC3339NANO"},
  "level": {"$resolver": "level"},
  "logger": {"$resolver": "logger"},
  "component": {"$resolver": "constant", "value": "extensibility"},
  "message": {"$resolver": "message"},
  "contextMap": {"$resolver": "mdc"}
}`, goarklog.WithJSONTemplateResolverRegistry(registry))
	if err != nil {
		panic(err)
	}

	handler, err := goarklog.NewHandler(goarklog.Options{
		Appenders: []goarklog.Appender{
			goarklog.NewConsoleAppender(
				goarklog.WithConsoleWriter(os.Stdout),
				goarklog.WithConsoleLayout(layout),
			),
		},
		Root: goarklog.RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"console"}},
	})
	if err != nil {
		panic(err)
	}
	defer handler.Close()

	logger, err := goarklog.NewNativeLogger(handler, "goark.example",
		goarklog.WithLoggerMessageFactory(goarklog.SimpleMessageFactory{}),
	)
	if err != nil {
		panic(err)
	}
	ctx := goarklog.WithContextAttrs(context.Background(), slog.String("tenant", "tenant-a"))
	_ = logger.AtInfo().WithContext(ctx).Logf("literal message from custom message factory")
}

func tenantLookup(key string) (string, bool) {
	if key == "default" {
		return "tenant-a", true
	}
	return "", false
}

func buildConstantResolver(config goarklog.JSONTemplateResolverBuildConfig) (goarklog.JSONTemplateResolver, error) {
	var value string
	if err := sonic.Unmarshal(config.Options["value"], &value); err != nil {
		return nil, fmt.Errorf("constant resolver value is invalid: %w", err)
	}
	return constantResolver(value), nil
}

type constantResolver string

func (r constantResolver) AppendJSON(buf *bytes.Buffer, _ goarklog.Event) {
	data, err := sonic.Marshal(string(r))
	if err != nil {
		buf.WriteString("null")
		return
	}
	buf.Write(data)
}
