package main

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/bytedance/sonic"

	"goark.dev/log"
)

func main() {
	registry := log.NewPluginRegistry()
	plugins := log.NewPluginSet(
		log.WithPluginLookup("tenant", tenantLookup),
		log.WithPluginJSONTemplateResolver("constant", buildConstantResolver),
	)
	if err := registry.RegisterPlugins(plugins); err != nil {
		panic(err)
	}

	layout, err := log.NewJSONTemplateLayout(`{
  "timestamp": {"$resolver": "timestamp", "format": "RFC3339NANO"},
  "level": {"$resolver": "level"},
  "logger": {"$resolver": "logger"},
  "component": {"$resolver": "constant", "value": "extensibility"},
  "message": {"$resolver": "message"},
  "contextMap": {"$resolver": "mdc"}
}`, log.WithJSONTemplateResolverRegistry(registry))
	if err != nil {
		panic(err)
	}

	handler, err := log.NewHandler(log.Options{
		Appenders: []log.Appender{
			log.NewConsoleAppender(
				log.WithConsoleWriter(os.Stdout),
				log.WithConsoleLayout(layout),
			),
		},
		Root: log.RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"console"}},
	})
	if err != nil {
		panic(err)
	}
	defer handler.Close()

	logger, err := log.NewNativeLogger(handler, "goark.example",
		log.WithLoggerMessageFactory(log.SimpleMessageFactory{}),
	)
	if err != nil {
		panic(err)
	}
	ctx := log.WithContextAttrs(context.Background(), slog.String("tenant", "tenant-a"))
	_ = logger.AtInfo().WithContext(ctx).Logf("literal message from custom message factory")
}

func tenantLookup(key string) (string, bool) {
	if key == "default" {
		return "tenant-a", true
	}
	return "", false
}

func buildConstantResolver(
	config log.JSONTemplateResolverBuildConfig,
) (log.JSONTemplateResolver, error) {
	var value string
	if err := sonic.Unmarshal(config.Options["value"], &value); err != nil {
		return nil, fmt.Errorf("constant resolver value is invalid: %w", err)
	}
	return constantResolver(value), nil
}

type constantResolver string

func (r constantResolver) AppendJSON(buf *bytes.Buffer, _ log.Event) {
	data, err := sonic.Marshal(string(r))
	if err != nil {
		buf.WriteString("null")
		return
	}
	buf.Write(data)
}
