package logcontext

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"goark.dev/log/internal/logvalue"
)

const (
	loggerNameKey = "goark.logger"

	// StackAttrKey 是 NDC/ContextStack 的标准属性键。
	StackAttrKey = "goark.contextStack"
	// MarkerAttrKey 是 goark-log 标准 marker 属性键。
	MarkerAttrKey = "goark.marker"
	// ThreadNameAttrKey 是 goark-log 标准线程名属性键。
	ThreadNameAttrKey = "goark.thread"
	// DefaultThreadName 是事件没有显式逻辑线程名时的默认值。
	DefaultThreadName = "main"
)

type attrsKey struct{}

type markerKey struct{}

type stackKey struct{}

type threadNameKey struct{}

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

// MarkerPointer 返回 marker 的不可变快照指针。
func MarkerPointer(marker Marker) *Marker {
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

// WithAttrs 返回携带日志上下文属性的新 context。
func WithAttrs(ctx context.Context, attrs ...slog.Attr) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(attrs) == 0 {
		return ctx
	}
	current := Attrs(ctx)
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
	return context.WithValue(ctx, attrsKey{}, next)
}

// WithAttr 返回携带单个日志上下文属性的新 context。
func WithAttr(ctx context.Context, key string, value slog.Value) context.Context {
	key = strings.TrimSpace(key)
	if key == "" {
		return ctx
	}
	return WithAttrs(ctx, slog.Attr{Key: key, Value: value})
}

// Attrs 返回 context 中的日志属性快照。
func Attrs(ctx context.Context) []slog.Attr {
	if ctx == nil {
		return nil
	}
	attrs, ok := ctx.Value(attrsKey{}).([]slog.Attr)
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
	return context.WithValue(ctx, markerKey{}, marker)
}

// ContextMarker 返回 context 上绑定的 marker 快照。
func ContextMarker(ctx context.Context) (Marker, bool) {
	if ctx == nil {
		return Marker{}, false
	}
	marker, ok := ctx.Value(markerKey{}).(Marker)
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
	return context.WithValue(ctx, threadNameKey{}, name)
}

// ThreadName 返回 context 中的逻辑线程名。
func ThreadName(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	name, ok := ctx.Value(threadNameKey{}).(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(name)
}

// WithStack 返回追加 NDC 栈值的新 context。
func WithStack(ctx context.Context, values ...string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	next := AppendStackValues(Stack(ctx), values...)
	if len(next) == 0 {
		return ctx
	}
	return context.WithValue(ctx, stackKey{}, next)
}

// Stack 返回 context 中的 NDC 栈快照。
func Stack(ctx context.Context) []string {
	if ctx == nil {
		return nil
	}
	values, ok := ctx.Value(stackKey{}).([]string)
	if !ok || len(values) == 0 {
		return nil
	}
	return append([]string(nil), values...)
}

// AppendStackValues 返回追加并清理后的 NDC 栈快照。
func AppendStackValues(dst []string, values ...string) []string {
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

// StackFromAttrs 从事件属性中提取 NDC 栈。
func StackFromAttrs(attrs []slog.Attr) []string {
	for index := len(attrs) - 1; index >= 0; index-- {
		attr := attrs[index]
		if attr.Key != StackAttrKey && attr.Key != "contextStack" && attr.Key != "ndc" {
			continue
		}
		return stackFromValue(attr.Value)
	}
	return nil
}

func stackFromValue(value slog.Value) []string {
	value = value.Resolve()
	switch typed := value.Any().(type) {
	case []string:
		return AppendStackValues(nil, typed...)
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			values = append(values, strings.TrimSpace(strings.Trim(fmt.Sprint(item), "[]")))
		}
		return AppendStackValues(nil, values...)
	default:
		text := strings.TrimSpace(logvalue.String(value))
		if text == "" {
			return nil
		}
		return AppendStackValues(nil, strings.Fields(text)...)
	}
}

// StackString 把 NDC 栈编码为 pattern 布局使用的字符串。
func StackString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return strings.Join(values, " ")
}

// MarkerFromAttrs 从事件属性中提取 marker。
func MarkerFromAttrs(attrs []slog.Attr) *Marker {
	for index := len(attrs) - 1; index >= 0; index-- {
		attr := attrs[index]
		if attr.Key != MarkerAttrKey && attr.Key != "marker" {
			continue
		}
		if marker, ok := markerFromValue(attr.Value); ok {
			return MarkerPointer(marker)
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

// ThreadNameFromAttrs 从事件属性中提取逻辑线程名。
func ThreadNameFromAttrs(attrs []slog.Attr) string {
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

// Empty 判断 context 是否没有 goark-log 运行期上下文。
func Empty(ctx context.Context) bool {
	if ctx == nil {
		return true
	}
	if attrs, ok := ctx.Value(attrsKey{}).([]slog.Attr); ok && len(attrs) > 0 {
		return false
	}
	if values, ok := ctx.Value(stackKey{}).([]string); ok && len(values) > 0 {
		return false
	}
	if marker, ok := ctx.Value(markerKey{}).(Marker); ok && marker.Name != "" {
		return false
	}
	if name, ok := ctx.Value(threadNameKey{}).(string); ok && name != "" {
		return false
	}
	return true
}

func normalizeAttr(attr slog.Attr) slog.Attr {
	attr.Value = attr.Value.Resolve()
	return attr
}
