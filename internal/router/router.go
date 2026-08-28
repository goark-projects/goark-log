package router

import (
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync/atomic"

	logfilter "goark.dev/log/internal/filter"
)

// AsyncAppenderMatcher 判断 appender 是否需要优先关闭。
type AsyncAppenderMatcher func(Appender) bool

// Options 是路由运行期需要的最小配置。
type Options struct {
	Appenders       []Appender
	Filters         []Filter
	Root            RootLogger
	Loggers         []LoggerRule
	IsAsyncAppender AsyncAppenderMatcher
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

// Router 保存不可变路由快照，并为 reload 提供原子替换边界。
type Router struct {
	current atomic.Pointer[runtimeConfig]
}

type runtimeConfig struct {
	root            Route
	loggers         []loggerRuntime
	globalFilters   []Filter
	all             []Appender
	includeLocation bool
	isAsyncAppender AsyncAppenderMatcher
}

type loggerRuntime struct {
	name  string
	route Route
}

// Route 是一次 logger 匹配后的最终输出计划。
type Route struct {
	Level           slog.Level
	Appenders       []AppenderControl
	Filters         []Filter
	IncludeLocation bool
}

// Plan 是一次 logger 名称匹配得到的路由计划。
type Plan struct {
	Route         Route
	GlobalFilters []Filter
}

// New 创建路由运行期。
func New(options Options) (*Router, error) {
	config, err := buildRuntimeConfig(options)
	if err != nil {
		return nil, err
	}
	router := &Router{}
	router.current.Store(config)
	return router, nil
}

// Plan 返回指定 logger 名称的路由计划。
func (r *Router) Plan(name string) Plan {
	if r == nil {
		return Plan{Route: Route{Level: slog.LevelInfo}}
	}
	config := r.current.Load()
	if config == nil {
		return Plan{Route: Route{Level: slog.LevelInfo}}
	}
	return routePlanFromConfig(config, name)
}

// IncludeLocation 返回指定 logger 是否需要采集调用位置。
func (r *Router) IncludeLocation(name string) bool {
	if r == nil {
		return false
	}
	config := r.current.Load()
	if config == nil || !config.includeLocation {
		return false
	}
	return routePlanFromConfig(config, name).Route.IncludeLocation
}

// Close 关闭当前快照中的所有 appender。
func (r *Router) Close() error {
	if r == nil {
		return nil
	}
	config := r.current.Load()
	if config == nil {
		return nil
	}
	return config.close()
}

// Replace 原子替换运行期配置，只有新配置构建成功后才关闭旧 appender。
func (r *Router) Replace(options Options) error {
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

// CloseAppenders 按运行期关闭顺序关闭一组 appender。
func CloseAppenders(appenders []Appender, isAsyncAppender AsyncAppenderMatcher) error {
	config := &runtimeConfig{
		all:             appenders,
		isAsyncAppender: isAsyncAppender,
	}
	return config.close()
}

func routePlanFromConfig(config *runtimeConfig, name string) Plan {
	name = strings.TrimSpace(name)
	for _, logger := range config.loggers {
		if !loggerMatches(name, logger.name) {
			continue
		}
		return Plan{Route: logger.route, GlobalFilters: config.globalFilters}
	}
	return Plan{Route: config.root, GlobalFilters: config.globalFilters}
}

func (c *runtimeConfig) close() error {
	if c == nil {
		return nil
	}
	var joined error
	closed := make(map[string]struct{}, len(c.all))
	if c.isAsyncAppender != nil {
		for _, appender := range c.all {
			if appender != nil && c.isAsyncAppender(appender) {
				closed[appender.Name()] = struct{}{}
				joined = errors.Join(joined, appender.Close())
			}
		}
	}
	for _, appender := range c.all {
		if appender == nil {
			continue
		}
		if _, ok := closed[appender.Name()]; ok {
			continue
		}
		joined = errors.Join(joined, appender.Close())
	}
	return joined
}

func buildRuntimeConfig(options Options) (*runtimeConfig, error) {
	if len(options.Appenders) == 0 {
		return nil, fmt.Errorf("goark-log: requires at least one appender")
	}
	globalFilters, err := logfilter.Normalize("global", options.Filters)
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
	rootFilters, err := logfilter.Normalize("root", options.Root.Filters)
	if err != nil {
		return nil, err
	}
	config := &runtimeConfig{
		root: Route{
			Level:           options.Root.Level,
			Appenders:       rootAppenders,
			Filters:         rootFilters,
			IncludeLocation: routeRequiresLocation(options.Root.IncludeLocation, rootAppenders),
		},
		globalFilters:   globalFilters,
		all:             all,
		isAsyncAppender: options.IsAsyncAppender,
	}
	config.includeLocation = config.root.IncludeLocation
	for _, rule := range options.Loggers {
		if err := appendLoggerRuntime(config, appenderByName, rootFilters, options.Root, rule); err != nil {
			return nil, err
		}
	}
	sort.Slice(config.loggers, func(i, j int) bool {
		return loggerSpecificity(config.loggers[i].name) > loggerSpecificity(config.loggers[j].name)
	})
	return config, nil
}

func appendLoggerRuntime(config *runtimeConfig, appenderByName map[string]Appender, rootFilters []Filter, root RootLogger, rule LoggerRule) error {
	name := strings.TrimSpace(rule.Name)
	if name == "" {
		return fmt.Errorf("goark-log: logger name is empty")
	}
	additivity := true
	if rule.AdditivitySet {
		additivity = rule.Additivity
	}
	appenders, err := resolveAppenderControls(appenderByName, mergeAppenderRefs(rule.AppenderRefs, rule.AppenderRefControls))
	if err != nil {
		return fmt.Errorf("goark-log: logger %q: %w", name, err)
	}
	if !additivity && len(appenders) == 0 {
		return fmt.Errorf("goark-log: logger %q disables additivity but has no appender", name)
	}
	filters, err := logfilter.Normalize("logger "+name, rule.Filters)
	if err != nil {
		return err
	}
	effectiveFilters := append([]Filter(nil), filters...)
	if additivity {
		appenders = appendUniqueAppenderControls(appenders, config.root.Appenders)
		effectiveFilters = logfilter.Append(effectiveFilters, rootFilters)
	}
	includeLocation := root.IncludeLocation
	if rule.IncludeLocation != nil {
		includeLocation = *rule.IncludeLocation
	}
	loggerRoute := Route{
		Level:           loggerLevel(root.Level, rule.Level),
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
	return nil
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

func routeRequiresLocation(includeLocation bool, appenders []AppenderControl) bool {
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
