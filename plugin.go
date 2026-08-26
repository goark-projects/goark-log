package goarklog

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"
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
	URL              string
	Method           string
	Address          string
	Network          string
	Facility         string
	AppName          string
	ConnectTimeout   string
	WriteTimeout     string
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
	CronSchedule    string
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
	MaxCount int
	MaxSize  string
}

// LayoutBuildConfig 是 layout 插件的构建输入。
type LayoutBuildConfig struct {
	Type          string
	Pattern       string
	EventTemplate string
}

// FilterBuildConfig 是 filter 插件的构建输入。
type FilterBuildConfig struct {
	Name             string
	Type             string
	Level            string
	MinLevel         string
	MaxLevel         string
	Marker           string
	Text             string
	Operator         string
	Start            string
	End              string
	Timezone         string
	Rate             string
	MaxBurst         int
	Field            string
	Key              string
	Value            string
	Values           map[string]string
	Thresholds       map[string]string
	DefaultThreshold string
	Pattern          string
	OnMatch          string
	OnMismatch       string
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
	if isBlockedLookupNamespace(namespace) {
		return fmt.Errorf("goark-log: lookup namespace %q is blocked by security policy", namespace)
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
	_ = registry.RegisterLayout("jsonTemplate", func(config LayoutBuildConfig) (Layout, error) {
		return NewJSONTemplateLayout(config.EventTemplate)
	})
	_ = registry.RegisterLayout("xml", func(_ LayoutBuildConfig) (Layout, error) {
		return XMLLayout{}, nil
	})
	_ = registry.RegisterLayout("xmlLayout", func(_ LayoutBuildConfig) (Layout, error) {
		return XMLLayout{}, nil
	})
	_ = registry.RegisterLayout("csv", func(_ LayoutBuildConfig) (Layout, error) {
		return CSVLayout{}, nil
	})
	_ = registry.RegisterLayout("csvLayout", func(_ LayoutBuildConfig) (Layout, error) {
		return CSVLayout{}, nil
	})
	_ = registry.RegisterLayout("gelf", func(_ LayoutBuildConfig) (Layout, error) {
		return GELFLayout{}, nil
	})
	_ = registry.RegisterLayout("gelfLayout", func(_ LayoutBuildConfig) (Layout, error) {
		return GELFLayout{}, nil
	})
	_ = registry.RegisterLayout("rfc5424", func(_ LayoutBuildConfig) (Layout, error) {
		return RFC5424Layout{}, nil
	})
	_ = registry.RegisterLayout("rfc5424Layout", func(_ LayoutBuildConfig) (Layout, error) {
		return RFC5424Layout{}, nil
	})
	_ = registry.RegisterLayout("syslog", func(_ LayoutBuildConfig) (Layout, error) {
		return SyslogLayout{}, nil
	})
	_ = registry.RegisterLayout("syslogLayout", func(_ LayoutBuildConfig) (Layout, error) {
		return SyslogLayout{}, nil
	})
	_ = registry.RegisterLayout("yaml", func(_ LayoutBuildConfig) (Layout, error) {
		return YAMLLayout{}, nil
	})
	_ = registry.RegisterLayout("yamlLayout", func(_ LayoutBuildConfig) (Layout, error) {
		return YAMLLayout{}, nil
	})
	_ = registry.RegisterLayout("html", func(_ LayoutBuildConfig) (Layout, error) {
		return HTMLLayout{}, nil
	})
	_ = registry.RegisterLayout("htmlLayout", func(_ LayoutBuildConfig) (Layout, error) {
		return HTMLLayout{}, nil
	})

	_ = registry.RegisterFilter("threshold", buildThresholdFilterPlugin)
	_ = registry.RegisterFilter("thresholdFilter", buildThresholdFilterPlugin)
	_ = registry.RegisterFilter("level", buildLevelFilterPlugin)
	_ = registry.RegisterFilter("levelFilter", buildLevelFilterPlugin)
	_ = registry.RegisterFilter("levelRange", buildLevelRangeFilterPlugin)
	_ = registry.RegisterFilter("levelRangeFilter", buildLevelRangeFilterPlugin)
	_ = registry.RegisterFilter("regex", buildRegexFilterPlugin)
	_ = registry.RegisterFilter("regexFilter", buildRegexFilterPlugin)
	_ = registry.RegisterFilter("attr", buildAttrFilterPlugin)
	_ = registry.RegisterFilter("attribute", buildAttrFilterPlugin)
	_ = registry.RegisterFilter("attrFilter", buildAttrFilterPlugin)
	_ = registry.RegisterFilter("attributeFilter", buildAttrFilterPlugin)
	_ = registry.RegisterFilter("deny", buildDenyFilterPlugin)
	_ = registry.RegisterFilter("denyAll", buildDenyFilterPlugin)
	_ = registry.RegisterFilter("denyFilter", buildDenyFilterPlugin)
	_ = registry.RegisterFilter("denyAllFilter", buildDenyFilterPlugin)
	_ = registry.RegisterFilter("marker", buildMarkerFilterPlugin)
	_ = registry.RegisterFilter("markerFilter", buildMarkerFilterPlugin)
	_ = registry.RegisterFilter("noMarker", buildNoMarkerFilterPlugin)
	_ = registry.RegisterFilter("noMarkerFilter", buildNoMarkerFilterPlugin)
	_ = registry.RegisterFilter("map", buildMapFilterPlugin)
	_ = registry.RegisterFilter("mapFilter", buildMapFilterPlugin)
	_ = registry.RegisterFilter("threadContextMap", buildThreadContextMapFilterPlugin)
	_ = registry.RegisterFilter("threadContextMapFilter", buildThreadContextMapFilterPlugin)
	_ = registry.RegisterFilter("threadContextStack", buildThreadContextStackFilterPlugin)
	_ = registry.RegisterFilter("threadContextStackFilter", buildThreadContextStackFilterPlugin)
	_ = registry.RegisterFilter("structuredData", buildStructuredDataFilterPlugin)
	_ = registry.RegisterFilter("structuredDataFilter", buildStructuredDataFilterPlugin)
	_ = registry.RegisterFilter("throwable", buildThrowableFilterPlugin)
	_ = registry.RegisterFilter("throwableFilter", buildThrowableFilterPlugin)
	_ = registry.RegisterFilter("stringMatch", buildStringMatchFilterPlugin)
	_ = registry.RegisterFilter("stringMatchFilter", buildStringMatchFilterPlugin)
	_ = registry.RegisterFilter("time", buildTimeFilterPlugin)
	_ = registry.RegisterFilter("timeFilter", buildTimeFilterPlugin)
	_ = registry.RegisterFilter("burst", buildBurstFilterPlugin)
	_ = registry.RegisterFilter("burstFilter", buildBurstFilterPlugin)
	_ = registry.RegisterFilter("dynamicThreshold", buildDynamicThresholdFilterPlugin)
	_ = registry.RegisterFilter("dynamicThresholdFilter", buildDynamicThresholdFilterPlugin)
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
	if strings.TrimSpace(config.Rolling.CronSchedule) != "" {
		options = append(options, WithRollingCronSchedule(config.Rolling.CronSchedule))
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
		MaxCount: config.MaxCount,
	}
	if strings.TrimSpace(config.MaxAge) != "" {
		age, err := ParseRollingMaxAge(config.MaxAge)
		if err != nil {
			return RollingDeleteAction{}, err
		}
		action.MaxAge = age
	}
	if strings.TrimSpace(config.MaxSize) != "" {
		size, err := ParseByteSize(config.MaxSize)
		if err != nil {
			return RollingDeleteAction{}, err
		}
		action.MaxSize = size
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

func buildDenyFilterPlugin(FilterBuildConfig) (Filter, error) {
	return NewDenyFilter(), nil
}

func buildMarkerFilterPlugin(config FilterBuildConfig) (Filter, error) {
	options, err := config.filterOptions()
	if err != nil {
		return nil, err
	}
	return NewMarkerFilter(firstNonBlank(config.Marker, config.Value), options...)
}

func buildNoMarkerFilterPlugin(config FilterBuildConfig) (Filter, error) {
	options, err := config.filterOptions()
	if err != nil {
		return nil, err
	}
	return NewNoMarkerFilter(options...), nil
}

func buildMapFilterPlugin(config FilterBuildConfig) (Filter, error) {
	options, values, err := config.mapFilterOptions()
	if err != nil {
		return nil, err
	}
	return NewMapFilter(values, options...)
}

func buildThreadContextMapFilterPlugin(config FilterBuildConfig) (Filter, error) {
	options, values, err := config.mapFilterOptions()
	if err != nil {
		return nil, err
	}
	return NewThreadContextMapFilter(values, options...)
}

func buildThreadContextStackFilterPlugin(config FilterBuildConfig) (Filter, error) {
	options, err := config.filterOptions()
	if err != nil {
		return nil, err
	}
	return NewThreadContextStackFilter(firstNonBlank(config.Value, config.Text, config.Pattern), options...)
}

func buildStructuredDataFilterPlugin(config FilterBuildConfig) (Filter, error) {
	options, values, err := config.mapFilterOptions()
	if err != nil {
		return nil, err
	}
	return NewStructuredDataFilter(values, options...)
}

func buildThrowableFilterPlugin(config FilterBuildConfig) (Filter, error) {
	options, err := config.filterOptions()
	if err != nil {
		return nil, err
	}
	return NewThrowableFilter(firstNonBlank(config.Pattern, config.Text, config.Value), options...)
}

func buildStringMatchFilterPlugin(config FilterBuildConfig) (Filter, error) {
	options, err := config.filterOptions()
	if err != nil {
		return nil, err
	}
	return NewStringMatchFilter(firstNonBlank(config.Text, config.Value, config.Pattern), options...)
}

func buildTimeFilterPlugin(config FilterBuildConfig) (Filter, error) {
	options, err := config.filterOptions()
	if err != nil {
		return nil, err
	}
	start := firstNonBlank(config.Start, "00:00:00")
	end := firstNonBlank(config.End, "23:59:59.999999999")
	if strings.TrimSpace(config.Timezone) == "" {
		return NewTimeFilter(start, end, options...)
	}
	location, err := time.LoadLocation(strings.TrimSpace(config.Timezone))
	if err != nil {
		return nil, fmt.Errorf("goark-log: filter %q timezone %q is invalid", config.Name, config.Timezone)
	}
	return NewTimeFilterInLocation(start, end, location, options...)
}

func buildBurstFilterPlugin(config FilterBuildConfig) (Filter, error) {
	level, err := ParseLevel(firstNonBlank(config.Level, "warn"))
	if err != nil {
		return nil, err
	}
	rate := 10.0
	if strings.TrimSpace(config.Rate) != "" {
		parsed, err := parseFloat(config.Rate, "burst filter rate")
		if err != nil {
			return nil, err
		}
		rate = parsed
	}
	options, err := config.filterOptions()
	if err != nil {
		return nil, err
	}
	maxBurst := config.MaxBurst
	if maxBurst == 0 {
		maxBurst = int(rate * 10)
		if maxBurst <= 0 {
			maxBurst = 1
		}
	}
	return NewBurstFilter(level, rate, maxBurst, options...)
}

func buildDynamicThresholdFilterPlugin(config FilterBuildConfig) (Filter, error) {
	defaultLevel, err := ParseLevel(firstNonBlank(config.DefaultThreshold, config.Level, "error"))
	if err != nil {
		return nil, err
	}
	thresholds := make(map[string]slog.Level, len(config.Thresholds))
	for value, levelText := range config.Thresholds {
		level, err := ParseLevel(levelText)
		if err != nil {
			return nil, err
		}
		thresholds[value] = level
	}
	options, err := config.filterOptions()
	if err != nil {
		return nil, err
	}
	return NewDynamicThresholdFilter(config.Key, defaultLevel, thresholds, options...)
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

func (c FilterBuildConfig) mapFilterOptions() ([]MapFilterOption, map[string]string, error) {
	values := make(map[string]string, len(c.Values)+1)
	for key, value := range c.Values {
		values[key] = value
	}
	if strings.TrimSpace(c.Key) != "" {
		values[c.Key] = c.Value
	}
	operator, err := ParseMapFilterOperator(c.Operator)
	if err != nil {
		return nil, nil, err
	}
	onMatch, err := parseFilterDecisionOrDefault(c.OnMatch, FilterNeutral)
	if err != nil {
		return nil, nil, err
	}
	onMismatch, err := parseFilterDecisionOrDefault(c.OnMismatch, FilterDeny)
	if err != nil {
		return nil, nil, err
	}
	return []MapFilterOption{
		WithMapFilterOperator(operator),
		WithMapFilterOnMatch(onMatch),
		WithMapFilterOnMismatch(onMismatch),
	}, values, nil
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
