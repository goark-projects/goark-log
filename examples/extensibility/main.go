package main

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	goarklog "goark.dev/log"
)

func main() {
	registry := goarklog.NewPluginRegistry()
	if err := registry.RegisterPlugins(exampleRegistrar{}); err != nil {
		panic(err)
	}

	layout, err := goarklog.NewJSONTemplateLayout(`{
  "message": {"$resolver": "message"},
  "component": {"$resolver": "constant", "value": "extensibility"},
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
	_ = logger.AtInfo().
		WithContext(goarklog.WithContextAttrs(context.Background(), slog.String("trace_id", "trace-1"))).
		Logf("factory keeps this literal")
}

type exampleRegistrar struct{}

func (exampleRegistrar) RegisterLogPlugins(registry *goarklog.PluginRegistry) error {
	return registry.RegisterJSONTemplateResolver("constant", func(config goarklog.JSONTemplateResolverBuildConfig) (goarklog.JSONTemplateResolver, error) {
		raw := config.Options["value"]
		value, err := strconv.Unquote(string(raw))
		if err != nil {
			return nil, fmt.Errorf("constant resolver value is invalid: %w", err)
		}
		return constantResolver(value), nil
	})
}

type constantResolver string

func (r constantResolver) AppendJSON(buf *bytes.Buffer, _ goarklog.Event) {
	buf.WriteString(strconv.Quote(string(r)))
}
