package goarklog

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

// AppenderFactory 从配置构建 Appender。
type AppenderFactory func(config AppenderBuildConfig) (Appender, error)

// LayoutFactory 从配置构建 Layout。
type LayoutFactory func(config LayoutBuildConfig) (Layout, error)

// FilterFactory 从配置构建 Filter。
type FilterFactory func(config FilterBuildConfig) (Filter, error)

// AppenderBuildConfig 是 appender 插件的构建输入。
type AppenderBuildConfig struct {
	Name             string
	Type             string
	Target           string
	FileName         string
	Layout           Layout
	AppenderRefs     []string
	Delegates        []Appender
	QueueSize        int
	OverflowStrategy string
	WaitStrategy     string
	BufferSize       string
	FlushOnWrite     bool
	Rolling          RollingBuildConfig
}

// RollingBuildConfig 是滚动文件插件的构建输入。
type RollingBuildConfig struct {
	FilePattern     string
	MaxSize         string
	Interval        string
	TimeModulate    *bool
	OnStartup       bool
	MaxBackups      *int
	MaxAge          string
	FileIndex       string
	Gzip            bool
	AsyncActions    bool
	DeleteActions   []RollingDeleteBuildConfig
	ActionQueueSize int
}

// RollingDeleteBuildConfig 是 YAML 删除动作的中间配置。
type RollingDeleteBuildConfig struct {
	BasePath string
	MaxDepth int
	Glob     string
	MaxAge   string
}

// LayoutBuildConfig 是 layout 插件的构建输入。
type LayoutBuildConfig struct {
	Type    string
	Pattern string
}

// FilterBuildConfig 是 filter 插件的构建输入。
type FilterBuildConfig struct {
	Name       string
	Type       string
	Level      string
	MinLevel   string
	MaxLevel   string
	Field      string
	Key        string
	Value      string
	Pattern    string
	OnMatch    string
	OnMismatch string
}

// PluginRegistry 保存显式注册的日志插件。
type PluginRegistry struct {
	mu        sync.RWMutex
	appenders map[string]AppenderFactory
	layouts   map[string]LayoutFactory
	filters   map[string]FilterFactory
	lookups   map[string]LookupFunc
}

var (
	defaultRegistryOnce sync.Once
	defaultRegistry     *PluginRegistry
)

// NewPluginRegistry 创建包含内置插件的新注册表。
func NewPluginRegistry() *PluginRegistry {
	registry := &PluginRegistry{
		appenders: make(map[string]AppenderFactory),
		layouts:   make(map[string]LayoutFactory),
		filters:   make(map[string]FilterFactory),
		lookups:   make(map[string]LookupFunc),
	}
	registerBuiltInPlugins(registry)
	return registry
}

// DefaultPluginRegistry 返回进程默认插件注册表。
func DefaultPluginRegistry() *PluginRegistry {
	defaultRegistryOnce.Do(func() {
		defaultRegistry = NewPluginRegistry()
	})
	return defaultRegistry
}

