package nativelogger

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
	"time"

	configlevel "goark.dev/log/internal/level"
	"goark.dev/log/internal/logevent"
	logmessage "goark.dev/log/internal/message"
)

// Handler 是原生 Logger 依赖的最小热路径端口。
type Handler interface {
	Enabled(ctx context.Context, logger string, level slog.Level) bool
	IncludeCaller(logger string) bool
	LogAttrs(
		ctx context.Context,
		logger string,
		handlerAttrs []slog.Attr,
		groups []string,
		when time.Time,
		level slog.Level,
		message string,
		pc uintptr,
		attrs []slog.Attr,
	) error
	Log3Attrs(
		ctx context.Context,
		logger string,
		handlerAttrs []slog.Attr,
		groups []string,
		when time.Time,
		level slog.Level,
		message string,
		pc uintptr,
		attr0 slog.Attr,
		attr1 slog.Attr,
		attr2 slog.Attr,
	) error
	SlogHandler() slog.Handler
}

// Message 表示可被日志事件快照化的消息对象。
type Message = logmessage.Message

// AttributedMessage 表示会同时贡献结构化属性的消息对象。
type AttributedMessage = logmessage.AttributedMessage

// MessageFactory 创建日志消息对象。
type MessageFactory = logmessage.MessageFactory

// Logger 是 goark-log 的低分配原生日志入口。
type Logger struct {
	handler        Handler
	name           string
	attrs          []slog.Attr
	groups         []string
	includeCaller  bool
	messageFactory MessageFactory
}

// Option 调整原生 Logger。
type Option func(*Logger)

// WithCaller 设置是否采集调用位置。
func WithCaller(enabled bool) Option {
	return func(logger *Logger) {
		logger.includeCaller = enabled
	}
}

// WithMessageFactory 设置参数化消息工厂。
func WithMessageFactory(factory MessageFactory) Option {
	return func(logger *Logger) {
		if factory != nil {
			logger.messageFactory = factory
		}
	}
}

// New 基于 Handler 端口创建低分配命名 logger。
func New(handler Handler, name string, options ...Option) (*Logger, error) {
	if handler == nil {
		return nil, fmt.Errorf("goark-log: native logger handler is nil")
	}
	logger := &Logger{
		handler:        handler,
		name:           strings.TrimSpace(name),
		messageFactory: logmessage.ParameterizedMessageFactory{},
	}
	if logger.name == "" {
		logger.name = logevent.DefaultLoggerName
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
	handler := l.handler.SlogHandler()
	if handler == nil {
		return slog.Default()
	}
	logger := slog.New(handler).With(logevent.LoggerNameKey, l.name)
	if len(l.attrs) > 0 {
		values := make([]any, 0, len(l.attrs))
		for _, attr := range l.attrs {
			values = append(values, attr)
		}
		logger = logger.With(values...)
	}
	for _, group := range l.groups {
		logger = logger.WithGroup(group)
	}
	return logger
}

// Name 返回 logger 名称。
func (l *Logger) Name() string {
	if l == nil || strings.TrimSpace(l.name) == "" {
		return logevent.DefaultLoggerName
	}
	return l.name
}

// Enabled 判断指定级别当前是否会进入日志管线。
func (l *Logger) Enabled(ctx context.Context, level slog.Level) bool {
	if l == nil || l.handler == nil {
		return false
	}
	return l.handler.Enabled(ctx, l.Name(), level)
}

// WithAttrs 返回绑定额外属性的新 Logger。
func (l *Logger) WithAttrs(attrs ...slog.Attr) *Logger {
	if l == nil {
		return nil
	}
	next := l.clone()
	for _, attr := range attrs {
		attr = logevent.NormalizeAttr(attr)
		if attr.Key == "" || attr.Key == logevent.LoggerNameKey {
			continue
		}
		next.attrs = logevent.AppendAttr(next.attrs, next.groups, attr)
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
func (l *Logger) LogAttrs(
	ctx context.Context,
	level slog.Level,
	message string,
	attrs ...slog.Attr,
) error {
	return l.logAttrs(ctx, level, message, attrs, 2)
}

// LogAttrs3 使用三个固定属性写出事件，避免极热路径的 variadic slice 分配。
func (l *Logger) LogAttrs3(
	ctx context.Context,
	level slog.Level,
	message string,
	attr0 slog.Attr,
	attr1 slog.Attr,
	attr2 slog.Attr,
) error {
	if l == nil || l.handler == nil {
		return fmt.Errorf("goark-log: native logger is nil")
	}
	pc := uintptr(0)
	if l.includeCaller || l.handler.IncludeCaller(l.Name()) {
		pc = CallerPC(2)
	}
	return l.handler.Log3Attrs(
		ctx,
		l.Name(),
		l.attrs,
		l.groups,
		time.Now(),
		level,
		message,
		pc,
		attr0,
		attr1,
		attr2,
	)
}

func (l *Logger) logAttrs(
	ctx context.Context,
	level slog.Level,
	message string,
	attrs []slog.Attr,
	callerSkip int,
) error {
	if l == nil || l.handler == nil {
		return fmt.Errorf("goark-log: native logger is nil")
	}
	pc := uintptr(0)
	if l.includeCaller || l.handler.IncludeCaller(l.Name()) {
		pc = CallerPC(callerSkip)
	}
	return l.handler.LogAttrs(
		ctx,
		l.Name(),
		l.attrs,
		l.groups,
		time.Now(),
		level,
		message,
		pc,
		attrs,
	)
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

// Fatal 写出 FATAL 级别日志。
func (l *Logger) Fatal(message string, attrs ...slog.Attr) error {
	return l.logAttrs(context.Background(), configlevel.Fatal, message, attrs, 2)
}

// FatalContext 写出带 context 的 FATAL 级别日志。
func (l *Logger) FatalContext(ctx context.Context, message string, attrs ...slog.Attr) error {
	return l.logAttrs(ctx, configlevel.Fatal, message, attrs, 2)
}

func (l *Logger) clone() *Logger {
	next := *l
	next.attrs = append([]slog.Attr(nil), l.attrs...)
	next.groups = append([]string(nil), l.groups...)
	return &next
}

// CallerPC 返回指定 skip 对应的调用位置程序计数器。
func CallerPC(skip int) uintptr {
	var pcs [1]uintptr
	if runtime.Callers(skip+2, pcs[:]) == 0 {
		return 0
	}
	return pcs[0]
}
