// Package goarklog 提供基于 log/slog 的 Goark 日志实现。
//
// 推荐的稳定入口是 NewHandler、NewConfigured、ConfigureDefault、Appender、
// Layout、LayoutOptions、Options 以及各 appender 的 Option 构造函数。YAML 文件结构由
// LoadOptions 和 NewConfigured 系列函数解析，内部解析结构不作为公共 API 暴露。
package goarklog

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"time"

	internallayout "goark.dev/log/internal/layout"
	"goark.dev/log/internal/logcontext"
	"goark.dev/log/internal/logevent"
	logmessage "goark.dev/log/internal/message"
)

const loggerNameKey = logevent.LoggerNameKey

const defaultLoggerName = logevent.DefaultLoggerName

const (
	// ThrowableAttrKey 是 goark-log 标准异常属性键。
	ThrowableAttrKey = logevent.ThrowableAttrKey
	// ContextStackAttrKey 是 NDC/ContextStack 的标准属性键。
	ContextStackAttrKey = logcontext.StackAttrKey
	// MarkerAttrKey 是 goark-log 标准 marker 属性键。
	MarkerAttrKey = logcontext.MarkerAttrKey
	// ThreadNameAttrKey 是 goark-log 标准线程名属性键。
	ThreadNameAttrKey = logcontext.ThreadNameAttrKey
	defaultThreadName = logevent.DefaultThreadName
	// StructuredDataIDAttrKey 是结构化消息 ID 的标准属性键。
	StructuredDataIDAttrKey = logmessage.StructuredDataIDAttrKey
	// StructuredDataTypeAttrKey 是结构化消息类型的标准属性键。
	StructuredDataTypeAttrKey = logmessage.StructuredDataTypeAttrKey
	// DefaultSpringBootPattern 是默认控制台输出格式，风格对齐 Spring Boot。
	DefaultSpringBootPattern = internallayout.DefaultSpringBootPattern
)

// Event 是一次已经快照化的日志事件。
type Event = logevent.Event

// Marker 表示事件标签，支持父级层次匹配。
type Marker = logcontext.Marker

// Throwable 是 Go error 的异常快照。
type Throwable = logevent.Throwable

// Message 表示可被日志事件快照化的消息对象。
type Message = logmessage.Message

// AttributedMessage 表示会同时贡献结构化属性的消息对象。
type AttributedMessage = logmessage.AttributedMessage

// MessageFactory 创建日志消息对象。
type MessageFactory = logmessage.MessageFactory

// MessageFactoryFunc 把函数适配为 MessageFactory。
type MessageFactoryFunc = logmessage.MessageFactoryFunc

// ParameterizedMessageFactory 创建 {} 占位符参数化消息。
type ParameterizedMessageFactory = logmessage.ParameterizedMessageFactory

// SimpleMessageFactory 忽略参数并创建普通字符串消息。
type SimpleMessageFactory = logmessage.SimpleMessageFactory

// SimpleMessage 是不可变字符串消息。
type SimpleMessage = logmessage.SimpleMessage

// ParameterizedMessage 使用 {} 占位符格式化消息。
type ParameterizedMessage = logmessage.ParameterizedMessage

// MapMessage 用属性集合表达消息，适合结构化日志。
type MapMessage = logmessage.MapMessage

// StructuredDataMessage 表示 RFC5424 风格的结构化消息。
type StructuredDataMessage = logmessage.StructuredDataMessage

// Layout 把日志事件编码为字节。
type Layout = internallayout.Layout

// LayoutOptions 描述通用结构化布局参数。
type LayoutOptions = internallayout.LayoutOptions

// TextLayout 输出稳定的 key=value 文本。
type TextLayout = internallayout.TextLayout

// CSVLayout 输出单行 CSV，字段顺序固定。
type CSVLayout = internallayout.CSVLayout

// HTMLLayout 输出 HTML 表格行，适合文件或控制台片段组合。
type HTMLLayout = internallayout.HTMLLayout

// GELFLayout 输出 Graylog Extended Log Format 单行 JSON。
type GELFLayout = internallayout.GELFLayout

// RFC5424Layout 输出 RFC 5424 syslog 单行事件。
type RFC5424Layout = internallayout.RFC5424Layout

