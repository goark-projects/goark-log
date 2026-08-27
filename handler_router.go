package goarklog

import (
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync/atomic"
)

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
