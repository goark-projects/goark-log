package goarklog

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync/atomic"
	"time"
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

// Handler 是 goark-log 的 slog.Handler 实现。
type Handler struct {
	router *router
	name   string
	attrs  []slog.Attr
	groups []string
	async  *asyncLogger
}

var _ slog.Handler = (*Handler)(nil)

// NewHandler 创建 slog.Handler。
func NewHandler(options Options) (*Handler, error) {
	router, err := newRouter(options)
	if err != nil {
		return nil, err
	}
	handler := &Handler{router: router, name: defaultLoggerName}
	if options.Async.Enabled {
		async, err := newAsyncLogger(handler, options.Async)
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
			return h.async.append(ctx, event, false)
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
		return h.async.append(ctx, event, levelAccepted)
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
		return h.async.append(ctx, event, levelAccepted)
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
		if err := h.async.close(); err != nil {
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
		if !sameAsyncLoggerRuntimeOptions(normalized, h.async.options) {
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
	return h.async.droppedCount()
}

// AsyncFailed 返回 Handler 层异步后台写入失败批次数量。
func (h *Handler) AsyncFailed() uint64 {
	if h == nil || h.async == nil {
		return 0
	}
	return h.async.failedCount()
}

// AsyncRemainingCapacity 返回 Handler 层异步队列剩余容量。
func (h *Handler) AsyncRemainingCapacity() int64 {
	if h == nil || h.async == nil {
		return 0
	}
	return h.async.remainingCapacity()
}

func (h *Handler) asyncIncludeLocation() bool {
	return h != nil && h.async != nil && h.async.includeLocation()
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

// router 保存不可变路由快照，并为后续 reload 留出原子替换边界。
type router struct {
	current atomic.Pointer[runtimeConfig]
}

type runtimeConfig struct {
	root            route
	loggers         []loggerRuntime
	globalFilters   []Filter
	all             []Appender
	includeLocation bool
}

type loggerRuntime struct {
	name  string
	route route
}

// route 是一次 logger 匹配后的最终输出计划。
type route struct {
	Level           slog.Level
	Appenders       []appenderControl
	Filters         []Filter
	IncludeLocation bool
}

type routePlan struct {
	route         route
	globalFilters []Filter
}

func newRouter(options Options) (*router, error) {
	config, err := buildRuntimeConfig(options)
	if err != nil {
		return nil, err
	}
	router := &router{}
	router.current.Store(config)
	return router, nil
}

func (r *router) route(name string) route {
	return r.plan(name).route
}

func (r *router) plan(name string) routePlan {
	if r == nil {
		return routePlan{route: route{Level: slog.LevelInfo}}
	}
	config := r.current.Load()
	if config == nil {
		return routePlan{route: route{Level: slog.LevelInfo}}
	}
	return routePlanFromConfig(config, name)
}

func routePlanFromConfig(config *runtimeConfig, name string) routePlan {
	name = strings.TrimSpace(name)
	for _, logger := range config.loggers {
		if !loggerMatches(name, logger.name) {
			continue
		}
		return routePlan{route: logger.route, globalFilters: config.globalFilters}
	}
	return routePlan{route: config.root, globalFilters: config.globalFilters}
}

func (r *router) Close() error {
	config := r.current.Load()
	if config == nil {
		return nil
	}
	return config.close()
}

// Replace 原子替换运行期配置，只有新配置构建成功后才关闭旧 appender。
func (r *router) Replace(options Options) error {
	if r == nil {
		return fmt.Errorf("goark-log: router is nil")
	}
	config, err := buildRuntimeConfig(options)
	if err != nil {
		return err
	}
	old := r.current.Swap(config)
	if old == nil {
		return nil
	}
	return old.close()
}

func (c *runtimeConfig) close() error {
	if c == nil {
		return nil
	}
	var joined error
	closed := make(map[string]struct{}, len(c.all))
	for _, appender := range c.all {
		if isAsyncAppender(appender) && appender != nil {
			closed[appender.Name()] = struct{}{}
			joined = errors.Join(joined, appender.Close())
		}
	}
	for _, appender := range c.all {
		if appender != nil {
			if _, ok := closed[appender.Name()]; ok {
				continue
			}
			joined = errors.Join(joined, appender.Close())
		}
	}
	return joined
}

func buildRuntimeConfig(options Options) (*runtimeConfig, error) {
	if len(options.Appenders) == 0 {
		defaults := DefaultOptions()
		options.Appenders = defaults.Appenders
		if len(options.Root.AppenderRefs) == 0 && len(options.Root.AppenderRefControls) == 0 {
			options.Root.AppenderRefs = defaults.Root.AppenderRefs
		}
	}
	globalFilters, err := normalizeFilters("global", options.Filters)
	if err != nil {
		return nil, err
	}
	appenderByName := make(map[string]Appender, len(options.Appenders))
	all := make([]Appender, 0, len(options.Appenders))
	for _, appender := range options.Appenders {
		if appender == nil {
			return nil, fmt.Errorf("goark-log: appender is nil")
		}
		name := strings.TrimSpace(appender.Name())
		if name == "" {
			return nil, fmt.Errorf("goark-log: appender name is empty")
		}
		if _, exists := appenderByName[name]; exists {
			return nil, fmt.Errorf("goark-log: duplicate appender %q", name)
		}
		appenderByName[name] = appender
		all = append(all, appender)
	}
	rootRefs := mergeAppenderRefs(options.Root.AppenderRefs, options.Root.AppenderRefControls)
	if len(rootRefs) == 0 {
		rootRefs = []AppenderRef{{Ref: all[0].Name()}}
	}
	rootAppenders, err := resolveAppenderControls(appenderByName, rootRefs)
	if err != nil {
		return nil, err
	}
	configRootFilters, err := normalizeFilters("root", options.Root.Filters)
	if err != nil {
		return nil, err
	}
	config := &runtimeConfig{
		root: route{
			Level:           options.Root.Level,
			Appenders:       rootAppenders,
			Filters:         configRootFilters,
			IncludeLocation: routeRequiresLocation(options.Root.IncludeLocation, rootAppenders),
		},
		globalFilters: globalFilters,
		all:           all,
	}
	config.includeLocation = config.root.IncludeLocation
	for _, rule := range options.Loggers {
		name := strings.TrimSpace(rule.Name)
		if name == "" {
			return nil, fmt.Errorf("goark-log: logger name is empty")
		}
		additivity := true
		if rule.AdditivitySet {
			additivity = rule.Additivity
		}
		appenders, err := resolveAppenderControls(appenderByName, mergeAppenderRefs(rule.AppenderRefs, rule.AppenderRefControls))
		if err != nil {
			return nil, fmt.Errorf("goark-log: logger %q: %w", name, err)
		}
		if !additivity && len(appenders) == 0 {
			return nil, fmt.Errorf("goark-log: logger %q disables additivity but has no appender", name)
		}
		filters, err := normalizeFilters("logger "+name, rule.Filters)
		if err != nil {
			return nil, err
		}
		effectiveFilters := append([]Filter(nil), filters...)
		if additivity {
			appenders = appendUniqueAppenderControls(appenders, config.root.Appenders)
			effectiveFilters = appendFilters(effectiveFilters, configRootFilters)
		}
		includeLocation := options.Root.IncludeLocation
		if rule.IncludeLocation != nil {
			includeLocation = *rule.IncludeLocation
		}
		loggerRoute := route{
			Level:           loggerLevel(options.Root.Level, rule.Level),
			Appenders:       appenders,
			Filters:         effectiveFilters,
			IncludeLocation: routeRequiresLocation(includeLocation, appenders),
		}
		if loggerRoute.IncludeLocation {
			config.includeLocation = true
		}
		config.loggers = append(config.loggers, loggerRuntime{
			name:  name,
			route: loggerRoute,
		})
	}
	sort.Slice(config.loggers, func(i, j int) bool {
		return loggerSpecificity(config.loggers[i].name) > loggerSpecificity(config.loggers[j].name)
	})
	return config, nil
}

func loggerMatches(name string, rule string) bool {
	return name == rule || strings.HasPrefix(name, rule+".")
}

func loggerSpecificity(name string) int {
	return strings.Count(name, ".")*1024 + len(name)
}

func loggerLevel(root slog.Level, level *slog.Level) slog.Level {
	if level == nil {
		return root
	}
	return *level
}

func routeRequiresLocation(includeLocation bool, appenders []appenderControl) bool {
	if includeLocation {
		return true
	}
	for _, appender := range appenders {
		if appender.requiresLocation() {
			return true
		}
	}
	return false
}
