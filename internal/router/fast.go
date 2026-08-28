package router

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"goark.dev/log/internal/jsonappender"
	"goark.dev/log/internal/logcontext"
	"goark.dev/log/internal/logevent"
)

// DispatchAttrsFast 在无上下文合并、无过滤器、无 caller 时直写 JSON appender。
func DispatchAttrsFast(ctx context.Context, route Route, handlerAttrs []slog.Attr, groups []string, logger string, when time.Time, level slog.Level, message string, attrs []slog.Attr) (bool, error) {
	if !canDispatchAttrsFast(ctx, route, handlerAttrs, groups, attrs) {
		return false, nil
	}
	if logger == "" {
		logger = logevent.DefaultLoggerName
	}
	event := jsonappender.AttrEvent{
		Time:    when,
		Level:   level,
		Logger:  logger,
		Message: message,
		Attrs:   attrs,
	}
	var joined error
	wrote := false
	for _, control := range route.Appenders {
		appender, ok := control.appender.(jsonappender.AttrAppender)
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

// DispatchFixedAttrsFast 针对三个固定属性事件直写 JSON appender，避免属性切片分配。
func DispatchFixedAttrsFast(ctx context.Context, route Route, handlerAttrs []slog.Attr, groups []string, logger string, when time.Time, level slog.Level, message string, attr0 slog.Attr, attr1 slog.Attr, attr2 slog.Attr) (bool, error) {
	if !canDispatchFixedAttrsFast(ctx, route, handlerAttrs, groups, attr0, attr1, attr2) {
		return false, nil
	}
	if logger == "" {
		logger = logevent.DefaultLoggerName
	}
	event := jsonappender.FixedAttrEvent{
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
		appender, ok := control.appender.(jsonappender.FixedAttrAppender)
		if !ok {
			return false, nil
		}
		if control.level != nil && level < *control.level {
			continue
		}
		wrote = true
		if err := appender.AppendFixedAttrs(ctx, event); err != nil {
			joined = errors.Join(joined, err)
		}
	}
	return wrote || joined != nil, joined
}

func canDispatchAttrsFast(ctx context.Context, route Route, handlerAttrs []slog.Attr, groups []string, attrs []slog.Attr) bool {
	if route.IncludeLocation || len(route.Filters) != 0 || len(route.Appenders) == 0 {
		return false
	}
	if len(handlerAttrs) != 0 || len(groups) != 0 || !logcontext.Empty(ctx) {
		return false
	}
	if !logevent.AttrsCanShare(attrs) {
		return false
	}
	for _, control := range route.Appenders {
		if len(control.filters) != 0 {
			return false
		}
		if _, ok := control.appender.(jsonappender.AttrAppender); !ok {
			return false
		}
	}
	return true
}

func canDispatchFixedAttrsFast(ctx context.Context, route Route, handlerAttrs []slog.Attr, groups []string, attr0 slog.Attr, attr1 slog.Attr, attr2 slog.Attr) bool {
	if route.IncludeLocation || len(route.Filters) != 0 || len(route.Appenders) == 0 {
		return false
	}
	if len(handlerAttrs) != 0 || len(groups) != 0 || !logcontext.Empty(ctx) {
		return false
	}
	if !attrCanShare(attr0) || !attrCanShare(attr1) || !attrCanShare(attr2) {
		return false
	}
	for _, control := range route.Appenders {
		if len(control.filters) != 0 {
			return false
		}
		if _, ok := control.appender.(jsonappender.FixedAttrAppender); !ok {
			return false
		}
	}
	return true
}

func attrCanShare(attr slog.Attr) bool {
	if attr.Key == "" || attr.Key == logevent.LoggerNameKey {
		return false
	}
	return attr.Value.Kind() != slog.KindLogValuer
}