// SyslogLayout 是 RFC5424Layout 的语义别名。
type SyslogLayout = internallayout.SyslogLayout

// JSONLayout 输出 JSON 事件。
type JSONLayout = internallayout.JSONLayout

// XMLLayout 输出单事件 XML 片段。
type XMLLayout = internallayout.XMLLayout

// YAMLLayout 输出单事件 YAML 文档。
type YAMLLayout = internallayout.YAMLLayout

// PatternLayout 支持常用日志 pattern 占位符。
type PatternLayout = internallayout.PatternLayout

// JSONTemplateLayout 按 JSON 事件模板输出日志事件。
type JSONTemplateLayout = internallayout.JSONTemplateLayout

// JSONTemplateLayoutOption 调整 JSONTemplateLayout 编译行为。
type JSONTemplateLayoutOption = internallayout.JSONTemplateLayoutOption

// JSONTemplateResolver 是 JSON Template 字段值编码器。
type JSONTemplateResolver = internallayout.JSONTemplateResolver

// JSONTemplateResolverFactory 从配置构建 JSON Template resolver。
type JSONTemplateResolverFactory = internallayout.JSONTemplateResolverFactory

// JSONTemplateResolverBuildConfig 是 JSON Template resolver 插件的构建输入。
type JSONTemplateResolverBuildConfig = internallayout.JSONTemplateResolverBuildConfig

// NewMarker 创建不可变语义的 marker 值对象。
func NewMarker(name string, parents ...Marker) Marker {
	return logcontext.NewMarker(name, parents...)
}

func markerPointer(marker Marker) *Marker {
	return logevent.MarkerPointer(marker)
}

// WithContextAttrs 返回携带日志上下文属性的新 context。
func WithContextAttrs(ctx context.Context, attrs ...slog.Attr) context.Context {
	return logcontext.WithAttrs(ctx, attrs...)
}

// WithContextAttr 返回携带单个日志上下文属性的新 context。
func WithContextAttr(ctx context.Context, key string, value slog.Value) context.Context {
	return logcontext.WithAttr(ctx, key, value)
}

// ContextAttrs 返回 context 中的日志属性快照。
func ContextAttrs(ctx context.Context) []slog.Attr {
	return logcontext.Attrs(ctx)
}

// MarkerAttr 把 marker 按标准属性键注入 slog 记录。
func MarkerAttr(marker Marker) slog.Attr {
	return logcontext.MarkerAttr(marker)
}

// WithMarker 返回携带 marker 的 context，适合请求链路级别复用。
func WithMarker(ctx context.Context, marker Marker) context.Context {
	return logcontext.WithMarker(ctx, marker)
}

// ContextMarker 返回 context 上绑定的 marker 快照。
func ContextMarker(ctx context.Context) (Marker, bool) {
	return logcontext.ContextMarker(ctx)
}

// ThreadNameAttr 把 Go 运行期逻辑线程名注入 slog 记录。
func ThreadNameAttr(name string) slog.Attr {
	return logcontext.ThreadNameAttr(name)
}

// WithThreadName 返回携带逻辑线程名的新 context。
func WithThreadName(ctx context.Context, name string) context.Context {
	return logcontext.WithThreadName(ctx, name)
}

// ContextThreadName 返回 context 中的逻辑线程名。
func ContextThreadName(ctx context.Context) string {
	return logcontext.ThreadName(ctx)
}

// WithContextStack 返回追加 NDC 栈值的新 context。
func WithContextStack(ctx context.Context, values ...string) context.Context {
	return logcontext.WithStack(ctx, values...)
}

// ContextStack 返回 context 中的 NDC 栈快照。
func ContextStack(ctx context.Context) []string {
	return logcontext.Stack(ctx)
}

// NewThrowable 把 error 转成轻量快照，不主动采集调用栈。
func NewThrowable(err error) *Throwable {
	return logevent.NewThrowable(err)
}

// NewThrowableWithStack 把 error 转成包含调用栈的快照。
func NewThrowableWithStack(err error) *Throwable {
	return logevent.NewThrowableWithStack(err)
}

