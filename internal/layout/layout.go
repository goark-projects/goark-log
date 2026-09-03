package layout

import (
	"bytes"
	"html"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"goark.dev/log/internal/layoutsupport"
	configlevel "goark.dev/log/internal/level"
	"goark.dev/log/internal/logevent"
	"goark.dev/log/internal/logvalue"
	"goark.dev/log/internal/textutil"
	"goark.dev/log/internal/timepattern"
)

const (
	// DefaultSpringBootPattern 是默认控制台输出格式，风格对齐 Spring Boot。
	DefaultSpringBootPattern = "%d{yyyy-MM-dd HH:mm:ss.SSS}  %5level %pid - [%15.15thread] %-40.40logger{1.2*} : %msg%attrs%n"
	defaultTimeFormat        = timepattern.DefaultLayout
	defaultThreadName        = logevent.DefaultThreadName
)

var processIDString = strconv.Itoa(os.Getpid())
var patternSequence atomic.Uint64
var patternStartTime = time.Now()
var bufferPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

// Event 是布局看到的事件快照。
type Event = logevent.Event

// Throwable 是布局输出使用的异常快照。
type Throwable = logevent.Throwable

// Layout 把日志事件编码为字节。
type Layout interface {
	Format(buf *bytes.Buffer, event Event) error
}

// LayoutOptions 描述通用结构化布局参数。
type LayoutOptions struct {
	// Compact 禁用默认事件换行，适合由上层协议自行分隔事件的场景。
	Compact bool
	// EventEOL 在 Compact 模式下仍然为每个事件追加换行。
	EventEOL bool
	// Complete 启用布局页眉和页脚输出，由 appender 在流生命周期内写一次。
	Complete bool
	// IncludeStacktrace 在支持异常结构的布局中输出完整异常结构。
	IncludeStacktrace bool
	// StacktraceAsString 将异常栈输出为字符串，便于兼容文本型采集器。
	StacktraceAsString bool
	// PropertiesAsList 将上下文属性输出为键值列表。
	PropertiesAsList bool
	// IncludeNullDelimiter 在事件结束后追加 NUL 字节，用于 GELF 等协议分隔。
	IncludeNullDelimiter bool
	// DisableANSI 禁用 PatternLayout 中 highlight/style 转换器的 ANSI SGR 输出。
	DisableANSI bool
	// Header 是 Complete 模式下流打开时写入的页眉。
	Header string
	// Footer 是 Complete 模式下流关闭时写入的页脚。
	Footer string
}

type lifecycleLayout interface {
	AppendHeader(buf *bytes.Buffer) error
	AppendFooter(buf *bytes.Buffer) error
}

type cloneableLayout interface {
	CloneLayout() Layout
}

type synchronizedFormatLayout interface {
	RequiresSynchronizedFormatting() bool
}

// CloneLayout 为 appender 绑定创建隔离布局，避免生命周期状态跨输出流共享。
func CloneLayout(layout Layout) Layout {
	if layout == nil {
		return nil
	}
	cloneable, ok := layout.(cloneableLayout)
	if !ok {
		return layout
	}
	cloned := cloneable.CloneLayout()
	if cloned == nil {
		return layout
	}
	return cloned
}

// RequiresSynchronizedFormatting 判断布局格式化是否必须与写入在同一临界区内完成。
func RequiresSynchronizedFormatting(layout Layout) bool {
	synchronized, ok := layout.(synchronizedFormatLayout)
	return ok && synchronized.RequiresSynchronizedFormatting()
}

// NewDefaultLayout 创建默认 Spring Boot 风格布局。
func NewDefaultLayout() Layout {
	layout, _ := NewPatternLayout(DefaultSpringBootPattern)
	return layout
}

// TextLayout 输出稳定的 key=value 文本。
type TextLayout struct{}

