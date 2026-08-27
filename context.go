package goarklog

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"goark.dev/log/internal/logvalue"
)

const (
	// ContextStackAttrKey 是 NDC/ContextStack 的标准属性键。
	ContextStackAttrKey = "goark.contextStack"
	// ThreadNameAttrKey 是 goark-log 标准线程名属性键。
	ThreadNameAttrKey = "goark.thread"
	defaultThreadName = "main"
)

type contextAttrsKey struct{}

type contextStackKey struct{}

type threadNameContextKey struct{}

// WithContextAttrs 返回携带日志上下文属性的新 context。
func WithContextAttrs(ctx context.Context, attrs ...slog.Attr) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(attrs) == 0 {
		return ctx
	}
	current := ContextAttrs(ctx)
	next := make([]slog.Attr, 0, len(current)+len(attrs))
	next = append(next, current...)
	for _, attr := range attrs {
		attr = normalizeAttr(attr)
		if attr.Key == "" || attr.Key == loggerNameKey {
			continue
		}
		next = append(next, attr)
	}
	if len(next) == len(current) {
		return ctx
	}
	return context.WithValue(ctx, contextAttrsKey{}, next)
}

// WithContextAttr 返回携带单个日志上下文属性的新 context。
func WithContextAttr(ctx context.Context, key string, value slog.Value) context.Context {
	key = strings.TrimSpace(key)
	if key == "" {
		return ctx
	}
	return WithContextAttrs(ctx, slog.Attr{Key: key, Value: value})
}

// ContextAttrs 返回 context 中的日志属性快照。
func ContextAttrs(ctx context.Context) []slog.Attr {
	if ctx == nil {
		return nil
	}
	attrs, ok := ctx.Value(contextAttrsKey{}).([]slog.Attr)
	if !ok || len(attrs) == 0 {
		return nil
	}
	return append([]slog.Attr(nil), attrs...)
}

// ThreadNameAttr 把 Go 运行期逻辑线程名注入 slog 记录。
func ThreadNameAttr(name string) slog.Attr {
	return slog.String(ThreadNameAttrKey, strings.TrimSpace(name))
}

// WithThreadName 返回携带逻辑线程名的新 context。
func WithThreadName(ctx context.Context, name string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return ctx
	}
	return context.WithValue(ctx, threadNameContextKey{}, name)
}

// ContextThreadName 返回 context 中的逻辑线程名。
func ContextThreadName(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	name, ok := ctx.Value(threadNameContextKey{}).(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(name)
}

// WithContextStack 返回追加 NDC 栈值的新 context。
func WithContextStack(ctx context.Context, values ...string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	next := appendContextStackValues(ContextStack(ctx), values...)
	if len(next) == 0 {
		return ctx
	}
	return context.WithValue(ctx, contextStackKey{}, next)
}

// ContextStack 返回 context 中的 NDC 栈快照。
func ContextStack(ctx context.Context) []string {
	if ctx == nil {
		return nil
	}
	values, ok := ctx.Value(contextStackKey{}).([]string)
	if !ok || len(values) == 0 {
		return nil
	}
	return append([]string(nil), values...)
}

func appendContextStackValues(dst []string, values ...string) []string {
	out := append([]string(nil), dst...)
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func contextStackFromAttrs(attrs []slog.Attr) []string {
	for index := len(attrs) - 1; index >= 0; index-- {
		attr := attrs[index]
		if attr.Key != ContextStackAttrKey && attr.Key != "contextStack" && attr.Key != "ndc" {
			continue
		}
		return contextStackFromValue(attr.Value)
	}
	return nil
}

func contextStackFromValue(value slog.Value) []string {
	value = value.Resolve()
	switch typed := value.Any().(type) {
	case []string:
		return appendContextStackValues(nil, typed...)
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			values = append(values, strings.TrimSpace(strings.Trim(fmt.Sprint(item), "[]")))
		}
		return appendContextStackValues(nil, values...)
	default:
		text := strings.TrimSpace(logvalue.String(value))
		if text == "" {
			return nil
		}
		return appendContextStackValues(nil, strings.Fields(text)...)
	}
}

func contextStackString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return strings.Join(values, " ")
}

func threadNameFromAttrs(attrs []slog.Attr) string {
	for index := len(attrs) - 1; index >= 0; index-- {
		attr := attrs[index]
		switch attr.Key {
		case ThreadNameAttrKey, "thread", "goroutine":
			name := strings.TrimSpace(logvalue.String(attr.Value))
			if name != "" {
				return name
			}
		}
	}
	return ""
}
