package goarklog

import (
	"log/slog"
	"time"
)

const loggerNameKey = "goark.logger"

const defaultLoggerName = "goark"

// Event 是一次已经快照化的日志事件。
type Event struct {
	Time    time.Time
	Level   slog.Level
	Message string
	Logger  string
	PC      uintptr
	Attrs   []slog.Attr
}

// Attr 按键查找事件属性。
func (e Event) Attr(key string) (slog.Value, bool) {
	for _, attr := range e.Attrs {
		if attr.Key == key {
			return attr.Value, true
		}
	}
	return slog.Value{}, false
}

func newEvent(logger string, handlerAttrs []slog.Attr, groups []string, record slog.Record) Event {
	if logger == "" {
		logger = defaultLoggerName
	}
	attrs := make([]slog.Attr, 0, len(handlerAttrs)+record.NumAttrs())
	attrs = appendAttrs(attrs, nil, handlerAttrs)
	record.Attrs(func(attr slog.Attr) bool {
		attrs = appendAttr(attrs, groups, attr)
		return true
	})
	return Event{
		Time:    record.Time,
		Level:   record.Level,
		Message: record.Message,
		Logger:  logger,
		PC:      record.PC,
		Attrs:   attrs,
	}
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
