package goarklog

import (
	"context"
	"log/slog"
	"strings"

	"goark.dev/log/internal/logvalue"
)

const (
	// MarkerAttrKey 是 goark-log 标准 marker 属性键。
	MarkerAttrKey = "goark.marker"
)

type markerContextKey struct{}

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
