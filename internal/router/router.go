package router

import (
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"sync/atomic"
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
	result = append(
		result,
		LoggerConfiguration{
			Name:            "ROOT",
			ConfiguredLevel: levelPointer(rootLevel),
			EffectiveLevel:  rootLevel,
		},
	)
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
