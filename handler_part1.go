package log

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	internalnativelogger "goark.dev/log/internal/nativelogger"
	internalrouter "goark.dev/log/internal/router"
)

func (h *Handler) logAttrs(
	ctx context.Context,
	logger string,
	handlerAttrs []slog.Attr,
	groups []string,
	when time.Time,
	level slog.Level,
	message string,
	pc uintptr,
	attrs []slog.Attr,
) error {
	if h == nil || h.router == nil {
		return fmt.Errorf("goark-log: handler is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	plan := h.router.Plan(logger)
	if len(plan.GlobalFilters) == 0 {
		if level < plan.Route.Level {
			return nil
		}
		if h.async == nil && pc == 0 {
			handled, err := internalrouter.DispatchAttrsFast(
				ctx, plan.Route, handlerAttrs, groups, logger, when, level, message, attrs,
			)
			if handled {
				return err
			}
		}
	}
	event := newEventFromAttrs(
		ctx,
		logger,
		handlerAttrs,
		groups,
		when,
		level,
		message,
		pc,
		attrs,
		h.async != nil,
	)
	levelAccepted, denied := applyGlobalFilters(ctx, plan.GlobalFilters, event)
	if denied {
		return nil
	}
	if !levelAccepted && level < plan.Route.Level {
		return nil
	}
	if h.async != nil {
		return h.async.Append(ctx, event, levelAccepted)
	}
	return h.dispatchRoute(ctx, plan.Route, event)
}

func (h nativeHandler) Log3Attrs(
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
) error {
	return h.handler.log3Attrs(
		ctx,
		logger,
		handlerAttrs,
		groups,
		when,
		level,
		message,
		pc,
		attr0,
		attr1,
		attr2,
	)
}

func (h *Handler) dispatchRoute(
	ctx context.Context,
	route internalrouter.Route,
	event Event,
) error {
	if applyFilters(ctx, route.Filters, event) == FilterDeny {
		return nil
	}
	var joined error
	for _, appender := range route.Appenders {
		_, err := appender.AppendResult(ctx, event)
		if err != nil {
			joined = errors.Join(joined, err)
			continue
		}
	}
	return joined
}

func applyGlobalFilters(
	ctx context.Context,
	filters []Filter,
	event Event,
) (levelAccepted bool, denied bool) {
	switch applyFilters(ctx, filters, event) {
	case FilterDeny:
		return false, true
	case FilterAccept:
		return true, false
	default:
		return false, false
	}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	if h == nil {
		return h
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return h
	}
	next := h.clone()
	next.groups = append(next.groups, name)
	return next
}

func (h *Handler) enabled(name string, level slog.Level) bool {
	if h == nil || h.router == nil {
		return level >= slog.LevelInfo
	}
	plan := h.router.Plan(name)
	if len(plan.GlobalFilters) > 0 {
		return true
	}
	return level >= plan.Route.Level
}

// New 创建默认命名 logger 和对应 Handler。
func New(options Options) (*slog.Logger, *Handler, error) {
	handler, err := NewHandler(options)
	if err != nil {
		return nil, nil, err
	}
	return NewLogger(handler, defaultLoggerName), handler, nil
}

// WithName 返回带有 goark-log logger 名称的 logger。
func WithName(logger *slog.Logger, name string) *slog.Logger {
	if logger == nil {
		logger = slog.Default()
	}
	return logger.With(loggerNameKey, name)
}

// NewNativeLogger 基于 Handler 创建低分配命名 logger。
func NewNativeLogger(handler *Handler, name string, options ...LoggerOption) (*Logger, error) {
	if handler == nil {
		return nil, fmt.Errorf("goark-log: native logger handler is nil")
	}
	return internalnativelogger.New(nativeHandler{handler: handler}, name, options...)
}

func (h *Handler) routeIncludeLocation(name string) bool {
	if h == nil || h.router == nil {
		return false
	}
	return h.router.IncludeLocation(name)
}

// NewDefault 创建默认 stdout INFO logger。
func NewDefault() (*slog.Logger, *Handler) {
	handler := NewDefaultHandler()
	return NewLogger(handler, defaultLoggerName), handler
}

// WithLoggerMessageFactory 设置参数化消息工厂。
func WithLoggerMessageFactory(factory MessageFactory) LoggerOption {
	return internalnativelogger.WithMessageFactory(factory)
}

func (h *Handler) asyncIncludeLocation() bool {
	return h != nil && h.async != nil && h.async.IncludeLocation()
}

func (h nativeHandler) Enabled(_ context.Context, logger string, level slog.Level) bool {
	return h.handler.enabled(logger, level)
}

// RootLogger 描述根 logger。
type RootLogger = internalrouter.RootLogger

// LoggerRule 描述命名 logger 的级别和输出路由。
type LoggerRule = internalrouter.LoggerRule

// Logger 是 goark-log 的低分配原生日志入口。
type Logger = internalnativelogger.Logger

// LogBuilder 是低分配链式事件构造器。
type LogBuilder = internalnativelogger.LogBuilder

var _ slog.Handler = (*Handler)(nil)
