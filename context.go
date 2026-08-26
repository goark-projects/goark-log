package goarklog

import (
	"context"
	"log/slog"
	"strings"
)

type contextAttrsKey struct{}

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
