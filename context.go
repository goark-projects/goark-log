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
	// MarkerAttrKey 是 goark-log 标准 marker 属性键。
	MarkerAttrKey = "goark.marker"
	// ThreadNameAttrKey 是 goark-log 标准线程名属性键。
	ThreadNameAttrKey = "goark.thread"
	defaultThreadName = "main"
)

type contextAttrsKey struct{}

type markerContextKey struct{}

type contextStackKey struct{}

type threadNameContextKey struct{}

// Marker 表示事件标签，支持父级层次匹配。
type Marker struct {
	Name    string
	Parents []Marker
}

// NewMarker 创建不可变语义的 marker 值对象。
func NewMarker(name string, parents ...Marker) Marker {
	marker := Marker{Name: strings.TrimSpace(name)}
	if len(parents) == 0 {
		return marker
	}
	marker.Parents = make([]Marker, 0, len(parents))
	for _, parent := range parents {
		if parent.Name == "" {
			continue
		}
		marker.Parents = append(marker.Parents, parent)
	}
	return marker
}

func markerPointer(marker Marker) *Marker {
	if marker.Name == "" {
		return nil
	}
	copied := Marker{
		Name:    marker.Name,
		Parents: append([]Marker(nil), marker.Parents...),
	}
	return &copied
}

// Contains 判断当前 marker 或任意父级是否匹配给定名称。
func (m Marker) Contains(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	if m.Name == name {
		return true
	}
	for _, parent := range m.Parents {
		if parent.Contains(name) {
			return true
		}
	}
	return false
}

func (m Marker) String() string {
	return m.Name
}

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

// MarkerAttr 把 marker 按标准属性键注入 slog 记录。
func MarkerAttr(marker Marker) slog.Attr {
	return slog.Any(MarkerAttrKey, marker)
}

// WithMarker 返回携带 marker 的 context，适合请求链路级别复用。
func WithMarker(ctx context.Context, marker Marker) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if marker.Name == "" {
		return ctx
	}
	return context.WithValue(ctx, markerContextKey{}, marker)
}

// ContextMarker 返回 context 上绑定的 marker 快照。
func ContextMarker(ctx context.Context) (Marker, bool) {
	if ctx == nil {
		return Marker{}, false
	}
	marker, ok := ctx.Value(markerContextKey{}).(Marker)
	if !ok || marker.Name == "" {
		return Marker{}, false
	}
	return marker, true
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

func markerFromAttrs(attrs []slog.Attr) *Marker {
	for index := len(attrs) - 1; index >= 0; index-- {
		attr := attrs[index]
		if attr.Key != MarkerAttrKey && attr.Key != "marker" {
			continue
		}
		if marker, ok := markerFromValue(attr.Value); ok {
			return markerPointer(marker)
		}
	}
	return nil
}

func markerFromValue(value slog.Value) (Marker, bool) {
	value = value.Resolve()
	switch typed := value.Any().(type) {
	case Marker:
		return typed, typed.Name != ""
	case *Marker:
		if typed == nil || typed.Name == "" {
			return Marker{}, false
		}
		return *typed, true
	default:
		name := strings.TrimSpace(logvalue.String(value))
		if name == "" {
			return Marker{}, false
		}
		return NewMarker(name), true
	}
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
