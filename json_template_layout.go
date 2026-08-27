package goarklog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
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
	fields   []jsonTemplateField
	registry *PluginRegistry
	options  LayoutOptions
	state    *jsonLayoutState
}

// JSONTemplateLayoutOption 调整 JSONTemplateLayout 编译行为。
type JSONTemplateLayoutOption func(*jsonTemplateLayoutOptions)

type jsonTemplateLayoutOptions struct {
	registry      *PluginRegistry
	layoutOptions LayoutOptions
}

// JSONTemplateResolver 是 JSON Template 字段值编码器。
type JSONTemplateResolver interface {
	AppendJSON(buf *bytes.Buffer, event Event)
}

// JSONTemplateResolverFactory 从配置构建 JSON Template resolver。
type JSONTemplateResolverFactory func(config JSONTemplateResolverBuildConfig) (JSONTemplateResolver, error)

// JSONTemplateResolverBuildConfig 是 JSON Template resolver 插件的构建输入。
type JSONTemplateResolverBuildConfig struct {
	Name    string
	Options map[string]json.RawMessage
}

// WithJSONTemplateResolverRegistry 设置用于解析自定义 resolver 的插件注册表。
func WithJSONTemplateResolverRegistry(registry *PluginRegistry) JSONTemplateLayoutOption {
	return func(options *jsonTemplateLayoutOptions) {
		options.registry = registry
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
	rawFields, err := decodeJSONTemplateRawFields(template)
	if err != nil {
		return nil, fmt.Errorf("goark-log: parse JSON template layout: %w", err)
	}
	if len(rawFields) == 0 {
		return nil, fmt.Errorf("goark-log: JSON template layout requires at least one field")
	}
	fields := make([]jsonTemplateField, 0, len(rawFields))
	for _, rawField := range rawFields {
		resolver, err := compileJSONTemplateResolver(rawField.raw, settings.registry, settings.layoutOptions)
		if err != nil {
			return nil, fmt.Errorf("goark-log: JSON template field %q: %w", rawField.key, err)
		}
		fields = append(fields, jsonTemplateField{key: rawField.key, resolver: resolver})
	}
	layout := &JSONTemplateLayout{fields: fields, registry: settings.registry, options: settings.layoutOptions}
	if settings.layoutOptions.Complete {
		layout.state = &jsonLayoutState{}
	}
	return layout, nil
}

// NewJSONTemplateLayoutFromFile 从本地文件编译 JSON 事件模板。
func NewJSONTemplateLayoutFromFile(path string, options ...JSONTemplateLayoutOption) (*JSONTemplateLayout, error) {
	template, err := readJSONTemplateFile(path)
	if err != nil {
		return nil, err
	}
	return NewJSONTemplateLayout(template, options...)
}

func newJSONTemplateLayoutOptions(options ...JSONTemplateLayoutOption) jsonTemplateLayoutOptions {
	settings := jsonTemplateLayoutOptions{registry: DefaultPluginRegistry()}
	for _, option := range options {
		if option != nil {
			option(&settings)
		}
	}
	if settings.registry == nil {
		settings.registry = DefaultPluginRegistry()
	}
	return settings
}

func (l *JSONTemplateLayout) Format(buf *bytes.Buffer, event Event) error {
	if l == nil {
		return JSONLayout{}.Format(buf, event)
	}
	appendJSONCompleteSeparator(buf, l.options, l.state)
	buf.WriteByte('{')
	for index, field := range l.fields {
		appendJSONKey(buf, field.key, index > 0)
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
