package goarklog

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
	"time"

	logmessage "goark.dev/log/internal/message"
)

const logBuilderInlineAttrs = 8

const (
	// StructuredDataIDAttrKey 是结构化消息 ID 的标准属性键。
	StructuredDataIDAttrKey = logmessage.StructuredDataIDAttrKey
	// StructuredDataTypeAttrKey 是结构化消息类型的标准属性键。
	StructuredDataTypeAttrKey = logmessage.StructuredDataTypeAttrKey
)

// Message 表示可被日志事件快照化的消息对象。
type Message = logmessage.Message

// AttributedMessage 表示会同时贡献结构化属性的消息对象。
type AttributedMessage = logmessage.AttributedMessage

// MessageFactory 创建日志消息对象。
type MessageFactory = logmessage.MessageFactory

// MessageFactoryFunc 把函数适配为 MessageFactory。
type MessageFactoryFunc = logmessage.MessageFactoryFunc

// ParameterizedMessageFactory 创建 {} 占位符参数化消息。
type ParameterizedMessageFactory = logmessage.ParameterizedMessageFactory

// SimpleMessageFactory 忽略参数并创建普通字符串消息。
type SimpleMessageFactory = logmessage.SimpleMessageFactory

// SimpleMessage 是不可变字符串消息。
type SimpleMessage = logmessage.SimpleMessage

// ParameterizedMessage 使用 {} 占位符格式化消息。
type ParameterizedMessage = logmessage.ParameterizedMessage

// MapMessage 用属性集合表达消息，适合结构化日志。
type MapMessage = logmessage.MapMessage

// StructuredDataMessage 表示 RFC5424 风格的结构化消息。
type StructuredDataMessage = logmessage.StructuredDataMessage

// NewSimpleMessage 创建字符串消息。
func NewSimpleMessage(text string) SimpleMessage {
	return logmessage.NewSimpleMessage(text)
}

// NewParameterizedMessage 创建参数化消息，参数会被快照复制。
func NewParameterizedMessage(pattern string, args ...any) ParameterizedMessage {
	return logmessage.NewParameterizedMessage(pattern, args...)
}

// NewMapMessage 创建结构化属性消息。
func NewMapMessage(attrs ...slog.Attr) MapMessage {
	return logmessage.NewMapMessage(attrs...)
}

// NewStructuredDataMessage 创建结构化数据消息。
func NewStructuredDataMessage(id string, msgType string, message string, attrs ...slog.Attr) StructuredDataMessage {
	return logmessage.NewStructuredDataMessage(id, msgType, message, attrs...)
}

// Logger 是 goark-log 的低分配原生日志入口。
type Logger struct {
	handler        *Handler
	name           string
	attrs          []slog.Attr
	groups         []string
	includeCaller  bool
	messageFactory MessageFactory
}

// LoggerOption 调整原生 Logger。
type LoggerOption func(*Logger)

// WithLoggerCaller 设置是否采集调用位置。
func WithLoggerCaller(enabled bool) LoggerOption {
	return func(logger *Logger) {
		logger.includeCaller = enabled
	}
}

// WithLoggerMessageFactory 设置参数化消息工厂。
func WithLoggerMessageFactory(factory MessageFactory) LoggerOption {
	return func(logger *Logger) {
		if factory != nil {
			logger.messageFactory = factory
		}
	}
}

