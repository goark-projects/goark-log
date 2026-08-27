package layout

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"goark.dev/log/internal/jsoncodec"
	"goark.dev/log/internal/jsontemplate"
	"goark.dev/log/internal/logvalue"
	"goark.dev/log/internal/textutil"
	"goark.dev/log/internal/timepattern"
)

const defaultJSONEventTemplate = `{
  "timestamp": {"$resolver": "timestamp"},
  "level": {"$resolver": "level"},
  "loggerName": {"$resolver": "logger"},
  "message": {"$resolver": "message"},
  "thread": {"$resolver": "thread"},
  "marker": {"$resolver": "marker"},
  "thrown": {"$resolver": "throwable"},
  "contextStack": {"$resolver": "contextStack"},
  "endOfBatch": {"$resolver": "endOfBatch"},
  "contextMap": {"$resolver": "mdc"}
}`

// JSONTemplateLayout 按 JSON 事件模板输出日志事件。
type JSONTemplateLayout struct {
	fields  []jsonTemplateField
	options LayoutOptions
	state   *jsonLayoutState
}

// JSONTemplateLayoutOption 调整 JSONTemplateLayout 编译行为。
type JSONTemplateLayoutOption func(*jsonTemplateLayoutOptions)

type jsonTemplateLayoutOptions struct {
	resolverLookup JSONTemplateResolverLookup
	layoutOptions  LayoutOptions
}

// JSONTemplateResolver 是 JSON Template 字段值编码器。
type JSONTemplateResolver interface {
	AppendJSON(buf *bytes.Buffer, event Event)
}

// JSONTemplateResolverFactory 从配置构建 JSON Template resolver。
type JSONTemplateResolverFactory func(config JSONTemplateResolverBuildConfig) (JSONTemplateResolver, error)

// JSONTemplateResolverLookup 按规范化前的名称查找自定义 resolver 工厂。
type JSONTemplateResolverLookup func(kind string) (JSONTemplateResolverFactory, bool)

// JSONTemplateResolverBuildConfig 是 JSON Template resolver 插件的构建输入。
type JSONTemplateResolverBuildConfig struct {
	Name    string
	Options map[string]json.RawMessage
}

// WithJSONTemplateResolverLookup 设置用于解析自定义 resolver 的查找函数。
func WithJSONTemplateResolverLookup(lookup JSONTemplateResolverLookup) JSONTemplateLayoutOption {
	return func(options *jsonTemplateLayoutOptions) {
		options.resolverLookup = lookup
	}
}

// WithJSONTemplateLayoutOptions 设置 JSON Template 布局的通用输出参数。
func WithJSONTemplateLayoutOptions(layoutOptions LayoutOptions) JSONTemplateLayoutOption {
	return func(options *jsonTemplateLayoutOptions) {
		options.layoutOptions = layoutOptions
	}
}

type jsonTemplateField struct {
	key      string
	resolver JSONTemplateResolver
}

func NewJSONTemplateLayout(template string, options ...JSONTemplateLayoutOption) (*JSONTemplateLayout, error) {
	settings := newJSONTemplateLayoutOptions(options...)
	if strings.TrimSpace(template) == "" {
		template = defaultJSONEventTemplate
	}
	rawFields, err := jsontemplate.DecodeRawFields(template)
	if err != nil {
		return nil, fmt.Errorf("goark-log: parse JSON template layout: %w", err)
	}
	if len(rawFields) == 0 {
		return nil, fmt.Errorf("goark-log: JSON template layout requires at least one field")
	}
	fields := make([]jsonTemplateField, 0, len(rawFields))
	for _, rawField := range rawFields {
		resolver, err := compileJSONTemplateResolver(rawField.Raw, settings.resolverLookup, settings.layoutOptions)
		if err != nil {
			return nil, fmt.Errorf("goark-log: JSON template field %q: %w", rawField.Key, err)
		}
		fields = append(fields, jsonTemplateField{key: rawField.Key, resolver: resolver})
	}
	layout := &JSONTemplateLayout{fields: fields, options: settings.layoutOptions}
	if settings.layoutOptions.Complete {
		layout.state = &jsonLayoutState{}
	}
	return layout, nil
}

// NewJSONTemplateLayoutFromFile 从本地文件编译 JSON 事件模板。
func NewJSONTemplateLayoutFromFile(path string, options ...JSONTemplateLayoutOption) (*JSONTemplateLayout, error) {
	template, err := jsontemplate.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return NewJSONTemplateLayout(template, options...)
}

func newJSONTemplateLayoutOptions(options ...JSONTemplateLayoutOption) jsonTemplateLayoutOptions {
	settings := jsonTemplateLayoutOptions{}
	for _, option := range options {
		if option != nil {
			option(&settings)
		}
	}
	return settings
}

