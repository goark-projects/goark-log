package goarklog

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
	"time"
)

// Logger 是 goark-log 的低分配原生日志入口。
type Logger struct {
	handler       *Handler
	name          string
	attrs         []slog.Attr
	groups        []string
	includeCaller bool
}

// LoggerOption 调整原生 Logger。
type LoggerOption func(*Logger)

// WithLoggerCaller 设置是否采集调用位置。
func WithLoggerCaller(enabled bool) LoggerOption {
	return func(logger *Logger) {
		logger.includeCaller = enabled
	}
}

// NewNativeLogger 基于 Handler 创建低分配命名 logger。
func NewNativeLogger(handler *Handler, name string, options ...LoggerOption) (*Logger, error) {
	if handler == nil {
		return nil, fmt.Errorf("goark-log: native logger handler is nil")
	}
	logger := &Logger{
		handler: handler,
		name:    strings.TrimSpace(name),
	}
	if logger.name == "" {
		logger.name = defaultLoggerName
	}
	for _, option := range options {
		if option != nil {
			option(logger)
		}
	}
	return logger, nil
}

// Slog 返回同名 slog.Logger，便于和标准库生态互通。
func (l *Logger) Slog() *slog.Logger {
	if l == nil || l.handler == nil {
		return slog.Default()
	}
	logger := slog.New(l.handler).With(loggerNameKey, l.name)
	if len(l.attrs) > 0 {
		values := make([]any, 0, len(l.attrs)*2)
		for _, attr := range l.attrs {
			values = append(values, attr.Key, attr.Value)
		}
		logger = logger.With(values...)
	}
	return logger
}

// Name 返回 logger 名称。
func (l *Logger) Name() string {
	if l == nil || strings.TrimSpace(l.name) == "" {
		return defaultLoggerName
	}
	return l.name
}

// Enabled 判断指定级别当前是否会进入日志管线。
func (l *Logger) Enabled(ctx context.Context, level slog.Level) bool {
	if l == nil || l.handler == nil {
		return false
	}
	return l.handler.enabled(l.Name(), level)
}

// WithAttrs 返回绑定额外属性的新 Logger。
func (l *Logger) WithAttrs(attrs ...slog.Attr) *Logger {
	if l == nil {
		return nil
	}
	next := l.clone()
	for _, attr := range attrs {
		attr = normalizeAttr(attr)
		if attr.Key == "" || attr.Key == loggerNameKey {
			continue
		}
		next.attrs = appendAttr(next.attrs, next.groups, attr)
	}
	return next
}

// WithGroup 返回绑定属性分组的新 Logger。
func (l *Logger) WithGroup(name string) *Logger {
	if l == nil {
		return nil
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return l
	}
	next := l.clone()
	next.groups = append(next.groups, name)
	return next
}

// LogAttrs 使用 slog.Attr 直写日志事件。
func (l *Logger) LogAttrs(ctx context.Context, level slog.Level, message string, attrs ...slog.Attr) error {
	return l.logAttrs(ctx, level, message, attrs, 2)
}

func (l *Logger) logAttrs(ctx context.Context, level slog.Level, message string, attrs []slog.Attr, callerSkip int) error {
	if l == nil || l.handler == nil {
		return fmt.Errorf("goark-log: native logger is nil")
	}
	pc := uintptr(0)
	if l.includeCaller || l.handler.asyncIncludeLocation() {
		pc = callerPC(callerSkip)
	}
	return l.handler.logAttrs(ctx, l.Name(), l.attrs, l.groups, time.Now(), level, message, pc, attrs)
}

// Debug 写出 DEBUG 级别日志。
func (l *Logger) Debug(message string, attrs ...slog.Attr) error {
	return l.logAttrs(context.Background(), slog.LevelDebug, message, attrs, 2)
}

// DebugContext 写出带 context 的 DEBUG 级别日志。
func (l *Logger) DebugContext(ctx context.Context, message string, attrs ...slog.Attr) error {
	return l.logAttrs(ctx, slog.LevelDebug, message, attrs, 2)
}

// Info 写出 INFO 级别日志。
func (l *Logger) Info(message string, attrs ...slog.Attr) error {
	return l.logAttrs(context.Background(), slog.LevelInfo, message, attrs, 2)
}

// InfoContext 写出带 context 的 INFO 级别日志。
func (l *Logger) InfoContext(ctx context.Context, message string, attrs ...slog.Attr) error {
	return l.logAttrs(ctx, slog.LevelInfo, message, attrs, 2)
}

// Warn 写出 WARN 级别日志。
func (l *Logger) Warn(message string, attrs ...slog.Attr) error {
	return l.logAttrs(context.Background(), slog.LevelWarn, message, attrs, 2)
}

// WarnContext 写出带 context 的 WARN 级别日志。
func (l *Logger) WarnContext(ctx context.Context, message string, attrs ...slog.Attr) error {
	return l.logAttrs(ctx, slog.LevelWarn, message, attrs, 2)
}

// Error 写出 ERROR 级别日志。
func (l *Logger) Error(message string, attrs ...slog.Attr) error {
	return l.logAttrs(context.Background(), slog.LevelError, message, attrs, 2)
}

// ErrorContext 写出带 context 的 ERROR 级别日志。
func (l *Logger) ErrorContext(ctx context.Context, message string, attrs ...slog.Attr) error {
	return l.logAttrs(ctx, slog.LevelError, message, attrs, 2)
}

func (l *Logger) clone() *Logger {
	next := *l
	next.attrs = append([]slog.Attr(nil), l.attrs...)
	next.groups = append([]string(nil), l.groups...)
	return &next
}

func callerPC(skip int) uintptr {
	var pcs [1]uintptr
	if runtime.Callers(skip+2, pcs[:]) == 0 {
		return 0
	}
	return pcs[0]
}
