package logevent

import (
	"context"
	"errors"
	"log/slog"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"

	"goark.dev/log/internal/callsite"
	"goark.dev/log/internal/logcontext"
	"goark.dev/log/internal/logvalue"
)

const (
	// LoggerNameKey 是 slog 记录中覆盖 logger 名称的内部属性键。
	LoggerNameKey = "goark.logger"
	// DefaultLoggerName 是未显式命名时使用的默认 logger 名称。
	DefaultLoggerName = "goark"
	// ThrowableAttrKey 是 goark-log 标准异常属性键。
	ThrowableAttrKey = "goark.throwable"
	// DefaultThreadName 是事件没有显式逻辑线程名时的默认值。
	DefaultThreadName = logcontext.DefaultThreadName
)

// NormalizeContext 返回可安全读取的 context。
func NormalizeContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

// Event 是一次已经快照化的日志事件。
type Event struct {
	Time         time.Time
	Level        slog.Level
	Message      string
	Logger       string
	PC           uintptr
	Attrs        []slog.Attr
	Marker       *logcontext.Marker
	Throwable    *Throwable
	ThreadName   string
	ContextStack []string
	EndOfBatch   bool
}

// Attr 按键查找事件属性。
func (e Event) Attr(key string) (slog.Value, bool) {
	for index := len(e.Attrs) - 1; index >= 0; index-- {
		attr := e.Attrs[index]
		if attr.Key == key {
			return attr.Value, true
		}
	}
	return slog.Value{}, false
}

// MarkerPointer 返回 marker 的不可变快照指针。
func MarkerPointer(marker logcontext.Marker) *logcontext.Marker {
	return logcontext.MarkerPointer(marker)
}

// Throwable 是 Go error 的异常快照。
type Throwable struct {
	Type    string
	Message string
	Cause   *Throwable
	Stack   []string
}

// NewThrowable 把 error 转成轻量快照，不主动采集调用栈。
func NewThrowable(err error) *Throwable {
	return newThrowable(err, false, 0)
}

// NewThrowableWithStack 把 error 转成包含调用栈的快照。
func NewThrowableWithStack(err error) *Throwable {
	return newThrowable(err, true, 1)
}

func newThrowable(err error, withStack bool, skip int) *Throwable {
	if err == nil {
		return nil
	}
	throwable := &Throwable{
		Type:    reflect.TypeOf(err).String(),
		Message: err.Error(),
	}
	if withStack {
		throwable.Stack = captureThrowableStack(skip + 1)
	}
	if cause := errors.Unwrap(err); cause != nil && cause != err {
		throwable.Cause = NewThrowable(cause)
	}
	return throwable
}

func (t *Throwable) String() string {
	if t == nil {
		return ""
	}
	return t.Message
}

// ThrowableStackString 返回包含 cause 链的异常栈文本。
func ThrowableStackString(throwable *Throwable) string {
	if throwable == nil {
		return ""
	}
	var builder strings.Builder
	appendThrowableStackString(&builder, throwable)
	return builder.String()
}

func appendThrowableStackString(builder *strings.Builder, throwable *Throwable) {
	if throwable == nil {
		return
	}
	if throwable.Type != "" {
		builder.WriteString(throwable.Type)
		builder.WriteString(": ")
	}
	builder.WriteString(throwable.Message)
	for _, frame := range throwable.Stack {
		builder.WriteString("\n\tat ")
		builder.WriteString(frame)
	}
	if throwable.Cause != nil {
		builder.WriteString("\nCaused by: ")
		appendThrowableStackString(builder, throwable.Cause)
	}
}

// ThrowableAttr 把 error 按标准异常属性键注入 slog 记录。
func ThrowableAttr(err error) slog.Attr {
	return slog.Any(ThrowableAttrKey, NewThrowable(err))
}

// ThrowableWithStackAttr 把 error 和当前调用栈注入 slog 记录。
func ThrowableWithStackAttr(err error) slog.Attr {
	return slog.Any(ThrowableAttrKey, NewThrowableWithStack(err))
}

// ThrowableFromAttrs 从事件属性中提取异常快照。
func ThrowableFromAttrs(attrs []slog.Attr) *Throwable {
	for index := len(attrs) - 1; index >= 0; index-- {
		attr := attrs[index]
		switch attr.Key {
		case ThrowableAttrKey, "throwable", "error", "err":
			if throwable := ThrowableFromValue(attr.Value); throwable != nil {
				return throwable
			}
		}
	}
	return nil
}

// ThrowableFromValue 把 slog.Value 转成异常快照。
func ThrowableFromValue(value slog.Value) *Throwable {
	value = value.Resolve()
	switch typed := value.Any().(type) {
	case nil:
		return nil
	case *Throwable:
		return typed
	case Throwable:
		return &typed
	case error:
		return NewThrowable(typed)
	default:
		text := logvalue.String(value)
		if text == "" {
			return nil
		}
		return &Throwable{Type: "string", Message: text}
	}
}

func captureThrowableStack(skip int) []string {
	var pcs [32]uintptr
	count := runtime.Callers(skip+2, pcs[:])
	if count == 0 {
		return nil
	}
	frames := runtime.CallersFrames(pcs[:count])
	stack := make([]string, 0, count)
	for {
		frame, more := frames.Next()
		location := callsite.BaseName(frame.File)
		if frame.Line > 0 {
			location += ":" + strconv.Itoa(frame.Line)
		}
		if frame.Function != "" {
			location = frame.Function + "(" + location + ")"
		}
		stack = append(stack, location)
		if !more {
			break
		}
	}
	return stack
}

// AppendContextStackValues 返回追加并清理后的 NDC 栈快照。
func AppendContextStackValues(dst []string, values ...string) []string {
	return logcontext.AppendStackValues(dst, values...)
}

