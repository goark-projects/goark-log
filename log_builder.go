package goarklog

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// LogBuilder 是 Log4j2 风格的事件构造器。
type LogBuilder struct {
	logger  *Logger
	ctx     context.Context
	level   slog.Level
	attrs   []slog.Attr
	groups  []string
	enabled bool
}

// At 创建指定级别的事件构造器。
func (l *Logger) At(level slog.Level) LogBuilder {
	return LogBuilder{
		logger:  l,
		level:   level,
		enabled: l != nil && l.Enabled(context.Background(), level),
	}
}

// AtTrace 创建 TRACE 级别事件构造器。
func (l *Logger) AtTrace() LogBuilder {
	return l.At(LevelTrace)
}

// AtDebug 创建 DEBUG 级别事件构造器。
func (l *Logger) AtDebug() LogBuilder {
	return l.At(slog.LevelDebug)
}

// AtInfo 创建 INFO 级别事件构造器。
func (l *Logger) AtInfo() LogBuilder {
	return l.At(slog.LevelInfo)
}

// AtWarn 创建 WARN 级别事件构造器。
func (l *Logger) AtWarn() LogBuilder {
	return l.At(slog.LevelWarn)
}

// AtError 创建 ERROR 级别事件构造器。
func (l *Logger) AtError() LogBuilder {
	return l.At(slog.LevelError)
}

// Enabled 判断事件构造器是否会写出日志。
func (b LogBuilder) Enabled() bool {
	return b.enabled
}

// WithContext 设置事件 context。
func (b LogBuilder) WithContext(ctx context.Context) LogBuilder {
	if ctx == nil {
		return b
	}
	b.ctx = ctx
	return b
}

// WithGroup 为后续属性设置分组前缀。
func (b LogBuilder) WithGroup(name string) LogBuilder {
	if !b.enabled {
		return b
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return b
	}
	b.groups = append(append([]string(nil), b.groups...), name)
	return b
}

// WithAttr 追加结构化属性。
func (b LogBuilder) WithAttr(attr slog.Attr) LogBuilder {
	if !b.enabled {
		return b
	}
	attr = normalizeAttr(attr)
	if attr.Key == "" || attr.Key == loggerNameKey {
		return b
	}
	b.attrs = append(append([]slog.Attr(nil), b.attrs...), appendAttr(nil, b.groups, attr)...)
	return b
}

// WithAttrs 追加结构化属性集合。
func (b LogBuilder) WithAttrs(attrs ...slog.Attr) LogBuilder {
	if !b.enabled || len(attrs) == 0 {
		return b
	}
	next := make([]slog.Attr, 0, len(b.attrs)+len(attrs))
	next = append(next, b.attrs...)
	for _, attr := range attrs {
		attr = normalizeAttr(attr)
		if attr.Key == "" || attr.Key == loggerNameKey {
			continue
		}
		next = appendAttr(next, b.groups, attr)
	}
	b.attrs = next
	return b
}

// WithString 追加字符串属性。
func (b LogBuilder) WithString(key string, value string) LogBuilder {
	return b.WithAttr(slog.String(key, value))
}

// WithInt 追加整数属性。
func (b LogBuilder) WithInt(key string, value int) LogBuilder {
	return b.WithAttr(slog.Int(key, value))
}

// WithBool 追加布尔属性。
func (b LogBuilder) WithBool(key string, value bool) LogBuilder {
	return b.WithAttr(slog.Bool(key, value))
}

// WithAny 追加任意值属性。
func (b LogBuilder) WithAny(key string, value any) LogBuilder {
	return b.WithAttr(slog.Any(key, value))
}

// WithMarker 设置事件 marker。
func (b LogBuilder) WithMarker(marker Marker) LogBuilder {
	if marker.Name == "" {
		return b
	}
	return b.WithAttr(MarkerAttr(marker))
}

// WithError 设置事件异常快照。
func (b LogBuilder) WithError(err error) LogBuilder {
	if err == nil {
		return b
	}
	return b.WithAttr(ThrowableAttr(err))
}

// WithThrowable 是 WithError 的语义别名。
func (b LogBuilder) WithThrowable(err error) LogBuilder {
	return b.WithError(err)
}

// WithErrorStack 设置事件异常快照并采集当前调用栈。
func (b LogBuilder) WithErrorStack(err error) LogBuilder {
	if err == nil {
		return b
	}
	return b.WithAttr(ThrowableWithStackAttr(err))
}

// Log 写出字符串消息。
func (b LogBuilder) Log(message string) error {
	return b.LogMessage(SimpleMessage(message))
}

// Logf 使用 {} 占位符写出参数化消息。
func (b LogBuilder) Logf(pattern string, args ...any) error {
	return b.LogMessage(NewParameterizedMessage(pattern, args...))
}

// LogMessage 写出 Message 对象。
func (b LogBuilder) LogMessage(message Message) error {
	if b.logger == nil || b.logger.handler == nil {
		return fmt.Errorf("goark-log: log builder logger is nil")
	}
	if !b.enabled {
		return nil
	}
	if message == nil {
		message = SimpleMessage("")
	}
	attrs := b.attrs
	if attributed, ok := message.(AttributedMessage); ok {
		attrs = append(append([]slog.Attr(nil), attrs...), attributed.Attrs()...)
	}
	ctx := b.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	return b.logger.LogAttrs(ctx, b.level, message.String(), attrs...)
}
