package goarklog

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
	"goark.dev/log/internal/logvalue"
)

const loggerNameKey = "goark.logger"

const defaultLoggerName = "goark"

const (
	// ThrowableAttrKey 是 goark-log 标准异常属性键。
	ThrowableAttrKey = "goark.throwable"
)

func normalizeContext(ctx context.Context) context.Context {
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
	Marker       *Marker
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

func throwableStackString(throwable *Throwable) string {
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

func throwableFromAttrs(attrs []slog.Attr) *Throwable {
	for index := len(attrs) - 1; index >= 0; index-- {
		attr := attrs[index]
		switch attr.Key {
		case ThrowableAttrKey, "throwable", "error", "err":
			if throwable := throwableFromValue(attr.Value); throwable != nil {
				return throwable
			}
		}
	}
	return nil
}

func throwableFromValue(value slog.Value) *Throwable {
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

func newEvent(ctx context.Context, logger string, handlerAttrs []slog.Attr, groups []string, record slog.Record) Event {
	if logger == "" {
		logger = defaultLoggerName
	}
	contextAttrs := ContextAttrs(ctx)
	attrs := make([]slog.Attr, 0, len(handlerAttrs)+len(contextAttrs)+record.NumAttrs())
	attrs = appendAttrs(attrs, nil, handlerAttrs)
	attrs = appendAttrs(attrs, nil, contextAttrs)
	record.Attrs(func(attr slog.Attr) bool {
		attrs = appendAttr(attrs, groups, attr)
		return true
	})
	return newEventFromCollected(ctx, logger, record.Time, record.Level, record.Message, record.PC, attrs)
}

func newEventFromAttrs(ctx context.Context, logger string, handlerAttrs []slog.Attr, groups []string, when time.Time, level slog.Level, message string, pc uintptr, attrs []slog.Attr, copyAttrs bool) Event {
	if logger == "" {
		logger = defaultLoggerName
	}
	contextAttrs := ContextAttrs(ctx)
	collected := makeEventAttrs(handlerAttrs, contextAttrs, groups, attrs, copyAttrs)
	return newEventFromCollected(ctx, logger, when, level, message, pc, collected)
}

func newEventFromCollected(ctx context.Context, logger string, when time.Time, level slog.Level, message string, pc uintptr, collected []slog.Attr) Event {
	marker := markerFromAttrs(collected)
	if marker == nil {
		if contextMarker, ok := ContextMarker(ctx); ok {
			marker = markerPointer(contextMarker)
		}
	}
	threadName := threadNameFromAttrs(collected)
	if threadName == "" {
		threadName = ContextThreadName(ctx)
	}
	if threadName == "" {
		threadName = defaultThreadName
	}
	contextStack := ContextStack(ctx)
	contextStack = appendContextStackValues(contextStack, contextStackFromAttrs(collected)...)
	return Event{
		Time:         when,
		Level:        level,
		Message:      message,
		Logger:       logger,
		PC:           pc,
		Attrs:        collected,
		Marker:       marker,
		Throwable:    throwableFromAttrs(collected),
		ThreadName:   threadName,
		ContextStack: contextStack,
	}
}

func makeEventAttrs(handlerAttrs []slog.Attr, contextAttrs []slog.Attr, groups []string, attrs []slog.Attr, copyAttrs bool) []slog.Attr {
	total := len(handlerAttrs) + len(contextAttrs) + len(attrs)
	if total == 0 {
		return nil
	}
	if !copyAttrs && len(handlerAttrs) == 0 && len(contextAttrs) == 0 && len(groups) == 0 && attrsCanShare(attrs) {
		return attrs
	}
	collected := make([]slog.Attr, 0, total)
	collected = appendAttrs(collected, nil, handlerAttrs)
	collected = appendAttrs(collected, nil, contextAttrs)
	collected = appendAttrs(collected, groups, attrs)
	return collected
}

func attrsCanShare(attrs []slog.Attr) bool {
	for _, attr := range attrs {
		if attr.Key == "" || attr.Key == loggerNameKey {
			return false
		}
		if attr.Value.Kind() == slog.KindLogValuer {
			return false
		}
	}
	return true
}

func appendAttrs(dst []slog.Attr, groups []string, attrs []slog.Attr) []slog.Attr {
	for _, attr := range attrs {
		dst = appendAttr(dst, groups, attr)
	}
	return dst
}

func appendAttr(dst []slog.Attr, groups []string, attr slog.Attr) []slog.Attr {
	attr = normalizeAttr(attr)
	if attr.Key == "" {
		return dst
	}
	if attr.Key == loggerNameKey {
		return dst
	}
	if len(groups) > 0 {
		attr.Key = groupKey(groups, attr.Key)
	}
	return append(dst, attr)
}

func normalizeAttr(attr slog.Attr) slog.Attr {
	attr.Value = attr.Value.Resolve()
	return attr
}

func groupKey(groups []string, key string) string {
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
