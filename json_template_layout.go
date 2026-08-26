package goarklog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"goark.dev/log/internal/jsoncodec"
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

// JSONTemplateLayout 按 Log4j2 JSON Template 风格输出事件。
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

type jsonTemplateRawField struct {
	key string
	raw json.RawMessage
}

// NewJSONTemplateLayout 编译 JSON 事件模板。
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

func decodeJSONTemplateRawFields(template string) ([]jsonTemplateRawField, error) {
	decoder := json.NewDecoder(strings.NewReader(template))
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	if delimiter, ok := token.(json.Delim); !ok || delimiter != '{' {
		return nil, fmt.Errorf("event template must be a JSON object")
	}
	fields := make([]jsonTemplateRawField, 0, 8)
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		key, ok := token.(string)
		if !ok {
			return nil, fmt.Errorf("event template field key must be a string")
		}
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return nil, err
		}
		fields = append(fields, jsonTemplateRawField{key: key, raw: append([]byte(nil), raw...)})
	}
	token, err = decoder.Token()
	if err != nil {
		return nil, err
	}
	if delimiter, ok := token.(json.Delim); !ok || delimiter != '}' {
		return nil, fmt.Errorf("event template object is not closed")
	}
	if token, err = decoder.Token(); err != io.EOF {
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("event template has trailing token %v", token)
	}
	return fields, nil
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

func compileJSONTemplateResolver(raw json.RawMessage, registry *PluginRegistry, layoutOptions LayoutOptions) (JSONTemplateResolver, error) {
	var object map[string]json.RawMessage
	if err := jsoncodec.Unmarshal(raw, &object); err == nil {
		if resolverRaw, ok := object["$resolver"]; ok {
			var name string
			if err := jsoncodec.Unmarshal(resolverRaw, &name); err != nil {
				return nil, fmt.Errorf("$resolver must be a string")
			}
			return newJSONTemplateResolver(name, object, registry, layoutOptions)
		}
	}
	return rawJSONResolver{raw: append([]byte(nil), raw...)}, nil
}

func newJSONTemplateResolver(name string, options map[string]json.RawMessage, registry *PluginRegistry, layoutOptions LayoutOptions) (JSONTemplateResolver, error) {
	switch normalizeKind(name) {
	case "timestamp", "time":
		format := jsonTemplateStringOption(options, "format")
		layout, unix := normalizeTimePattern(format)
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
		if factory, ok := registry.jsonTemplateResolverFactory(name); ok {
			return factory(JSONTemplateResolverBuildConfig{Name: name, Options: copyJSONRawOptions(options)})
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

type rawJSONResolver struct {
	raw []byte
}

func (r rawJSONResolver) AppendJSON(buf *bytes.Buffer, _ Event) {
	if len(r.raw) == 0 {
		buf.WriteString("null")
		return
	}
	buf.Write(r.raw)
}

type timestampJSONResolver struct {
	layout string
	unix   timeUnixMode
}

func (r timestampJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	when := event.Time
	if when.IsZero() {
		when = time.Now()
	}
	switch r.unix {
	case timeUnixSeconds:
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), when.Unix(), 10))
	case timeUnixMillis:
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), when.UnixMilli(), 10))
	case timeUnixMicros:
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), when.UnixMicro(), 10))
	case timeUnixNanos:
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), when.UnixNano(), 10))
	default:
		buf.WriteByte('"')
		buf.Write(when.AppendFormat(buf.AvailableBuffer(), r.layout))
		buf.WriteByte('"')
	}
}

type levelJSONResolver struct {
	field string
}

func (r levelJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	switch normalizeKind(r.field) {
	case "int", "integer", "value":
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), int64(event.Level), 10))
	case "severity", "syslogseverity":
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), int64(syslogSeverity(event.Level)), 10))
	default:
		appendJSONString(buf, levelName(event.Level))
	}
}

type loggerJSONResolver struct {
	precision int
}

func (r loggerJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	appendJSONString(buf, loggerNameWithPrecision(event.Logger, r.precision))
}

type messageJSONResolver struct{}

func (messageJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	appendJSONString(buf, event.Message)
}

type threadJSONResolver struct{}

func (threadJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	appendJSONString(buf, eventThreadName(event))
}

type markerJSONResolver struct{}

func (markerJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	if event.Marker == nil {
		buf.WriteString("null")
		return
	}
	appendJSONString(buf, event.Marker.String())
}

type throwableJSONResolver struct {
	field              string
	stacktraceAsString bool
}

func (r throwableJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	throwable := event.Throwable
	if throwable == nil {
		buf.WriteString("null")
		return
	}
	switch normalizeKind(r.field) {
	case "", "object":
		if r.stacktraceAsString {
			appendJSONString(buf, throwableStackString(throwable))
			return
		}
		appendThrowableJSON(buf, throwable)
	case "type":
		appendJSONString(buf, throwable.Type)
	case "message":
		appendJSONString(buf, throwable.Message)
	case "string", "formatted":
		if r.stacktraceAsString {
			appendJSONString(buf, throwableStackString(throwable))
			return
		}
		appendJSONString(buf, throwable.String())
	case "rootcause":
		appendThrowableJSON(buf, rootThrowable(throwable))
	case "rootcausemessage":
		appendJSONString(buf, rootThrowable(throwable).Message)
	case "stacktrace":
		appendThrowableStackJSON(buf, throwable)
	case "stacktraceasstring", "stacktracestring":
		appendJSONString(buf, throwableStackString(throwable))
	default:
		appendThrowableJSON(buf, throwable)
	}
}

type sourceJSONResolver struct{}

