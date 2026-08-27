package goarklog

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	internalasynclogger "goark.dev/log/internal/asynclogger"
)

// Options 描述 Handler 的运行期结构。
type Options struct {
	Appenders []Appender
	Filters   []Filter
	Root      RootLogger
	Loggers   []LoggerRule
	Async     AsyncLoggerOptions
}

// RootLogger 描述根 logger。
type RootLogger struct {
	Level               slog.Level
	AppenderRefs        []string
	AppenderRefControls []AppenderRef
	Filters             []Filter
	IncludeLocation     bool
}

// LoggerRule 描述命名 logger 的级别和输出路由。
type LoggerRule struct {
	Name                string
	Level               *slog.Level
	AppenderRefs        []string
	AppenderRefControls []AppenderRef
	Filters             []Filter
	Additivity          bool
	AdditivitySet       bool
	IncludeLocation     *bool
}

// DefaultOptions 返回默认 Spring Boot 风格 stderr 配置。
func DefaultOptions() Options {
	return Options{
		Appenders: []Appender{NewConsoleAppender()},
		Root: RootLogger{
			Level:        slog.LevelInfo,
			AppenderRefs: []string{"console"},
		},
	}
}

// Handler 是 goark-log 的 slog.Handler 实现。
type Handler struct {
	router *router
	name   string
	attrs  []slog.Attr
	groups []string
	async  *internalasynclogger.Logger
}

var _ slog.Handler = (*Handler)(nil)

// New 创建默认命名 logger 和对应 Handler。
func New(options Options) (*slog.Logger, *Handler, error) {
	handler, err := NewHandler(options)
	if err != nil {
		return nil, nil, err
	}
	return NewLogger(handler, defaultLoggerName), handler, nil
}

// NewHandler 创建 slog.Handler。
func NewHandler(options Options) (*Handler, error) {
	router, err := newRouter(options)
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

// NewDefaultHandler 创建默认 stderr INFO Handler。
func NewDefaultHandler() *Handler {
	handler, err := NewHandler(DefaultOptions())
	if err != nil {
		panic(err)
	}
	return handler
}

// NewDefault 创建默认 stderr INFO logger。
func NewDefault() (*slog.Logger, *Handler) {
	handler := NewDefaultHandler()
	return NewLogger(handler, defaultLoggerName), handler
}

// NewLogger 基于 handler 创建命名 logger。
func NewLogger(handler slog.Handler, name string) *slog.Logger {
	return slog.New(handler).With(loggerNameKey, name)
}

// WithName 返回带有 goark-log logger 名称的 logger。
func WithName(logger *slog.Logger, name string) *slog.Logger {
	if logger == nil {
		logger = slog.Default()
	}
	return logger.With(loggerNameKey, name)
}

func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	return h.enabled(h.name, level)
}

func (h *Handler) enabled(name string, level slog.Level) bool {
	if h == nil || h.router == nil {
		return level >= slog.LevelInfo
	}
	plan := h.router.plan(name)
	if len(plan.globalFilters) > 0 {
		return true
	}
	return level >= plan.route.Level
}

