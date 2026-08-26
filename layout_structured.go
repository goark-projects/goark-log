package goarklog

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"html"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var hostNameString = resolveHostName()

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
	appendXMLAttr(buf, "time", eventTime(event.Time).Format(defaultTimeFormat))
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
			appendXMLText(buf, attrValueString(attr.Value))
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

// CSVLayout 输出单行 CSV，字段顺序固定。
type CSVLayout struct {
	options LayoutOptions
}

// NewCSVLayout 创建可配置 CSV 布局。
func NewCSVLayout(options LayoutOptions) CSVLayout {
	return CSVLayout{options: options}
}

// Format 把事件编码为 CSV。
func (l CSVLayout) Format(buf *bytes.Buffer, event Event) error {
	appendCSVField(buf, eventTime(event.Time).Format(defaultTimeFormat), false)
	appendCSVField(buf, levelName(event.Level), true)
	appendCSVField(buf, event.Logger, true)
	appendCSVField(buf, eventThreadName(event), true)
	appendCSVField(buf, event.Message, true)
	if len(event.Attrs) == 0 {
		appendLayoutTerminator(buf, l.options)
		return nil
	}
	var attrs bytes.Buffer
	appendPatternAttrs(&attrs, event.Attrs)
	appendCSVField(buf, attrs.String(), true)
	appendLayoutTerminator(buf, l.options)
	return nil
}

func (l CSVLayout) AppendHeader(buf *bytes.Buffer) error {
	appendLayoutHeader(buf, l.options)
	return nil
}

func (l CSVLayout) AppendFooter(buf *bytes.Buffer) error {
	appendLayoutFooter(buf, l.options)
	return nil
}

// GELFLayout 输出 Graylog Extended Log Format 单行 JSON。
type GELFLayout struct {
	options LayoutOptions
}

// NewGELFLayout 创建可配置 GELF 布局。
func NewGELFLayout(options LayoutOptions) GELFLayout {
	return GELFLayout{options: options}
}

// Format 把事件编码为 GELF JSON。
func (l GELFLayout) Format(buf *bytes.Buffer, event Event) error {
	when := eventTime(event.Time)
	buf.WriteByte('{')
	appendJSONFieldString(buf, "version", "1.1", false)
	appendJSONFieldString(buf, "host", hostNameString, true)
	appendJSONFieldString(buf, "short_message", event.Message, true)
	if thrown := gelfThrowableString(event, l.options); thrown != "" {
		appendJSONFieldString(buf, "full_message", thrown, true)
	}
	appendJSONKey(buf, "timestamp", true)
	buf.Write(strconv.AppendFloat(buf.AvailableBuffer(), float64(when.UnixNano())/1e9, 'f', 6, 64))
	appendJSONKey(buf, "level", true)
	buf.Write(strconv.AppendInt(buf.AvailableBuffer(), int64(syslogSeverity(event.Level)), 10))
	appendJSONFieldString(buf, "_logger", event.Logger, true)
	appendJSONFieldString(buf, "_thread", eventThreadName(event), true)
	if marker := eventMarkerString(event); marker != "" {
		appendJSONFieldString(buf, "_marker", marker, true)
	}
	for _, attr := range event.Attrs {
		key := gelfAdditionalFieldKey(attr.Key)
		if key == "" {
			continue
		}
		appendJSONFieldValue(buf, key, attr.Value, true)
	}
	buf.WriteByte('}')
	appendLayoutTerminator(buf, l.options)
	return nil
}

func (l GELFLayout) AppendHeader(buf *bytes.Buffer) error {
	appendLayoutHeader(buf, l.options)
	return nil
}

func (l GELFLayout) AppendFooter(buf *bytes.Buffer) error {
	appendLayoutFooter(buf, l.options)
	return nil
}

// RFC5424Layout 输出 RFC 5424 syslog 单行事件。
type RFC5424Layout struct {
	Facility  int
	AppName   string
	MessageID string
}

// SyslogLayout 是 RFC5424Layout 的语义别名。
type SyslogLayout = RFC5424Layout