// NewNativeLogger 基于 Handler 创建低分配命名 logger。
func NewNativeLogger(handler *Handler, name string, options ...LoggerOption) (*Logger, error) {
	if handler == nil {
		return nil, fmt.Errorf("goark-log: native logger handler is nil")
	}
	logger := &Logger{
		handler:        handler,
		name:           strings.TrimSpace(name),
		messageFactory: ParameterizedMessageFactory{},
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

// LogAttrs3 使用三个固定属性写出事件，避免极热路径的 variadic slice 分配。
func (l *Logger) LogAttrs3(ctx context.Context, level slog.Level, message string, attr0 slog.Attr, attr1 slog.Attr, attr2 slog.Attr) error {
	if l == nil || l.handler == nil {
		return fmt.Errorf("goark-log: native logger is nil")
	}
	pc := uintptr(0)
	if l.includeCaller || l.handler.asyncIncludeLocation() || l.handler.routeIncludeLocation(l.Name()) {
		pc = callerPC(2)
	}
	return l.handler.log3Attrs(ctx, l.Name(), l.attrs, l.groups, time.Now(), level, message, pc, attr0, attr1, attr2)
}

func (l *Logger) logAttrs(ctx context.Context, level slog.Level, message string, attrs []slog.Attr, callerSkip int) error {
	if l == nil || l.handler == nil {
		return fmt.Errorf("goark-log: native logger is nil")
	}
	pc := uintptr(0)
	if l.includeCaller || l.handler.asyncIncludeLocation() || l.handler.routeIncludeLocation(l.Name()) {
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

// Fatal 写出 FATAL 级别日志。
func (l *Logger) Fatal(message string, attrs ...slog.Attr) error {
	return l.logAttrs(context.Background(), LevelFatal, message, attrs, 2)
}

// FatalContext 写出带 context 的 FATAL 级别日志。
func (l *Logger) FatalContext(ctx context.Context, message string, attrs ...slog.Attr) error {
	return l.logAttrs(ctx, LevelFatal, message, attrs, 2)
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

// LogBuilder 是低分配链式事件构造器。
type LogBuilder struct {
	logger    *Logger
	ctx       context.Context
	level     slog.Level
	inline    [logBuilderInlineAttrs]slog.Attr
	attrCount int
	attrs     []slog.Attr
	groups    []string
	enabled   bool
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

// AtFatal 创建 FATAL 级别事件构造器。
func (l *Logger) AtFatal() LogBuilder {
	return l.At(LevelFatal)
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
	if len(b.groups) > 0 {
		attr.Key = groupKey(b.groups, attr.Key)
	}
	return b.appendOneAttr(attr)
}

// WithAttrs 追加结构化属性集合。
func (b LogBuilder) WithAttrs(attrs ...slog.Attr) LogBuilder {
	if !b.enabled || len(attrs) == 0 {
		return b
	}
	for _, attr := range attrs {
		attr = normalizeAttr(attr)
		if attr.Key == "" || attr.Key == loggerNameKey {
			continue
		}
		if len(b.groups) > 0 {
			attr.Key = groupKey(b.groups, attr.Key)
		}
		b = b.appendOneAttr(attr)
	}
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
	factory := MessageFactory(ParameterizedMessageFactory{})
	if b.logger != nil && b.logger.messageFactory != nil {
		factory = b.logger.messageFactory
	}
	return b.LogMessage(factory.NewMessage(pattern, args...))
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
	attrs := b.attrSlice()
	if attributed, ok := message.(AttributedMessage); ok {
		attrs = append(append([]slog.Attr(nil), attrs...), attributed.Attrs()...)
	}
	ctx := b.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	return b.logger.logAttrs(ctx, b.level, message.String(), attrs, 3)
}

func (b LogBuilder) appendOneAttr(attr slog.Attr) LogBuilder {
	if len(b.attrs) > 0 {
		next := make([]slog.Attr, 0, len(b.attrs)+1)
		next = append(next, b.attrs...)
		next = append(next, attr)
		b.attrs = next
		b.attrCount = len(next)
		return b
	}
	if b.attrCount < len(b.inline) {
		b.inline[b.attrCount] = attr
		b.attrCount++
		return b
	}
	next := make([]slog.Attr, 0, b.attrCount+1)
	next = append(next, b.inline[:b.attrCount]...)
	next = append(next, attr)
	b.attrs = next
	b.attrCount = len(next)
	return b
}

func (b LogBuilder) attrSlice() []slog.Attr {
	if len(b.attrs) > 0 {
		return b.attrs
	}
	return b.inline[:b.attrCount]
}
