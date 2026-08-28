package layout

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"

	"goark.dev/log/internal/layoutsupport"
	"goark.dev/log/internal/logvalue"
	"gopkg.in/yaml.v3"
)

// JSONLayout 输出 JSON 事件。
type JSONLayout struct {
	options LayoutOptions
	state   *jsonLayoutState
}

// NewJSONLayout 创建可配置 JSON 布局。
func NewJSONLayout(options LayoutOptions) JSONLayout {
	layout := JSONLayout{options: options}
	if options.Complete {
		layout.state = &jsonLayoutState{}
	}
	return layout
}

func (l JSONLayout) Format(buf *bytes.Buffer, event Event) error {
	appendJSONCompleteSeparator(buf, l.options, l.state)
	appendJSONLayoutEvent(buf, event, l.options)
	return nil
}

func (l JSONLayout) AppendHeader(buf *bytes.Buffer) error {
	appendJSONCompleteHeader(buf, l.options, l.state)
	return nil
}

func (l JSONLayout) AppendFooter(buf *bytes.Buffer) error {
	appendJSONCompleteFooter(buf, l.options)
	return nil
}

// CloneLayout 为每个 appender 隔离 Complete 模式的事件计数。
func (l JSONLayout) CloneLayout() Layout {
	cloned := l
	if cloned.options.Complete {
		cloned.state = &jsonLayoutState{}
	} else {
		cloned.state = nil
	}
	return cloned
}

// RequiresSynchronizedFormatting 说明 Complete JSON 流需要按实际写入顺序生成分隔符。
func (l JSONLayout) RequiresSynchronizedFormatting() bool {
	return l.options.Complete
}

// Options 返回布局输出参数快照。
func (l JSONLayout) Options() LayoutOptions {
	return l.options
}

type jsonLayoutState struct {
	events atomic.Uint64
}

func appendJSONCompleteSeparator(buf *bytes.Buffer, options LayoutOptions, state *jsonLayoutState) {
	if !options.Complete || state == nil || state.events.Add(1) <= 1 {
		return
	}
	buf.WriteByte(',')
	if options.EventEOL || !options.Compact {
		buf.WriteByte('\n')
	}
}

func appendJSONCompleteHeader(buf *bytes.Buffer, options LayoutOptions, state *jsonLayoutState) {
	if state != nil {
		state.events.Store(0)
	}
	if !options.Complete {
		return
	}
	header := options.Header
	if strings.TrimSpace(header) == "" {
		header = "["
	}
	buf.WriteString(header)
	if options.EventEOL || !options.Compact {
		buf.WriteByte('\n')
	}
}

func appendJSONCompleteFooter(buf *bytes.Buffer, options LayoutOptions) {
	if !options.Complete {
		return
	}
	footer := options.Footer
	if strings.TrimSpace(footer) == "" {
		footer = "]"
	}
	buf.WriteString(footer)
}

// AppendJSONEvent 用零额外状态编码 JSON 单行事件。
func AppendJSONEvent(buf *bytes.Buffer, when time.Time, level slog.Level, logger string, message string, attrs []slog.Attr) {
	buf.WriteByte('{')
	logvalue.AppendJSONFieldTime(buf, "time", when, defaultTimeFormat, false)
	logvalue.AppendJSONFieldString(buf, "level", levelName(level), true)
	logvalue.AppendJSONFieldString(buf, "logger", logger, true)
	logvalue.AppendJSONFieldString(buf, "msg", message, true)
	for _, attr := range attrs {
		logvalue.AppendJSONFieldValue(buf, attr.Key, attr.Value, true)
	}
	buf.WriteString("}\n")
}

func appendJSONLayoutEvent(buf *bytes.Buffer, event Event, options LayoutOptions) {
	buf.WriteByte('{')
	logvalue.AppendJSONFieldTime(buf, "time", event.Time, defaultTimeFormat, false)
	logvalue.AppendJSONFieldString(buf, "level", levelName(event.Level), true)
	logvalue.AppendJSONFieldString(buf, "logger", event.Logger, true)
	logvalue.AppendJSONFieldString(buf, "msg", event.Message, true)
	if options.PropertiesAsList {
		logvalue.AppendJSONAttrsListField(buf, "contextMap", event.Attrs, true)
	} else {
		for _, attr := range event.Attrs {
			logvalue.AppendJSONFieldValue(buf, attr.Key, attr.Value, true)
		}
	}
	if event.Throwable != nil && (options.IncludeStacktrace || options.StacktraceAsString) {
		logvalue.AppendJSONKey(buf, "thrown", true)
		if options.StacktraceAsString {
			logvalue.AppendJSONString(buf, throwableStackString(event.Throwable))
		} else {
			appendThrowableJSON(buf, event.Throwable)
		}
	}
	buf.WriteByte('}')
	appendLayoutTerminator(buf, options)
}

