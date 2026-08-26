package goarklog

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

func (h *Handler) dispatchAttrsFast(ctx context.Context, route route, logger string, when time.Time, level slog.Level, message string, attrs []slog.Attr) (bool, error) {
	if !canDispatchAttrsFast(ctx, route, h.attrs, h.groups, attrs) {
		return false, nil
	}
	if logger == "" {
		logger = defaultLoggerName
	}
	event := attrEvent{
		Time:    when,
		Level:   level,
		Logger:  logger,
		Message: message,
		Attrs:   attrs,
	}
	var joined error
	wrote := false
	for _, control := range route.Appenders {
		appender, ok := control.appender.(attrAppender)
		if !ok {
			return false, nil
		}
		if control.level != nil && level < *control.level {
			continue
		}
		wrote = true
		if err := appender.AppendAttrs(ctx, event); err != nil {
			joined = errors.Join(joined, err)
		}
	}
	return wrote || joined != nil, joined
}

func (h *Handler) dispatchFixedAttrsFast(ctx context.Context, route route, logger string, when time.Time, level slog.Level, message string, attr0 slog.Attr, attr1 slog.Attr, attr2 slog.Attr) (bool, error) {
	if !canDispatchFixedAttrsFast(ctx, route, h.attrs, h.groups, attr0, attr1, attr2) {
		return false, nil
	}
	if logger == "" {
		logger = defaultLoggerName
	}
	event := fixedAttrEvent{
		Time:    when,
		Level:   level,
		Logger:  logger,
		Message: message,
		Attrs:   [3]slog.Attr{attr0, attr1, attr2},
		Count:   3,
	}
	var joined error
	wrote := false
	for _, control := range route.Appenders {
		appender, ok := control.appender.(*JSONAppender)
		if !ok {
			return false, nil
		}
		if control.level != nil && level < *control.level {
			continue
		}
		wrote = true
		if err := appender.appendFixedAttrs(ctx, event); err != nil {
			joined = errors.Join(joined, err)
		}
	}
	return wrote || joined != nil, joined
}

func canDispatchAttrsFast(ctx context.Context, route route, handlerAttrs []slog.Attr, groups []string, attrs []slog.Attr) bool {
	if len(route.Filters) != 0 || len(route.Appenders) == 0 {
		return false
	}
	if len(handlerAttrs) != 0 || len(groups) != 0 || !contextLogStateEmpty(ctx) {
		return false
	}
	if !attrsCanShare(attrs) {
		return false
	}
	for _, control := range route.Appenders {
		if len(control.filters) != 0 {
			return false
		}
		if _, ok := control.appender.(attrAppender); !ok {
			return false
		}
	}
	return true
}

func canDispatchFixedAttrsFast(ctx context.Context, route route, handlerAttrs []slog.Attr, groups []string, attr0 slog.Attr, attr1 slog.Attr, attr2 slog.Attr) bool {
	if len(route.Filters) != 0 || len(route.Appenders) == 0 {
		return false
	}
	if len(handlerAttrs) != 0 || len(groups) != 0 || !contextLogStateEmpty(ctx) {
		return false
	}
	if !attrCanShare(attr0) || !attrCanShare(attr1) || !attrCanShare(attr2) {
		return false
	}
	for _, control := range route.Appenders {
		if len(control.filters) != 0 {
			return false
		}
		if _, ok := control.appender.(*JSONAppender); !ok {
			return false
		}
	}
	return true
}

func attrCanShare(attr slog.Attr) bool {
	if attr.Key == "" || attr.Key == loggerNameKey {
		return false
	}
	return attr.Value.Kind() != slog.KindLogValuer
}

func contextLogStateEmpty(ctx context.Context) bool {
	if ctx == nil {
		return true
	}
	if attrs, ok := ctx.Value(contextAttrsKey{}).([]slog.Attr); ok && len(attrs) > 0 {
		return false
	}
	if values, ok := ctx.Value(contextStackKey{}).([]string); ok && len(values) > 0 {
		return false
	}
	if marker, ok := ctx.Value(markerContextKey{}).(Marker); ok && marker.Name != "" {
		return false
	}
	if name, ok := ctx.Value(threadNameContextKey{}).(string); ok && name != "" {
		return false
	}
	return true
}
