package goarklog

import (
	"context"
	"log/slog"
	"strings"
)

const (
	// ThreadNameAttrKey 是 goark-log 标准线程名属性键。
	ThreadNameAttrKey = "goark.thread"
	defaultThreadName = "main"
)

type threadNameContextKey struct{}

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

func threadNameFromAttrs(attrs []slog.Attr) string {
	for index := len(attrs) - 1; index >= 0; index-- {
		attr := attrs[index]
		switch attr.Key {
		case ThreadNameAttrKey, "thread", "goroutine":
			name := strings.TrimSpace(attrValueString(attr.Value))
			if name != "" {
				return name
			}
		}
	}
	return ""
}
