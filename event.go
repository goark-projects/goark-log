package goarklog

import (
	"context"
	"log/slog"
	"time"
)

const loggerNameKey = "goark.logger"

const defaultLoggerName = "goark"

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