// AppendJSONFixedEvent 编码最多三个属性的固定数组事件，避免热路径切片分配。
func AppendJSONFixedEvent(buf *bytes.Buffer, when time.Time, level slog.Level, logger string, message string, attrs [3]slog.Attr, count int) {
	buf.WriteByte('{')
	logvalue.AppendJSONFieldTime(buf, "time", when, defaultTimeFormat, false)
	logvalue.AppendJSONFieldString(buf, "level", levelName(level), true)
	logvalue.AppendJSONFieldString(buf, "logger", logger, true)
	logvalue.AppendJSONFieldString(buf, "msg", message, true)
	for index := 0; index < count && index < len(attrs); index++ {
		attr := attrs[index]
		logvalue.AppendJSONFieldValue(buf, attr.Key, attr.Value, true)
	}
	buf.WriteString("}\n")
}

// XMLLayout 输出单事件 XML 片段。
type XMLLayout struct {
	options LayoutOptions
}

// NewXMLLayout 创建可配置 XML 布局。
func NewXMLLayout(options LayoutOptions) XMLLayout {
	return XMLLayout{options: options}
}

// Format 把事件编码为 XML。
func (l XMLLayout) Format(buf *bytes.Buffer, event Event) error {
	buf.WriteString("<Event")
	appendXMLAttr(buf, "time", layoutsupport.EventTime(event.Time).Format(defaultTimeFormat))
	appendXMLAttr(buf, "level", levelName(event.Level))
	appendXMLAttr(buf, "logger", event.Logger)
	appendXMLAttr(buf, "thread", eventThreadName(event))
	if event.EndOfBatch {
		appendXMLAttr(buf, "endOfBatch", "true")
	}
	buf.WriteByte('>')
	appendXMLElement(buf, "Message", event.Message)
	if marker := eventMarkerString(event); marker != "" {
		appendXMLElement(buf, "Marker", marker)
	}
	appendXMLThrowable(buf, l.options, event)
	if len(event.ContextStack) > 0 {
		buf.WriteString("<ContextStack>")
		for _, value := range event.ContextStack {
			appendXMLElement(buf, "Value", value)
		}
		buf.WriteString("</ContextStack>")
	}
	if len(event.Attrs) > 0 {
		buf.WriteString("<ContextMap>")
		for _, attr := range event.Attrs {
			buf.WriteString("<Entry")
			appendXMLAttr(buf, "key", attr.Key)
			buf.WriteByte('>')
			appendXMLText(buf, logvalue.String(attr.Value))
			buf.WriteString("</Entry>")
		}
		buf.WriteString("</ContextMap>")
	}
	buf.WriteString("</Event>")
	appendLayoutTerminator(buf, l.options)
	return nil
}

func (l XMLLayout) AppendHeader(buf *bytes.Buffer) error {
	appendLayoutHeader(buf, l.options)
	return nil
}

func (l XMLLayout) AppendFooter(buf *bytes.Buffer) error {
	appendLayoutFooter(buf, l.options)
	return nil
}

func appendXMLThrowable(buf *bytes.Buffer, options LayoutOptions, event Event) {
	if event.Throwable == nil {
		if thrown := eventErrorString(event); thrown != "" {
			appendXMLElement(buf, "Throwable", thrown)
		}
		return
	}
	if options.StacktraceAsString {
		appendXMLElement(buf, "Throwable", throwableStackString(event.Throwable))
		return
	}
	appendXMLElement(buf, "Throwable", event.Throwable.String())
	if !options.IncludeStacktrace || len(event.Throwable.Stack) == 0 {
		return
	}
	buf.WriteString("<StackTrace>")
	for _, frame := range event.Throwable.Stack {
		appendXMLElement(buf, "Frame", frame)
	}
	buf.WriteString("</StackTrace>")
}

func appendXMLElement(buf *bytes.Buffer, name string, value string) {
	buf.WriteByte('<')
	buf.WriteString(name)
	buf.WriteByte('>')
	appendXMLText(buf, value)
	buf.WriteString("</")
	buf.WriteString(name)
	buf.WriteByte('>')
}

func appendXMLAttr(buf *bytes.Buffer, key string, value string) {
	buf.WriteByte(' ')
	buf.WriteString(key)
	buf.WriteString("=\"")
	appendXMLText(buf, value)
	buf.WriteByte('"')
}

func appendXMLText(buf *bytes.Buffer, value string) {
	_ = xml.EscapeText(buf, []byte(value))
}

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