func (h *Handler) Handle(ctx context.Context, record slog.Record) error {
	if h == nil || h.router == nil {
		return fmt.Errorf("goark-log: handler is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	plan := h.router.plan(h.name)
	if len(plan.globalFilters) == 0 {
		if record.Level < plan.route.Level {
			return nil
		}
		event := newEvent(ctx, h.name, h.attrs, h.groups, record)
		if h.async != nil {
			return h.async.Append(ctx, event, false)
		}
		return h.dispatchRoute(ctx, plan.route, event)
	}
	event := newEvent(ctx, h.name, h.attrs, h.groups, record)
	levelAccepted, denied := applyGlobalFilters(ctx, plan.globalFilters, event)
	if denied {
		return nil
	}
	if !levelAccepted && record.Level < plan.route.Level {
		return nil
	}
	if h.async != nil {
		return h.async.Append(ctx, event, levelAccepted)
	}
	return h.dispatchRoute(ctx, plan.route, event)
}

func (h *Handler) dispatch(ctx context.Context, event Event, levelAccepted bool) error {
	plan := h.router.plan(event.Logger)
	if !levelAccepted && event.Level < plan.route.Level {
		return nil
	}
	return h.dispatchRoute(ctx, plan.route, event)
}

func (h *Handler) logAttrs(ctx context.Context, logger string, handlerAttrs []slog.Attr, groups []string, when time.Time, level slog.Level, message string, pc uintptr, attrs []slog.Attr) error {
	if h == nil || h.router == nil {
		return fmt.Errorf("goark-log: handler is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	plan := h.router.plan(logger)
	if len(plan.globalFilters) == 0 {
		if level < plan.route.Level {
			return nil
		}
		if h.async == nil && pc == 0 {
			if handled, err := h.dispatchAttrsFast(ctx, plan.route, logger, when, level, message, attrs); handled {
				return err
			}
		}
	}
	event := newEventFromAttrs(ctx, logger, handlerAttrs, groups, when, level, message, pc, attrs, h.async != nil)
	levelAccepted, denied := applyGlobalFilters(ctx, plan.globalFilters, event)
	if denied {
		return nil
	}
	if !levelAccepted && level < plan.route.Level {
		return nil
	}
	if h.async != nil {
		return h.async.Append(ctx, event, levelAccepted)
	}
	return h.dispatchRoute(ctx, plan.route, event)
}

func (h *Handler) log3Attrs(ctx context.Context, logger string, handlerAttrs []slog.Attr, groups []string, when time.Time, level slog.Level, message string, pc uintptr, attr0 slog.Attr, attr1 slog.Attr, attr2 slog.Attr) error {
	if h == nil || h.router == nil {
		return fmt.Errorf("goark-log: handler is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	plan := h.router.plan(logger)
	if len(plan.globalFilters) == 0 {
		if level < plan.route.Level {
			return nil
		}
		if h.async == nil && pc == 0 {
			if handled, err := h.dispatchFixedAttrsFast(ctx, plan.route, logger, when, level, message, attr0, attr1, attr2); handled {
				return err
			}
		}
	}
	attrs := []slog.Attr{attr0, attr1, attr2}
	return h.logAttrs(ctx, logger, handlerAttrs, groups, when, level, message, pc, attrs)
}

func applyGlobalFilters(ctx context.Context, filters []Filter, event Event) (levelAccepted bool, denied bool) {
	switch applyFilters(ctx, filters, event) {
	case FilterDeny:
		return false, true
	case FilterAccept:
		return true, false
	default:
		return false, false
	}
}

func (h *Handler) dispatchRoute(ctx context.Context, route route, event Event) error {
	if applyFilters(ctx, route.Filters, event) == FilterDeny {
		return nil
	}
	var joined error
	for _, appender := range route.Appenders {
		_, err := appender.append(ctx, event)
		if err != nil {
			joined = errors.Join(joined, err)
			continue
		}
	}
	return joined
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

// Close 关闭所有 appender。
func (h *Handler) Close() error {
	if h == nil || h.router == nil {
		return nil
	}
	if h.async != nil {
		if err := h.async.Close(); err != nil {
			return err
		}
	}
	return h.router.Close()
}

// Reload 使用新的运行期配置替换当前路由。
func (h *Handler) Reload(options Options) error {
	if h == nil || h.router == nil {
		return fmt.Errorf("goark-log: handler is nil")
	}
	if options.Async.Enabled != (h.async != nil) {
		return fmt.Errorf("goark-log: async logger enablement cannot be changed by reload")
	}
	if h.async != nil {
		normalized, err := normalizeAsyncLoggerOptions(options.Async)
		if err != nil {
			return err
		}
		if !sameAsyncLoggerRuntimeOptions(normalized, h.async.Options()) {
			return fmt.Errorf("goark-log: async logger queue settings cannot be changed by reload")
		}
	}
	return h.router.Replace(options)
}

// AsyncDropped 返回 Handler 层异步日志丢弃数量。
func (h *Handler) AsyncDropped() uint64 {
	if h == nil || h.async == nil {
		return 0
	}
	return h.async.Dropped()
}

// AsyncFailed 返回 Handler 层异步后台写入失败批次数量。
func (h *Handler) AsyncFailed() uint64 {
	if h == nil || h.async == nil {
		return 0
	}
	return h.async.Failed()
}

// AsyncRemainingCapacity 返回 Handler 层异步队列剩余容量。
func (h *Handler) AsyncRemainingCapacity() int64 {
	if h == nil || h.async == nil {
		return 0
	}
	return h.async.RemainingCapacity()
}

func (h *Handler) asyncIncludeLocation() bool {
	return h != nil && h.async != nil && h.async.IncludeLocation()
}

func (h *Handler) routeIncludeLocation(name string) bool {
	if h == nil || h.router == nil {
		return false
	}
	config := h.router.current.Load()
	if config == nil || !config.includeLocation {
		return false
	}
	return routePlanFromConfig(config, name).route.IncludeLocation
}

func (h *Handler) clone() *Handler {
	next := *h
	next.attrs = append([]slog.Attr(nil), h.attrs...)
	next.groups = append([]string(nil), h.groups...)
	return &next
}
