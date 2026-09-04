package router

import (
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
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
	mu      sync.Mutex
	base    *runtimeConfig
	levels  map[string]slog.Level
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
	name            string
	configuredLevel *slog.Level
	route           Route
}

// LoggerConfiguration 描述 Logger 的显式级别与最终生效级别。
type LoggerConfiguration struct {
	Name            string
	ConfiguredLevel *slog.Level
	EffectiveLevel  slog.Level
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
	router := &Router{base: config, levels: make(map[string]slog.Level)}
	router.current.Store(config)
	return router, nil
}

// SetLevel 原子设置 Logger 级别；level 为 nil 时恢复配置文件定义或继承关系。
func (r *Router) SetLevel(name string, level *slog.Level) error {
	if r == nil {
		return fmt.Errorf("goark-log: router is nil")
	}
	name = normalizeLevelName(name)
	if name == "" {
		return fmt.Errorf("goark-log: logger name is empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.base == nil {
		return fmt.Errorf("goark-log: router is closed")
	}
	if level == nil {
		delete(r.levels, name)
	} else {
		r.levels[name] = *level
	}
	r.current.Store(applyLevelOverrides(r.base, r.levels))
	return nil
}

// Configurations 返回 Root 和命名 Logger 的稳定配置快照。
func (r *Router) Configurations() []LoggerConfiguration {
	if r == nil {
		return nil
	}
	config := r.current.Load()
	if config == nil {
		return nil
	}
	result := make([]LoggerConfiguration, 0, len(config.loggers)+1)
	rootLevel := config.root.Level
	result = append(result, LoggerConfiguration{Name: "ROOT", ConfiguredLevel: levelPointer(rootLevel), EffectiveLevel: rootLevel})
	for _, logger := range config.loggers {
		result = append(result, LoggerConfiguration{
			Name:            logger.name,
			ConfiguredLevel: cloneLevel(logger.configuredLevel),
			EffectiveLevel:  logger.route.Level,
		})
	}
	sort.Slice(result[1:], func(i, j int) bool { return result[i+1].Name < result[j+1].Name })
	return result
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
	r.mu.Lock()
	if r.base == nil {
		r.mu.Unlock()
		return nil
	}
	config := r.current.Load()
	r.base = nil
	r.mu.Unlock()
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
	r.mu.Lock()
	if r.base == nil {
		r.mu.Unlock()
		_ = config.close()
		return fmt.Errorf("goark-log: router is closed")
	}
	r.base = config
	current := applyLevelOverrides(config, r.levels)
	old := r.current.Swap(current)
	r.mu.Unlock()
	if old == nil {
		return nil
	}
	return old.close()
}

func applyLevelOverrides(base *runtimeConfig, levels map[string]slog.Level) *runtimeConfig {
	if base == nil || len(levels) == 0 {
		return base
	}
	config := *base
	config.loggers = append([]loggerRuntime(nil), base.loggers...)
	names := make([]string, 0, len(levels))
	for name := range levels {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		level := levels[name]
		if name == "ROOT" {
			continue
		}
		found := false
		for index := range config.loggers {
			if config.loggers[index].name != name {
				continue
			}
			config.loggers[index].configuredLevel = levelPointer(level)
			config.loggers[index].route.Level = level
			found = true
			break
		}
		if found {
			continue
		}
		route := routePlanFromConfig(base, name).Route
		config.loggers = append(config.loggers, loggerRuntime{name: name, configuredLevel: levelPointer(level), route: route})
	}
	sort.Slice(config.loggers, func(i, j int) bool {
		return loggerSpecificity(config.loggers[i].name) > loggerSpecificity(config.loggers[j].name)
	})
	config.root.Level = effectiveRootLevel(base.root.Level, levels)
	for index := range config.loggers {
		config.loggers[index].route.Level = effectiveLoggerLevel(config.root.Level, config.loggers[index].name, config.loggers)
	}
	return &config
}

func effectiveRootLevel(configured slog.Level, levels map[string]slog.Level) slog.Level {
	if level, exists := levels["ROOT"]; exists {
		return level
	}
	return configured
}

func effectiveLoggerLevel(root slog.Level, name string, loggers []loggerRuntime) slog.Level {
	for _, logger := range loggers {
		if logger.configuredLevel != nil && loggerMatches(name, logger.name) {
			return *logger.configuredLevel
		}
	}
	return root
}

func normalizeLevelName(name string) string {
	name = strings.TrimSpace(name)
	if strings.EqualFold(name, "root") {
		return "ROOT"
	}
	return name
}

func cloneLevel(level *slog.Level) *slog.Level {
	if level == nil {
		return nil
	}
	return levelPointer(*level)
}

func levelPointer(level slog.Level) *slog.Level {
	copy := level
	return &copy
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
	for index := range config.loggers {
		config.loggers[index].route.Level = effectiveLoggerLevel(config.root.Level, config.loggers[index].name, config.loggers)
	}
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
		name:            name,
		configuredLevel: cloneLevel(rule.Level),
		route:           loggerRoute,
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
