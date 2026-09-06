package log

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	internalasynclogger "goark.dev/log/internal/asynclogger"
	internalnativelogger "goark.dev/log/internal/nativelogger"
	internalrouter "goark.dev/log/internal/router"
)

func (h *Handler) log3Attrs(
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
			handled, err := internalrouter.DispatchFixedAttrsFast(
				ctx, plan.Route, handlerAttrs, groups, logger, when, level, message,
				attr0, attr1, attr2,
			)
			if handled {
				return err
			}
		}
	}
	attrs := []slog.Attr{attr0, attr1, attr2}
	return h.logAttrs(ctx, logger, handlerAttrs, groups, when, level, message, pc, attrs)
}

func (h *Handler) Handle(ctx context.Context, record slog.Record) error {
	if h == nil || h.router == nil {
		return fmt.Errorf("goark-log: handler is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	plan := h.router.Plan(h.name)
	if len(plan.GlobalFilters) == 0 {
		if record.Level < plan.Route.Level {
			return nil
		}
		event := newEvent(ctx, h.name, h.attrs, h.groups, record)
		if h.async != nil {
			return h.async.Append(ctx, event, false)
		}
		return h.dispatchRoute(ctx, plan.Route, event)
	}
	event := newEvent(ctx, h.name, h.attrs, h.groups, record)
	levelAccepted, denied := applyGlobalFilters(ctx, plan.GlobalFilters, event)
	if denied {
		return nil
	}
	if !levelAccepted && record.Level < plan.Route.Level {
		return nil
	}
	if h.async != nil {
		return h.async.Append(ctx, event, levelAccepted)
	}
	return h.dispatchRoute(ctx, plan.Route, event)
}

// NewHandler 创建 slog.Handler。
func NewHandler(options Options) (*Handler, error) {
	options = defaultRuntimeOptions(options)
	router, err := internalrouter.New(toRouterOptions(options))
	if err != nil {
		return nil, err
	}
	handler := &Handler{router: router, name: defaultLoggerName}
	if options.Async.Enabled {
		async, err := internalasynclogger.New(handler.dispatch, options.Async)
		if err != nil {
			_ = router.Close()
			return nil, err
		}
		handler.async = async
	}
	return handler, nil
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if h == nil {
		return h
	}
	next := h.clone()
	for _, attr := range attrs {
		attr = normalizeAttr(attr)
		if attr.Key == loggerNameKey {
			if name := strings.TrimSpace(attr.Value.String()); name != "" {
				next.name = name
			}
			continue
		}
		next.attrs = appendAttrs(next.attrs, next.groups, []slog.Attr{attr})
	}
	return next
}

func (h nativeHandler) LogAttrs(
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
	return h.handler.logAttrs(ctx, logger, handlerAttrs, groups, when, level, message, pc, attrs)
}

// DefaultOptions 返回默认 Spring Boot 风格 stdout 配置。
func DefaultOptions() Options {
	return Options{
		Appenders: []Appender{NewConsoleAppender()},
		Root: RootLogger{
			Level:        slog.LevelInfo,
			AppenderRefs: []string{"console"},
		},
	}
}

// Options 描述 Handler 的运行期结构。
type Options struct {
	Appenders []Appender
	Filters   []Filter
	Root      RootLogger
	Loggers   []LoggerRule
	Async     AsyncLoggerOptions
}

// Handler 是 goark-log 的 slog.Handler 实现。
type Handler struct {
	router *internalrouter.Router
	name   string
	attrs  []slog.Attr
	groups []string
	async  *internalasynclogger.Logger
}

// NewDefaultHandler 创建默认 stdout INFO Handler。
func NewDefaultHandler() *Handler {
	handler, err := NewHandler(DefaultOptions())
	if err != nil {
		panic(err)
	}
	return handler
}

func (h *Handler) dispatch(ctx context.Context, event Event, levelAccepted bool) error {
	plan := h.router.Plan(event.Logger)
	if !levelAccepted && event.Level < plan.Route.Level {
		return nil
	}
	return h.dispatchRoute(ctx, plan.Route, event)
}

func (h *Handler) clone() *Handler {
	next := *h
	next.attrs = append([]slog.Attr(nil), h.attrs...)
	next.groups = append([]string(nil), h.groups...)
	return &next
}

// NewLogger 基于 handler 创建命名 logger。
func NewLogger(handler slog.Handler, name string) *slog.Logger {
	return slog.New(handler).With(loggerNameKey, name)
}

// WithLoggerCaller 设置是否采集调用位置。
func WithLoggerCaller(enabled bool) LoggerOption {
	return internalnativelogger.WithCaller(enabled)
}

func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	return h.enabled(h.name, level)
}

type nativeHandler struct {
	handler *Handler
}

func (h nativeHandler) IncludeCaller(logger string) bool {
	return h.handler.asyncIncludeLocation() || h.handler.routeIncludeLocation(logger)
}

// LoggerConfiguration 描述 Logger 的显式级别与最终生效级别。
type LoggerConfiguration = internalrouter.LoggerConfiguration

// LoggerOption 调整原生 Logger。
type LoggerOption = internalnativelogger.Option

func (h nativeHandler) SlogHandler() slog.Handler {
	return h.handler
}