// ContextStackFromAttrs 从事件属性中提取 NDC 栈。
func ContextStackFromAttrs(attrs []slog.Attr) []string {
	return logcontext.StackFromAttrs(attrs)
}

// ContextStackString 把 NDC 栈编码为 pattern 布局使用的字符串。
func ContextStackString(values []string) string {
	return logcontext.StackString(values)
}

// MarkerFromAttrs 从事件属性中提取 marker。
func MarkerFromAttrs(attrs []slog.Attr) *logcontext.Marker {
	return logcontext.MarkerFromAttrs(attrs)
}

// ThreadNameFromAttrs 从事件属性中提取逻辑线程名。
func ThreadNameFromAttrs(attrs []slog.Attr) string {
	return logcontext.ThreadNameFromAttrs(attrs)
}

// New 从 slog.Record 创建事件快照。
func New(ctx context.Context, logger string, handlerAttrs []slog.Attr, groups []string, record slog.Record) Event {
	if logger == "" {
		logger = DefaultLoggerName
	}
	contextAttrs := logcontext.Attrs(ctx)
	attrs := make([]slog.Attr, 0, len(handlerAttrs)+len(contextAttrs)+record.NumAttrs())
	attrs = AppendAttrs(attrs, nil, handlerAttrs)
	attrs = AppendAttrs(attrs, nil, contextAttrs)
	record.Attrs(func(attr slog.Attr) bool {
		attrs = AppendAttr(attrs, groups, attr)
		return true
	})
	return NewFromCollected(ctx, logger, record.Time, record.Level, record.Message, record.PC, attrs)
}

// NewFromAttrs 从已给定属性创建事件快照。
func NewFromAttrs(ctx context.Context, logger string, handlerAttrs []slog.Attr, groups []string, when time.Time, level slog.Level, message string, pc uintptr, attrs []slog.Attr, copyAttrs bool) Event {
	if logger == "" {
		logger = DefaultLoggerName
	}
	contextAttrs := logcontext.Attrs(ctx)
	collected := MakeAttrs(handlerAttrs, contextAttrs, groups, attrs, copyAttrs)
	return NewFromCollected(ctx, logger, when, level, message, pc, collected)
}

// NewFromCollected 从已合并属性创建事件快照。
func NewFromCollected(ctx context.Context, logger string, when time.Time, level slog.Level, message string, pc uintptr, collected []slog.Attr) Event {
	marker := MarkerFromAttrs(collected)
	if marker == nil {
		if contextMarker, ok := logcontext.ContextMarker(ctx); ok {
			marker = MarkerPointer(contextMarker)
		}
	}
	threadName := ThreadNameFromAttrs(collected)
	if threadName == "" {
		threadName = logcontext.ThreadName(ctx)
	}
	if threadName == "" {
		threadName = DefaultThreadName
	}
	contextStack := logcontext.Stack(ctx)
	contextStack = AppendContextStackValues(contextStack, ContextStackFromAttrs(collected)...)
	return Event{
		Time:         when,
		Level:        level,
		Message:      message,
		Logger:       logger,
		PC:           pc,
		Attrs:        collected,
		Marker:       marker,
		Throwable:    ThrowableFromAttrs(collected),
		ThreadName:   threadName,
		ContextStack: contextStack,
	}
}

// MakeAttrs 合并 handler、context 和调用方属性。
func MakeAttrs(handlerAttrs []slog.Attr, contextAttrs []slog.Attr, groups []string, attrs []slog.Attr, copyAttrs bool) []slog.Attr {
	total := len(handlerAttrs) + len(contextAttrs) + len(attrs)
	if total == 0 {
		return nil
	}
	if !copyAttrs && len(handlerAttrs) == 0 && len(contextAttrs) == 0 && len(groups) == 0 && AttrsCanShare(attrs) {
		return attrs
	}
	collected := make([]slog.Attr, 0, total)
	collected = AppendAttrs(collected, nil, handlerAttrs)
	collected = AppendAttrs(collected, nil, contextAttrs)
	collected = AppendAttrs(collected, groups, attrs)
	return collected
}

// AttrsCanShare 判断属性切片是否可直接复用。
func AttrsCanShare(attrs []slog.Attr) bool {
	for _, attr := range attrs {
		if attr.Key == "" || attr.Key == LoggerNameKey {
			return false
		}
		if attr.Value.Kind() == slog.KindLogValuer {
			return false
		}
	}
	return true
}

// AppendAttrs 追加并规范化属性。
func AppendAttrs(dst []slog.Attr, groups []string, attrs []slog.Attr) []slog.Attr {
	for _, attr := range attrs {
		dst = AppendAttr(dst, groups, attr)
	}
	return dst
}

// AppendAttr 追加单个属性并应用 group 前缀。
func AppendAttr(dst []slog.Attr, groups []string, attr slog.Attr) []slog.Attr {
	attr = NormalizeAttr(attr)
	if attr.Key == "" {
		return dst
	}
	if attr.Key == LoggerNameKey {
		return dst
	}
	if len(groups) > 0 {
		attr.Key = GroupKey(groups, attr.Key)
	}
	return append(dst, attr)
}

// NormalizeAttr 解析 slog.LogValuer，得到稳定属性值。
func NormalizeAttr(attr slog.Attr) slog.Attr {
	attr.Value = attr.Value.Resolve()
	return attr
}

// GroupKey 生成 slog group 展平后的属性键。
func GroupKey(groups []string, key string) string {
	total := len(key)
	for _, group := range groups {
		total += len(group) + 1
	}
	out := make([]byte, 0, total)
	for _, group := range groups {
		if group == "" {
			continue
		}
		out = append(out, group...)
		out = append(out, '.')
	}
	out = append(out, key...)
	return string(out)
}
