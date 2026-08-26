package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	goarklog "goark.dev/log"
)

func main() {
	registry := goarklog.NewPluginRegistry()
	if err := registry.RegisterPlugins(goarklog.NewPluginSet(
		goarklog.WithPluginJSONTemplateResolver("constant", buildConstantResolver),
	)); err != nil {
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

func buildConstantResolver(config goarklog.JSONTemplateResolverBuildConfig) (goarklog.JSONTemplateResolver, error) {
	var value string
	if err := json.Unmarshal(config.Options["value"], &value); err != nil {
		return nil, fmt.Errorf("constant resolver value is invalid: %w", err)
	}
	return constantResolver(value), nil
}

type constantResolver string

func (r constantResolver) AppendJSON(buf *bytes.Buffer, _ goarklog.Event) {
	data, err := json.Marshal(string(r))
	if err != nil {
		buf.WriteString("null")
		return
	}
	buf.Write(data)
}
