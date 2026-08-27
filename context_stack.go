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
)

type contextStackKey struct{}

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