// ThrowableAttr 把 error 按标准异常属性键注入 slog 记录。
func ThrowableAttr(err error) slog.Attr {
	return logevent.ThrowableAttr(err)
}

// ThrowableWithStackAttr 把 error 和当前调用栈注入 slog 记录。
func ThrowableWithStackAttr(err error) slog.Attr {
	return logevent.ThrowableWithStackAttr(err)
}

// NewSimpleMessage 创建字符串消息。
func NewSimpleMessage(text string) SimpleMessage {
	return logmessage.NewSimpleMessage(text)
}

// NewParameterizedMessage 创建参数化消息，参数会被快照复制。
func NewParameterizedMessage(pattern string, args ...any) ParameterizedMessage {
	return logmessage.NewParameterizedMessage(pattern, args...)
}

// NewMapMessage 创建结构化属性消息。
func NewMapMessage(attrs ...slog.Attr) MapMessage {
	return logmessage.NewMapMessage(attrs...)
}

// NewStructuredDataMessage 创建结构化数据消息。
func NewStructuredDataMessage(id string, msgType string, message string, attrs ...slog.Attr) StructuredDataMessage {
	return logmessage.NewStructuredDataMessage(id, msgType, message, attrs...)
}

// NewDefaultLayout 创建默认 Spring Boot 风格布局。
func NewDefaultLayout() Layout {
	return internallayout.NewDefaultLayout()
}

// NewCSVLayout 创建可配置 CSV 布局。
func NewCSVLayout(options LayoutOptions) CSVLayout {
	return internallayout.NewCSVLayout(options)
}

// NewHTMLLayout 创建可配置 HTML 布局。
func NewHTMLLayout(options LayoutOptions) HTMLLayout {
	return internallayout.NewHTMLLayout(options)
}

// NewGELFLayout 创建可配置 GELF 布局。
func NewGELFLayout(options LayoutOptions) GELFLayout {
	return internallayout.NewGELFLayout(options)
}

// NewJSONLayout 创建可配置 JSON 布局。
func NewJSONLayout(options LayoutOptions) JSONLayout {
	return internallayout.NewJSONLayout(options)
}

// NewXMLLayout 创建可配置 XML 布局。
func NewXMLLayout(options LayoutOptions) XMLLayout {
	return internallayout.NewXMLLayout(options)
}

// NewYAMLLayout 创建可配置 YAML 布局。
func NewYAMLLayout(options LayoutOptions) YAMLLayout {
	return internallayout.NewYAMLLayout(options)
}

// NewPatternLayout 编译 pattern，避免热路径反复解析。
func NewPatternLayout(pattern string) (*PatternLayout, error) {
	return internallayout.NewPatternLayout(pattern)
}

// NewPatternLayoutWithOptions 使用指定布局参数编译 pattern。
func NewPatternLayoutWithOptions(pattern string, options LayoutOptions) (*PatternLayout, error) {
	return internallayout.NewPatternLayoutWithOptions(pattern, options)
}

// NewCharsetLayout 使用指定字符集编码布局结果，UTF-8 不增加转换层。
func NewCharsetLayout(layout Layout, charset string) (Layout, error) {
	return internallayout.NewCharsetLayout(layout, charset)
}

// WithJSONTemplateResolverRegistry 设置用于解析自定义 resolver 的插件注册表。
func WithJSONTemplateResolverRegistry(registry *PluginRegistry) JSONTemplateLayoutOption {
	if registry == nil {
		registry = DefaultPluginRegistry()
	}
	return internallayout.WithJSONTemplateResolverLookup(registry.JSONTemplateResolverFactory)
}

// WithJSONTemplateLayoutOptions 设置 JSON Template 布局的通用输出参数。
func WithJSONTemplateLayoutOptions(layoutOptions LayoutOptions) JSONTemplateLayoutOption {
	return internallayout.WithJSONTemplateLayoutOptions(layoutOptions)
}

// NewJSONTemplateLayout 从 JSON 事件模板编译布局。
func NewJSONTemplateLayout(template string, options ...JSONTemplateLayoutOption) (*JSONTemplateLayout, error) {
	return internallayout.NewJSONTemplateLayout(template, jsonTemplateLayoutOptions(options...)...)
}