// RegisterAppender 注册 appender 插件。
func (r *PluginRegistry) RegisterAppender(kind string, factory AppenderFactory) error {
	if r == nil {
		return fmt.Errorf("goark-log: plugin registry is nil")
	}
	if factory == nil {
		return fmt.Errorf("goark-log: appender factory is nil")
	}
	kind = normalizeKind(kind)
	if kind == "" {
		return fmt.Errorf("goark-log: appender kind is empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.appenders[kind] = factory
	return nil
}

// RegisterLayout 注册 layout 插件。
func (r *PluginRegistry) RegisterLayout(kind string, factory LayoutFactory) error {
	if r == nil {
		return fmt.Errorf("goark-log: plugin registry is nil")
	}
	if factory == nil {
		return fmt.Errorf("goark-log: layout factory is nil")
	}
	kind = normalizeKind(kind)
	if kind == "" {
		return fmt.Errorf("goark-log: layout kind is empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.layouts[kind] = factory
	return nil
}

// RegisterFilter 注册 filter 插件。
func (r *PluginRegistry) RegisterFilter(kind string, factory FilterFactory) error {
	if r == nil {
		return fmt.Errorf("goark-log: plugin registry is nil")
	}
	if factory == nil {
		return fmt.Errorf("goark-log: filter factory is nil")
	}
	kind = normalizeKind(kind)
	if kind == "" {
		return fmt.Errorf("goark-log: filter kind is empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.filters[kind] = factory
	return nil
}

// RegisterLookup 注册配置变量 lookup。
func (r *PluginRegistry) RegisterLookup(namespace string, lookup LookupFunc) error {
	if r == nil {
		return fmt.Errorf("goark-log: plugin registry is nil")
	}
	if lookup == nil {
		return fmt.Errorf("goark-log: lookup factory is nil")
	}
	namespace = strings.ToLower(strings.TrimSpace(namespace))
	if namespace == "" {
		return fmt.Errorf("goark-log: lookup namespace is empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lookups[namespace] = lookup
	return nil
}

func (r *PluginRegistry) lookupResolver() *LookupResolver {
	resolver := NewLookupResolver()
	if r == nil {
		return resolver
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for namespace, lookup := range r.lookups {
		resolver.Register(namespace, lookup)
	}
	return resolver
}

func (r *PluginRegistry) appenderFactory(kind string) (AppenderFactory, bool) {
	if r == nil {
		r = DefaultPluginRegistry()
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	factory, ok := r.appenders[normalizeKind(kind)]
	return factory, ok
}

func (r *PluginRegistry) layoutFactory(kind string) (LayoutFactory, bool) {
	if r == nil {
		r = DefaultPluginRegistry()
	}
	kind = normalizeKind(kind)
	if kind == "" {
		kind = "pattern"
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	factory, ok := r.layouts[kind]
	return factory, ok
}

func (r *PluginRegistry) filterFactory(kind string) (FilterFactory, bool) {
	if r == nil {
		r = DefaultPluginRegistry()
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	factory, ok := r.filters[normalizeKind(kind)]
	return factory, ok
}

func registerBuiltInPlugins(registry *PluginRegistry) {
	_ = registry.RegisterAppender("console", buildConsolePlugin)
	_ = registry.RegisterAppender("file", buildFilePlugin)
	_ = registry.RegisterAppender("rolling", buildRollingPlugin)
	_ = registry.RegisterAppender("rollingFile", buildRollingPlugin)
	_ = registry.RegisterAppender("async", buildAsyncPlugin)

	_ = registry.RegisterLayout("pattern", func(config LayoutBuildConfig) (Layout, error) {
		return NewPatternLayout(config.Pattern)
	})
	_ = registry.RegisterLayout("text", func(_ LayoutBuildConfig) (Layout, error) {
		return TextLayout{}, nil
	})
	_ = registry.RegisterLayout("json", func(_ LayoutBuildConfig) (Layout, error) {
		return JSONLayout{}, nil
	})

	_ = registry.RegisterFilter("threshold", buildThresholdFilterPlugin)
	_ = registry.RegisterFilter("level", buildLevelFilterPlugin)
	_ = registry.RegisterFilter("levelRange", buildLevelRangeFilterPlugin)
	_ = registry.RegisterFilter("regex", buildRegexFilterPlugin)
	_ = registry.RegisterFilter("attr", buildAttrFilterPlugin)
	_ = registry.RegisterFilter("attribute", buildAttrFilterPlugin)
}

func buildConsolePlugin(config AppenderBuildConfig) (Appender, error) {
	target := strings.ToLower(strings.TrimSpace(config.Target))
	switch target {
	case "", "stderr":
		return NewConsoleAppender(WithConsoleName(config.Name), WithConsoleLayout(config.Layout), WithConsoleWriter(os.Stderr)), nil
	case "stdout":
		return NewConsoleAppender(WithConsoleName(config.Name), WithConsoleLayout(config.Layout), WithConsoleWriter(os.Stdout)), nil
	default:
		return nil, fmt.Errorf("goark-log: appender %q console target %q is invalid", config.Name, config.Target)
	}
}

func buildFilePlugin(config AppenderBuildConfig) (Appender, error) {
	if config.FileName == "" {
		return nil, fmt.Errorf("goark-log: appender %q fileName is empty", config.Name)
	}
	options := []FileOption{
		WithFileName(config.Name),
		WithFileLayout(config.Layout),
	}
	if strings.TrimSpace(config.BufferSize) != "" {
		size, err := ParseByteSize(config.BufferSize)
		if err != nil {
			return nil, fmt.Errorf("goark-log: appender %q: %w", config.Name, err)
		}
		options = append(options, WithFileBufferSize(int(size)))
	}
	if config.FlushOnWrite {
		options = append(options, WithFileFlushOnWrite(true))
	}
	return NewFileAppender(config.FileName, options...)
}

func buildRollingPlugin(config AppenderBuildConfig) (Appender, error) {
	if config.FileName == "" {
		return nil, fmt.Errorf("goark-log: appender %q fileName is empty", config.Name)
	}
	options := []RollingFileOption{
		WithRollingFileName(config.Name),
		WithRollingFileLayout(config.Layout),
	}
	if strings.TrimSpace(config.BufferSize) != "" {
		size, err := ParseByteSize(config.BufferSize)
		if err != nil {
			return nil, fmt.Errorf("goark-log: appender %q: %w", config.Name, err)
		}
		options = append(options, WithRollingFileBufferSize(int(size)))
	}
	if config.FlushOnWrite {
		options = append(options, WithRollingFileFlushOnWrite(true))
	}
	if strings.TrimSpace(config.Rolling.FilePattern) != "" {
		options = append(options, WithRollingFilePattern(config.Rolling.FilePattern))
	}
	if value := strings.ToLower(strings.TrimSpace(config.Rolling.FileIndex)); value != "" {
		switch value {
		case "max", "nomax", "no-max", "none":
		default:
			return nil, fmt.Errorf("goark-log: appender %q rolling fileIndex %q is unsupported", config.Name, config.Rolling.FileIndex)
		}
	}
	if value := config.Rolling.MaxSize; value != "" {
		size, err := ParseByteSize(value)
		if err != nil {
			return nil, fmt.Errorf("goark-log: appender %q: %w", config.Name, err)
		}
		options = append(options, WithRollingMaxSize(size))
	}
	if value := config.Rolling.Interval; strings.TrimSpace(value) != "" {
		interval, err := ParseRollingInterval(value)
		if err != nil {
			return nil, fmt.Errorf("goark-log: appender %q: %w", config.Name, err)
		}
		options = append(options, WithRollingInterval(interval))
	}
	if config.Rolling.TimeModulate != nil {
		options = append(options, WithRollingTimeModulate(*config.Rolling.TimeModulate))
	}
	if config.Rolling.OnStartup {
		options = append(options, WithRolloverOnStartup(true))
	}
	if config.Rolling.MaxBackups != nil {
		options = append(options, WithRollingMaxBackups(*config.Rolling.MaxBackups))
	}
	if strings.TrimSpace(config.Rolling.MaxAge) != "" {
		age, err := ParseRollingMaxAge(config.Rolling.MaxAge)
		if err != nil {
			return nil, fmt.Errorf("goark-log: appender %q: %w", config.Name, err)
		}
		options = append(options, WithRollingMaxAge(age))
	}
	if config.Rolling.Gzip {
		options = append(options, WithRollingGzip(true))
	}
	if config.Rolling.AsyncActions {
		options = append(options, WithRollingAsyncActions(true))
	}
	if config.Rolling.ActionQueueSize > 0 {
		options = append(options, WithRollingActionQueueSize(config.Rolling.ActionQueueSize))
	}
	if len(config.Rolling.DeleteActions) > 0 {
		actions := make([]RollingDeleteAction, 0, len(config.Rolling.DeleteActions))
		for index, actionConfig := range config.Rolling.DeleteActions {
			action, err := buildRollingDeleteAction(actionConfig)
			if err != nil {
				return nil, fmt.Errorf("goark-log: appender %q rolling delete action %d: %w", config.Name, index, err)
			}
			actions = append(actions, action)
		}
		options = append(options, WithRollingDeleteActions(actions...))
	}
	return NewRollingFileAppender(config.FileName, options...)
}

func buildRollingDeleteAction(config RollingDeleteBuildConfig) (RollingDeleteAction, error) {
	action := RollingDeleteAction{
		BasePath: config.BasePath,
		MaxDepth: config.MaxDepth,
		Glob:     config.Glob,
	}
	if strings.TrimSpace(config.MaxAge) != "" {
		age, err := ParseRollingMaxAge(config.MaxAge)
		if err != nil {
			return RollingDeleteAction{}, err
		}
		action.MaxAge = age
	}
	return action, nil
}

func buildAsyncPlugin(config AppenderBuildConfig) (Appender, error) {
	if len(config.Delegates) == 0 {
		return nil, fmt.Errorf("goark-log: async appender %q requires appenderRefs", config.Name)
	}
	strategy, err := ParseAsyncOverflowStrategy(config.OverflowStrategy)
	if err != nil {
		return nil, fmt.Errorf("goark-log: async appender %q: %w", config.Name, err)
	}
	options := []AsyncOption{
		WithAsyncName(config.Name),
		WithAsyncOverflowStrategy(strategy),
	}
	waitStrategy, err := ParseAsyncWaitStrategy(config.WaitStrategy)
	if err != nil {
		return nil, fmt.Errorf("goark-log: async appender %q: %w", config.Name, err)
	}
	options = append(options, WithAsyncWaitStrategy(waitStrategy))
	if config.QueueSize != 0 {
		options = append(options, WithAsyncQueueSize(config.QueueSize))
	}
	return NewAsyncAppender(config.Delegates, options...)
}

func buildThresholdFilterPlugin(config FilterBuildConfig) (Filter, error) {
	level, err := ParseLevel(config.Level)
	if err != nil {
		return nil, err
	}
	options, err := config.filterOptions()
	if err != nil {
		return nil, err
	}
	return NewThresholdFilter(level, options...), nil
}

func buildLevelFilterPlugin(config FilterBuildConfig) (Filter, error) {
	level, err := ParseLevel(config.Level)
	if err != nil {
		return nil, err
	}
	options, err := config.filterOptions()
	if err != nil {
		return nil, err
	}
	return NewLevelFilter(level, options...), nil
}

func buildLevelRangeFilterPlugin(config FilterBuildConfig) (Filter, error) {
	if config.MinLevel == "" || config.MaxLevel == "" {
		return nil, fmt.Errorf("goark-log: filter %q level range requires minLevel and maxLevel", config.Name)
	}
	min, err := ParseLevel(config.MinLevel)
	if err != nil {
		return nil, err
	}
	max, err := ParseLevel(config.MaxLevel)
	if err != nil {
		return nil, err
	}
	options, err := config.filterOptions()
	if err != nil {
		return nil, err
	}
	return NewLevelRangeFilter(min, max, options...)
}

func buildRegexFilterPlugin(config FilterBuildConfig) (Filter, error) {
	if strings.TrimSpace(config.Pattern) == "" {
		return nil, fmt.Errorf("goark-log: filter %q regex pattern is empty", config.Name)
	}
	options, err := config.regexOutcomeOptions()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(config.Field) != "" {
		field, err := parseRegexFilterField(config.Field)
		if err != nil {
			return nil, err
		}
		options = append(options, WithRegexField(field))
	}
	if strings.TrimSpace(config.Key) != "" {
		options = append(options, WithRegexAttrKey(config.Key))
	}
	return NewRegexFilter(config.Pattern, options...)
}

func buildAttrFilterPlugin(config FilterBuildConfig) (Filter, error) {
	options, err := config.filterOptions()
	if err != nil {
		return nil, err
	}
	return NewAttrFilter(config.Key, config.Value, options...)
}

func (c FilterBuildConfig) filterOptions() ([]FilterOption, error) {
	onMatch, err := parseFilterDecisionOrDefault(c.OnMatch, FilterNeutral)
	if err != nil {
		return nil, err
	}
	onMismatch, err := parseFilterDecisionOrDefault(c.OnMismatch, FilterDeny)
	if err != nil {
		return nil, err
	}
	return []FilterOption{
		WithFilterOnMatch(onMatch),
		WithFilterOnMismatch(onMismatch),
	}, nil
}

func (c FilterBuildConfig) regexOutcomeOptions() ([]RegexFilterOption, error) {
	onMatch, err := parseFilterDecisionOrDefault(c.OnMatch, FilterNeutral)
	if err != nil {
		return nil, err
	}
	onMismatch, err := parseFilterDecisionOrDefault(c.OnMismatch, FilterDeny)
	if err != nil {
		return nil, err
	}
	return []RegexFilterOption{
		WithRegexOnMatch(onMatch),
		WithRegexOnMismatch(onMismatch),
	}, nil
}
