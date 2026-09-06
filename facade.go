// Package log 提供基于 log/slog 的 Goark 日志实现。
//
// 推荐的稳定入口是 NewHandler、NewConfigured、ConfigureDefault、Appender、
// Layout、LayoutOptions、Options 以及各 appender 的 Option 构造函数。YAML 文件结构由
// LoadOptions 和 NewConfigured 系列函数解析，内部解析结构不作为公共 API 暴露。
package log

import (
	"context"
	"log/slog"
	"time"

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

// NewMarker 创建不可变语义的 marker 值对象。
func NewMarker(name string, parents ...Marker) Marker {
	return logcontext.NewMarker(name, parents...)
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
func NewStructuredDataMessage(
	id string,
	msgType string,
	message string,
	attrs ...slog.Attr,
) StructuredDataMessage {
	return logmessage.NewStructuredDataMessage(id, msgType, message, attrs...)
}

func newEvent(
	ctx context.Context,
	logger string,
	handlerAttrs []slog.Attr,
	groups []string,
	record slog.Record,
) Event {
	return logevent.New(ctx, logger, handlerAttrs, groups, record)
}

func newEventFromAttrs(
	ctx context.Context,
	logger string,
	handlerAttrs []slog.Attr,
	groups []string,
	when time.Time,
	level slog.Level,
	message string,
	pc uintptr,
	attrs []slog.Attr,
	copyAttrs bool,
) Event {
	return logevent.NewFromAttrs(
		ctx,
		logger,
		handlerAttrs,
		groups,
		when,
		level,
		message,
		pc,
		attrs,
		copyAttrs,
	)
}

func appendAttrs(dst []slog.Attr, groups []string, attrs []slog.Attr) []slog.Attr {
	return logevent.AppendAttrs(dst, groups, attrs)
}

func normalizeAttr(attr slog.Attr) slog.Attr {
	return logevent.NormalizeAttr(attr)
}