// Format 把事件编码为 RFC 5424 syslog。
func (l RFC5424Layout) Format(buf *bytes.Buffer, event Event) error {
	priority := syslogPriority(l.Facility, event.Level)
	buf.WriteByte('<')
	buf.Write(strconv.AppendInt(buf.AvailableBuffer(), int64(priority), 10))
	buf.WriteString(">1 ")
	buf.WriteString(eventTime(event.Time).UTC().Format(time.RFC3339Nano))
	buf.WriteByte(' ')
	appendSyslogToken(buf, hostNameString)
	buf.WriteByte(' ')
	appendSyslogToken(buf, firstNonBlank(l.AppName, event.Logger, "goark"))
	buf.WriteByte(' ')
	appendSyslogToken(buf, processIDString)
	buf.WriteByte(' ')
	appendSyslogToken(buf, firstNonBlank(l.MessageID, "-"))
	buf.WriteByte(' ')
	appendStructuredData(buf, event)
	buf.WriteByte(' ')
	buf.WriteString(event.Message)
	buf.WriteByte('\n')
	return nil
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
		"time":    eventTime(event.Time).Format(defaultTimeFormat),
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

// HTMLLayout 输出 HTML 表格行，适合文件或控制台片段组合。
type HTMLLayout struct {
	options LayoutOptions
}

// NewHTMLLayout 创建可配置 HTML 布局。
func NewHTMLLayout(options LayoutOptions) HTMLLayout {
	return HTMLLayout{options: options}
}

// Format 把事件编码为 HTML 表格行。
func (l HTMLLayout) Format(buf *bytes.Buffer, event Event) error {
	buf.WriteString("<tr>")
	appendHTMLCell(buf, eventTime(event.Time).Format(defaultTimeFormat))
	appendHTMLCell(buf, levelName(event.Level))
	appendHTMLCell(buf, event.Logger)
	appendHTMLCell(buf, eventThreadName(event))
	appendHTMLCell(buf, event.Message)
	if len(event.Attrs) > 0 {
		var attrs bytes.Buffer
		appendPatternAttrs(&attrs, event.Attrs)
		appendHTMLCell(buf, attrs.String())
	} else {
		appendHTMLCell(buf, "")
	}
	buf.WriteString("</tr>")
	appendLayoutTerminator(buf, l.options)
	return nil
}

func (l HTMLLayout) AppendHeader(buf *bytes.Buffer) error {
	appendLayoutHeader(buf, l.options)
	return nil
}

func (l HTMLLayout) AppendFooter(buf *bytes.Buffer) error {
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

func gelfThrowableString(event Event, options LayoutOptions) string {
	if event.Throwable == nil {
		return eventErrorString(event)
	}
	if options.StacktraceAsString || options.IncludeStacktrace {
		return throwableStackString(event.Throwable)
	}
	return event.Throwable.String()
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

func appendCSVField(buf *bytes.Buffer, value string, comma bool) {
	if comma {
		buf.WriteByte(',')
	}
	if !csvNeedsQuote(value) {
		buf.WriteString(value)
		return
	}
	buf.WriteByte('"')
	for _, r := range value {
		if r == '"' {
			buf.WriteString(`""`)
			continue
		}
		buf.WriteRune(r)
	}
	buf.WriteByte('"')
}

func csvNeedsQuote(value string) bool {
	for _, r := range value {
		switch r {
		case ',', '"', '\r', '\n':
			return true
		}
	}
	return value == ""
}

func eventTime(when time.Time) time.Time {
	if when.IsZero() {
		return time.Now()
	}
	return when
}

func resolveHostName() string {
	name, err := os.Hostname()
	if err != nil || strings.TrimSpace(name) == "" {
		return "localhost"
	}
	return strings.TrimSpace(name)
}

func gelfAdditionalFieldKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" || key == "id" || strings.HasPrefix(key, "_") {
		return ""
	}
	return "_" + key
}

func syslogPriority(facility int, level slog.Level) int {
	if facility <= 0 || facility > 23 {
		facility = 1
	}
	return facility*8 + syslogSeverity(level)
}

func syslogSeverity(level slog.Level) int {
	switch {
	case level >= slog.LevelError:
		return 3
	case level >= slog.LevelWarn:
		return 4
	case level >= slog.LevelInfo:
		return 6
	default:
		return 7
	}
}

func appendSyslogToken(buf *bytes.Buffer, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		buf.WriteByte('-')
		return
	}
	for _, r := range value {
		if r <= ' ' || r == ']' || r == '"' {
			buf.WriteByte('_')
			continue
		}
		buf.WriteRune(r)
	}
}

func appendStructuredData(buf *bytes.Buffer, event Event) {
	if len(event.Attrs) == 0 {
		buf.WriteByte('-')
		return
	}
	buf.WriteString("[goark@32473")
	for _, attr := range event.Attrs {
		if strings.TrimSpace(attr.Key) == "" {
			continue
		}
		buf.WriteByte(' ')
		buf.WriteString(attr.Key)
		buf.WriteString("=\"")
		appendStructuredDataValue(buf, attrValueString(attr.Value))
		buf.WriteByte('"')
	}
	buf.WriteByte(']')
}

func appendStructuredDataValue(buf *bytes.Buffer, value string) {
	for _, r := range value {
		switch r {
		case '"', '\\', ']':
			buf.WriteByte('\\')
			buf.WriteRune(r)
		default:
			buf.WriteRune(r)
		}
	}
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
		return attrValueString(value)
	}
}

func appendHTMLCell(buf *bytes.Buffer, value string) {
	buf.WriteString("<td>")
	buf.WriteString(html.EscapeString(value))
	buf.WriteString("</td>")
}