func (sourceJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	frame := callerFrameFromPC(event.PC)
	if frame.method == "" && frame.file == "" && frame.line == 0 {
		buf.WriteString("null")
		return
	}
	buf.WriteByte('{')
	appendJSONFieldString(buf, "class", frame.class, false)
	appendJSONFieldString(buf, "method", frame.method, true)
	appendJSONFieldString(buf, "file", frame.file, true)
	appendJSONKey(buf, "line", true)
	buf.Write(strconv.AppendInt(buf.AvailableBuffer(), int64(frame.line), 10))
	appendJSONFieldString(buf, "location", frame.location(), true)
	buf.WriteByte('}')
}

type processJSONResolver struct{}

func (processJSONResolver) AppendJSON(buf *bytes.Buffer, _ Event) {
	buf.WriteByte('{')
	appendJSONKey(buf, "pid", false)
	buf.WriteString(processIDString)
	buf.WriteByte('}')
}

type contextStackJSONResolver struct{}

func (contextStackJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	buf.WriteByte('[')
	for index, value := range event.ContextStack {
		if index > 0 {
			buf.WriteByte(',')
		}
		appendJSONString(buf, value)
	}
	buf.WriteByte(']')
}

type attrsJSONResolver struct {
	flatten          bool
	propertiesAsList bool
}

func (r attrsJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	attrs := event.Attrs
	if r.flatten {
		attrs = make([]slog.Attr, 0, len(event.Attrs))
		for _, attr := range event.Attrs {
			appendFlattenedJSONAttr(&attrs, "", attr)
		}
	}
	if r.propertiesAsList {
		appendJSONAttrsList(buf, attrs)
		return
	}
	appendJSONAttrsObject(buf, attrs)
}

type attrJSONResolver struct {
	key string
}

func (r attrJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	value, ok := event.Attr(r.key)
	if !ok {
		buf.WriteString("null")
		return
	}
	appendJSONValue(buf, value)
}

type endOfBatchJSONResolver struct{}

func (endOfBatchJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	if event.EndOfBatch {
		buf.WriteString("true")
		return
	}
	buf.WriteString("false")
}

func appendJSONAttrsObject(buf *bytes.Buffer, attrs []slog.Attr) {
	buf.WriteByte('{')
	for index, attr := range attrs {
		appendJSONFieldValue(buf, attr.Key, attr.Value, index > 0)
	}
	buf.WriteByte('}')
}

func appendThrowableJSON(buf *bytes.Buffer, throwable *Throwable) {
	if throwable == nil {
		buf.WriteString("null")
		return
	}
	buf.WriteByte('{')
	appendJSONFieldString(buf, "type", throwable.Type, false)
	appendJSONFieldString(buf, "message", throwable.Message, true)
	appendJSONKey(buf, "rootCause", true)
	appendThrowableRootCauseJSON(buf, rootThrowable(throwable))
	appendJSONKey(buf, "stackTrace", true)
	appendThrowableStackJSON(buf, throwable)
	if throwable.Cause != nil {
		appendJSONKey(buf, "cause", true)
		appendThrowableJSON(buf, throwable.Cause)
	}
	buf.WriteByte('}')
}

func appendThrowableRootCauseJSON(buf *bytes.Buffer, throwable *Throwable) {
	if throwable == nil {
		buf.WriteString("null")
		return
	}
	buf.WriteByte('{')
	appendJSONFieldString(buf, "type", throwable.Type, false)
	appendJSONFieldString(buf, "message", throwable.Message, true)
	buf.WriteByte('}')
}

func appendThrowableStackJSON(buf *bytes.Buffer, throwable *Throwable) {
	if throwable == nil || len(throwable.Stack) == 0 {
		buf.WriteString("null")
		return
	}
	buf.WriteByte('[')
	for index, frame := range throwable.Stack {
		if index > 0 {
			buf.WriteByte(',')
		}
		appendJSONString(buf, frame)
	}
	buf.WriteByte(']')
}

func rootThrowable(throwable *Throwable) *Throwable {
	if throwable == nil {
		return nil
	}
	for throwable.Cause != nil {
		throwable = throwable.Cause
	}
	return throwable
}

func appendFlattenedJSONAttr(attrs *[]slog.Attr, prefix string, attr slog.Attr) {
	attr = normalizeAttr(attr)
	if attr.Key == "" {
		return
	}
	key := attr.Key
	if prefix != "" {
		key = prefix + "." + key
	}
	if attr.Value.Kind() != slog.KindGroup {
		*attrs = append(*attrs, slog.Attr{Key: key, Value: attr.Value})
		return
	}
	for _, child := range attr.Value.Group() {
		appendFlattenedJSONAttr(attrs, key, child)
	}
}

func readJSONTemplateFile(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("goark-log: JSON template file path is empty")
	}
	resolved, err := localTemplatePath(path)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return "", fmt.Errorf("goark-log: read JSON template file %q: %w", resolved, err)
	}
	return string(data), nil
}

func localTemplatePath(value string) (string, error) {
	if runtime.GOOS == "windows" && len(value) >= 2 && value[1] == ':' {
		return value, nil
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" {
		return value, nil
	}
	if parsed.Scheme != "file" {
		return "", fmt.Errorf("goark-log: JSON template URI scheme %q is not allowed in core", parsed.Scheme)
	}
	path := parsed.Path
	if parsed.Host != "" {
		path = "//" + parsed.Host + parsed.Path
	}
	if runtime.GOOS == "windows" && len(path) >= 3 && path[0] == '/' && path[2] == ':' {
		path = path[1:]
	}
	return path, nil
}
