package goarklog

import (
	"bytes"
	"fmt"
	"log/slog"
	"time"

	"goark.dev/log/internal/layoutsupport"
	"goark.dev/log/internal/logvalue"
	"gopkg.in/yaml.v3"
)

// YAMLLayout 输出单事件 YAML 文档。
type YAMLLayout struct {
	options LayoutOptions
}

// NewYAMLLayout 创建可配置 YAML 布局。
func NewYAMLLayout(options LayoutOptions) YAMLLayout {
	return YAMLLayout{options: options}
}

// Format 把事件编码为 YAML。
func (l YAMLLayout) Format(buf *bytes.Buffer, event Event) error {
	fields := map[string]any{
		"time":    layoutsupport.EventTime(event.Time).Format(defaultTimeFormat),
		"level":   levelName(event.Level),
		"logger":  event.Logger,
		"thread":  eventThreadName(event),
		"message": event.Message,
	}
	if marker := eventMarkerString(event); marker != "" {
		fields["marker"] = marker
	}
	if throwable := yamlThrowableValue(event, l.options); throwable != nil {
		fields["throwable"] = throwable
	}
	if len(event.ContextStack) > 0 {
		fields["contextStack"] = append([]string(nil), event.ContextStack...)
	}
	if len(event.Attrs) > 0 {
		fields["contextMap"] = yamlContextMapValue(event.Attrs, l.options)
	}
	data, err := yaml.Marshal(fields)
	if err != nil {
		return fmt.Errorf("goark-log: format YAML layout: %w", err)
	}
	if l.options.Compact {
		data = bytes.TrimRight(data, "\n")
	}
	buf.Write(data)
	if l.options.Compact || l.options.IncludeNullDelimiter {
		appendLayoutTerminator(buf, l.options)
	}
	return nil
}

func (l YAMLLayout) AppendHeader(buf *bytes.Buffer) error {
	appendLayoutHeader(buf, l.options)
	return nil
}

func (l YAMLLayout) AppendFooter(buf *bytes.Buffer) error {
	appendLayoutFooter(buf, l.options)
	return nil
}

func yamlThrowableValue(event Event, options LayoutOptions) any {
	if event.Throwable == nil {
		if thrown := eventErrorString(event); thrown != "" {
			return thrown
		}
		return nil
	}
	if options.StacktraceAsString {
		return throwableStackString(event.Throwable)
	}
	if options.IncludeStacktrace {
		return throwableMapValue(event.Throwable)
	}
	return event.Throwable.String()
}

func throwableMapValue(throwable *Throwable) map[string]any {
	if throwable == nil {
		return nil
	}
	value := map[string]any{
		"type":      throwable.Type,
		"message":   throwable.Message,
		"rootCause": throwableRootMapValue(rootThrowable(throwable)),
	}
	if len(throwable.Stack) > 0 {
		value["stackTrace"] = append([]string(nil), throwable.Stack...)
	}
	if throwable.Cause != nil {
		value["cause"] = throwableMapValue(throwable.Cause)
	}
	return value
}

func throwableRootMapValue(throwable *Throwable) map[string]any {
	if throwable == nil {
		return nil
	}
	return map[string]any{
		"type":    throwable.Type,
		"message": throwable.Message,
	}
}

func yamlContextMapValue(attrs []slog.Attr, options LayoutOptions) any {
	if options.PropertiesAsList {
		values := make([]map[string]any, 0, len(attrs))
		for _, attr := range attrs {
			values = append(values, map[string]any{
				"key":   attr.Key,
				"value": slogValueAny(attr.Value),
			})
		}
		return values
	}
	values := make(map[string]any, len(attrs))
	for _, attr := range attrs {
		values[attr.Key] = slogValueAny(attr.Value)
	}
	return values
}

func slogValueAny(value slog.Value) any {
	value = value.Resolve()
	switch value.Kind() {
	case slog.KindString:
		return value.String()
	case slog.KindBool:
		return value.Bool()
	case slog.KindInt64:
		return value.Int64()
	case slog.KindUint64:
		return value.Uint64()
	case slog.KindFloat64:
		return value.Float64()
	case slog.KindDuration:
		return value.Duration().String()
	case slog.KindTime:
		return value.Time().Format(time.RFC3339Nano)
	case slog.KindGroup:
		group := make(map[string]any, len(value.Group()))
		for _, attr := range value.Group() {
			group[attr.Key] = slogValueAny(attr.Value)
		}
		return group
	case slog.KindAny:
		return value.Any()
	default:
		return logvalue.String(value)
	}
}