// NewJSONTemplateLayoutFromFile 从本地文件编译 JSON 事件模板。
func NewJSONTemplateLayoutFromFile(path string, options ...JSONTemplateLayoutOption) (*JSONTemplateLayout, error) {
	return internallayout.NewJSONTemplateLayoutFromFile(path, jsonTemplateLayoutOptions(options...)...)
}

func normalizeContext(ctx context.Context) context.Context {
	return logevent.NormalizeContext(ctx)
}

func throwableStackString(throwable *Throwable) string {
	return logevent.ThrowableStackString(throwable)
}

func throwableFromAttrs(attrs []slog.Attr) *Throwable {
	return logevent.ThrowableFromAttrs(attrs)
}

func appendContextStackValues(dst []string, values ...string) []string {
	return logevent.AppendContextStackValues(dst, values...)
}

func contextStackFromAttrs(attrs []slog.Attr) []string {
	return logevent.ContextStackFromAttrs(attrs)
}

func contextStackString(values []string) string {
	return logevent.ContextStackString(values)
}

func markerFromAttrs(attrs []slog.Attr) *Marker {
	return logevent.MarkerFromAttrs(attrs)
}

func threadNameFromAttrs(attrs []slog.Attr) string {
	return logevent.ThreadNameFromAttrs(attrs)
}

func newEvent(ctx context.Context, logger string, handlerAttrs []slog.Attr, groups []string, record slog.Record) Event {
	return logevent.New(ctx, logger, handlerAttrs, groups, record)
}

func newEventFromAttrs(ctx context.Context, logger string, handlerAttrs []slog.Attr, groups []string, when time.Time, level slog.Level, message string, pc uintptr, attrs []slog.Attr, copyAttrs bool) Event {
	return logevent.NewFromAttrs(ctx, logger, handlerAttrs, groups, when, level, message, pc, attrs, copyAttrs)
}

func newEventFromCollected(ctx context.Context, logger string, when time.Time, level slog.Level, message string, pc uintptr, collected []slog.Attr) Event {
	return logevent.NewFromCollected(ctx, logger, when, level, message, pc, collected)
}

func makeEventAttrs(handlerAttrs []slog.Attr, contextAttrs []slog.Attr, groups []string, attrs []slog.Attr, copyAttrs bool) []slog.Attr {
	return logevent.MakeAttrs(handlerAttrs, contextAttrs, groups, attrs, copyAttrs)
}

func attrsCanShare(attrs []slog.Attr) bool {
	return logevent.AttrsCanShare(attrs)
}

func appendAttrs(dst []slog.Attr, groups []string, attrs []slog.Attr) []slog.Attr {
	return logevent.AppendAttrs(dst, groups, attrs)
}

func appendAttr(dst []slog.Attr, groups []string, attr slog.Attr) []slog.Attr {
	return logevent.AppendAttr(dst, groups, attr)
}

func normalizeAttr(attr slog.Attr) slog.Attr {
	return logevent.NormalizeAttr(attr)
}

func groupKey(groups []string, key string) string {
	return logevent.GroupKey(groups, key)
}

func jsonTemplateLayoutOptions(options ...JSONTemplateLayoutOption) []JSONTemplateLayoutOption {
	merged := make([]JSONTemplateLayoutOption, 0, len(options)+1)
	merged = append(merged, WithJSONTemplateResolverRegistry(DefaultPluginRegistry()))
	return append(merged, options...)
}

func appendJSONEvent(buf *bytes.Buffer, when time.Time, level slog.Level, logger string, message string, attrs []slog.Attr) {
	internallayout.AppendJSONEvent(buf, when, level, logger, message, attrs)
}

func appendJSONFixedEvent(buf *bytes.Buffer, when time.Time, level slog.Level, logger string, message string, attrs [3]slog.Attr, count int) {
	internallayout.AppendJSONFixedEvent(buf, when, level, logger, message, attrs, count)
}

func writeLayoutHeader(writer io.Writer, layout Layout) (int, error) {
	return internallayout.WriteHeader(writer, layout)
}

func writeLayoutFooter(writer io.Writer, layout Layout) (int, error) {
	return internallayout.WriteFooter(writer, layout)
}