func compileJSONTemplateResolver(raw json.RawMessage, lookup JSONTemplateResolverLookup, layoutOptions LayoutOptions) (JSONTemplateResolver, error) {
	var object map[string]json.RawMessage
	if err := jsoncodec.Unmarshal(raw, &object); err == nil {
		if resolverRaw, ok := object["$resolver"]; ok {
			var name string
			if err := jsoncodec.Unmarshal(resolverRaw, &name); err != nil {
				return nil, fmt.Errorf("$resolver must be a string")
			}
			return newJSONTemplateResolver(name, object, lookup, layoutOptions)
		}
	}
	return rawJSONResolver{raw: append([]byte(nil), raw...)}, nil
}

func newJSONTemplateResolver(name string, options map[string]json.RawMessage, lookup JSONTemplateResolverLookup, layoutOptions LayoutOptions) (JSONTemplateResolver, error) {
	switch textutil.NormalizeKind(name) {
	case "timestamp", "time":
		format := jsonTemplateStringOption(options, "format")
		layout, unix := timepattern.Normalize(format)
		return timestampJSONResolver{layout: layout, unix: unix}, nil
	case "level":
		return levelJSONResolver{field: jsonTemplateStringOption(options, "field")}, nil
	case "logger", "loggername":
		return loggerJSONResolver{precision: jsonTemplateIntOption(options, "precision")}, nil
	case "message", "msg":
		return messageJSONResolver{}, nil
	case "thread":
		return threadJSONResolver{}, nil
	case "threadname":
		return threadJSONResolver{}, nil
	case "marker":
		return markerJSONResolver{}, nil
	case "throwable", "exception", "thrown":
		return throwableJSONResolver{
			field:              jsonTemplateStringOption(options, "field"),
			stacktraceAsString: layoutOptions.StacktraceAsString,
		}, nil
	case "rootcause":
		return throwableJSONResolver{field: "rootCause"}, nil
	case "stacktrace":
		field := "stackTrace"
		if layoutOptions.StacktraceAsString {
			field = "stackTraceAsString"
		}
		return throwableJSONResolver{field: field}, nil
	case "source", "location":
		return sourceJSONResolver{}, nil
	case "process":
		return processJSONResolver{}, nil
	case "contextstack", "ndc":
		return contextStackJSONResolver{}, nil
	case "mdc", "contextmap", "attrs":
		return attrsJSONResolver{
			flatten:          jsonTemplateBoolOption(options, "flatten"),
			propertiesAsList: layoutOptions.PropertiesAsList || jsonTemplateBoolOption(options, "propertiesAsList"),
		}, nil
	case "attr":
		key := jsonTemplateStringOption(options, "key")
		if strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("attr resolver requires key")
		}
		return attrJSONResolver{key: key}, nil
	case "endofbatch":
		return endOfBatchJSONResolver{}, nil
	default:
		if lookup != nil {
			if factory, ok := lookup(name); ok {
				return factory(JSONTemplateResolverBuildConfig{Name: name, Options: copyJSONRawOptions(options)})
			}
		}
		return nil, fmt.Errorf("unsupported resolver %q", name)
	}
}

func jsonTemplateStringOption(options map[string]json.RawMessage, key string) string {
	raw, ok := options[key]
	if !ok {
		return ""
	}
	var value string
	if err := jsoncodec.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return strings.TrimSpace(value)
}

func jsonTemplateBoolOption(options map[string]json.RawMessage, key string) bool {
	raw, ok := options[key]
	if !ok {
		return false
	}
	var value bool
	return jsoncodec.Unmarshal(raw, &value) == nil && value
}

func jsonTemplateIntOption(options map[string]json.RawMessage, key string) int {
	raw, ok := options[key]
	if !ok {
		return 0
	}
	var value int
	if err := jsoncodec.Unmarshal(raw, &value); err == nil {
		return value
	}
	var text string
	if err := jsoncodec.Unmarshal(raw, &text); err != nil {
		return 0
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil || parsed < 0 {
		return 0
	}
	return parsed
}

func copyJSONRawOptions(options map[string]json.RawMessage) map[string]json.RawMessage {
	copied := make(map[string]json.RawMessage, len(options))
	for key, raw := range options {
		copied[key] = append([]byte(nil), raw...)
	}
	return copied
}

func (l *JSONTemplateLayout) Format(buf *bytes.Buffer, event Event) error {
	if l == nil {
		return JSONLayout{}.Format(buf, event)
	}
	appendJSONCompleteSeparator(buf, l.options, l.state)
	buf.WriteByte('{')
	for index, field := range l.fields {
		logvalue.AppendJSONKey(buf, field.key, index > 0)
		field.resolver.AppendJSON(buf, event)
	}
	buf.WriteByte('}')
	appendLayoutTerminator(buf, l.options)
	return nil
}

func (l *JSONTemplateLayout) AppendHeader(buf *bytes.Buffer) error {
	if l == nil {
		return nil
	}
	appendJSONCompleteHeader(buf, l.options, l.state)
	return nil
}

func (l *JSONTemplateLayout) AppendFooter(buf *bytes.Buffer) error {
	if l == nil {
		return nil
	}
	appendJSONCompleteFooter(buf, l.options)
	return nil
}