func (TextLayout) Format(buf *bytes.Buffer, event Event) error {
	logvalue.AppendKey(buf, "time")
	buf.Write(event.Time.AppendFormat(buf.AvailableBuffer(), defaultTimeFormat))
	logvalue.AppendKeyValue(buf, "level", levelName(event.Level))
	logvalue.AppendKeyValue(buf, "logger", event.Logger)
	logvalue.AppendKeyValue(buf, "msg", event.Message)
	for _, attr := range event.Attrs {
		logvalue.AppendKeyValueAttr(buf, attr.Key, attr.Value)
	}
	buf.WriteByte('\n')
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
	appendCSVField(buf, layoutsupport.EventTime(event.Time).Format(defaultTimeFormat), false)
	appendCSVField(buf, levelName(event.Level), true)
	appendCSVField(buf, event.Logger, true)
	appendCSVField(buf, eventThreadName(event), true)
	appendCSVField(buf, event.Message, true)
	if len(event.Attrs) == 0 {
		appendLayoutTerminator(buf, l.options)
		return nil
	}
	var attrs bytes.Buffer
	logvalue.AppendPatternAttrs(&attrs, event.Attrs)
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
	appendHTMLCell(buf, layoutsupport.EventTime(event.Time).Format(defaultTimeFormat))
	appendHTMLCell(buf, levelName(event.Level))
	appendHTMLCell(buf, event.Logger)
	appendHTMLCell(buf, eventThreadName(event))
	appendHTMLCell(buf, event.Message)
	if len(event.Attrs) > 0 {
		var attrs bytes.Buffer
		logvalue.AppendPatternAttrs(&attrs, event.Attrs)
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

func appendHTMLCell(buf *bytes.Buffer, value string) {
	buf.WriteString("<td>")
	buf.WriteString(html.EscapeString(value))
	buf.WriteString("</td>")
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
	when := layoutsupport.EventTime(event.Time)
	buf.WriteByte('{')
	logvalue.AppendJSONFieldString(buf, "version", "1.1", false)
	logvalue.AppendJSONFieldString(buf, "host", layoutsupport.HostName(), true)
	logvalue.AppendJSONFieldString(buf, "short_message", event.Message, true)
	if thrown := gelfThrowableString(event, l.options); thrown != "" {
		logvalue.AppendJSONFieldString(buf, "full_message", thrown, true)
	}
	logvalue.AppendJSONKey(buf, "timestamp", true)
	buf.Write(strconv.AppendFloat(buf.AvailableBuffer(), float64(when.UnixNano())/1e9, 'f', 6, 64))
	logvalue.AppendJSONKey(buf, "level", true)
	buf.Write(strconv.AppendInt(buf.AvailableBuffer(), int64(syslogSeverity(event.Level)), 10))
	logvalue.AppendJSONFieldString(buf, "_logger", event.Logger, true)
	logvalue.AppendJSONFieldString(buf, "_thread", eventThreadName(event), true)
	if marker := eventMarkerString(event); marker != "" {
		logvalue.AppendJSONFieldString(buf, "_marker", marker, true)
	}
	for _, attr := range event.Attrs {
		key := gelfAdditionalFieldKey(attr.Key)
		if key == "" {
			continue
		}
		logvalue.AppendJSONFieldValue(buf, key, attr.Value, true)
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

func gelfThrowableString(event Event, options LayoutOptions) string {
	if event.Throwable == nil {
		return eventErrorString(event)
	}
	if options.StacktraceAsString || options.IncludeStacktrace {
		return throwableStackString(event.Throwable)
	}
	return event.Throwable.String()
}

func gelfAdditionalFieldKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" || key == "id" || strings.HasPrefix(key, "_") {
		return ""
	}
	return "_" + key
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
	buf.WriteString(layoutsupport.EventTime(event.Time).UTC().Format(time.RFC3339Nano))
	buf.WriteByte(' ')
	appendSyslogToken(buf, layoutsupport.HostName())
	buf.WriteByte(' ')
	appendSyslogToken(buf, textutil.FirstNonBlank(l.AppName, event.Logger, "goark"))
	buf.WriteByte(' ')
	appendSyslogToken(buf, processIDString)
	buf.WriteByte(' ')
	appendSyslogToken(buf, textutil.FirstNonBlank(l.MessageID, "-"))
	buf.WriteByte(' ')
	appendStructuredData(buf, event)
	buf.WriteByte(' ')
	buf.WriteString(event.Message)
	buf.WriteByte('\n')
	return nil
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

func levelName(level slog.Level) string {
	return configlevel.NameDefault(level)
}

func throwableStackString(throwable *Throwable) string {
	return logevent.ThrowableStackString(throwable)
}

func contextStackString(values []string) string {
	return logevent.ContextStackString(values)
}

func normalizeAttr(attr slog.Attr) slog.Attr {
	return logevent.NormalizeAttr(attr)
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
		appendStructuredDataValue(buf, logvalue.String(attr.Value))
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

func appendLayoutHeader(buf *bytes.Buffer, options LayoutOptions) {
	if options.Complete && strings.TrimSpace(options.Header) != "" {
		buf.WriteString(options.Header)
	}
}

func appendLayoutFooter(buf *bytes.Buffer, options LayoutOptions) {
	if options.Complete && strings.TrimSpace(options.Footer) != "" {
		buf.WriteString(options.Footer)
	}
}

func appendLayoutTerminator(buf *bytes.Buffer, options LayoutOptions) {
	if options.EventEOL || !options.Compact {
		buf.WriteByte('\n')
	}
	if options.IncludeNullDelimiter {
		buf.WriteByte(0)
	}
}

// WriteHeader 在支持生命周期的布局上写入流页眉。
func WriteHeader(writer io.Writer, layout Layout) (int, error) {
	lifecycle, ok := layout.(lifecycleLayout)
	if !ok {
		return 0, nil
	}
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer releaseBuffer(buf)
	if err := lifecycle.AppendHeader(buf); err != nil {
		return 0, err
	}
	if buf.Len() == 0 {
		return 0, nil
	}
	return writer.Write(buf.Bytes())
}

// WriteFooter 在支持生命周期的布局上写入流页脚。
func WriteFooter(writer io.Writer, layout Layout) (int, error) {
	lifecycle, ok := layout.(lifecycleLayout)
	if !ok {
		return 0, nil
	}
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer releaseBuffer(buf)
	if err := lifecycle.AppendFooter(buf); err != nil {
		return 0, err
	}
	if buf.Len() == 0 {
		return 0, nil
	}
	return writer.Write(buf.Bytes())
}

func releaseBuffer(buf *bytes.Buffer) {
	if buf.Cap() > 64*1024 {
		return
	}
	buf.Reset()
	bufferPool.Put(buf)
}
