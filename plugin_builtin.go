package goarklog

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"goark.dev/log/internal/textutil"
)

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
	Routes           map[string]Appender
	DefaultRoute     Appender
	RouteKey         string
	QueueSize        int
	BatchSize        int
	OverflowStrategy string
	WaitStrategy     string
	WaitOptions      AsyncWaitOptions
	BufferSize       string
	FlushOnWrite     bool
	Append           *bool
	CreateOnDemand   bool
	FilePermissions  string
	Rolling          RollingBuildConfig
	Rewrite          RewriteBuildConfig
}

// RewriteBuildConfig 是 rewrite appender 的内置重写策略配置。
type RewriteBuildConfig struct {
	Attrs       map[string]string
	RemoveAttrs []string
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
	DirectWrite     bool
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
	Type             string
	Pattern          string
	EventTemplate    string
	EventTemplateURI string
	Options          LayoutOptions
	Registry         *PluginRegistry
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
	Filters          []Filter
	DefaultThreshold string
	Pattern          string
	OnMatch          string
	OnMismatch       string
}

func registerBuiltInPlugins(registry *PluginRegistry) {
	_ = registry.RegisterAppender("console", buildConsolePlugin)
	_ = registry.RegisterAppender("file", buildFilePlugin)
	_ = registry.RegisterAppender("json", buildJSONPlugin)
	_ = registry.RegisterAppender("jsonDirect", buildJSONPlugin)
	_ = registry.RegisterAppender("jsonWriter", buildJSONPlugin)
	_ = registry.RegisterAppender("rolling", buildRollingPlugin)
	_ = registry.RegisterAppender("rollingFile", buildRollingPlugin)
	_ = registry.RegisterAppender("async", buildAsyncPlugin)
	_ = registry.RegisterAppender("failover", buildFailoverPlugin)
	_ = registry.RegisterAppender("failoverAppender", buildFailoverPlugin)
	_ = registry.RegisterAppender("routing", buildRoutingPlugin)
	_ = registry.RegisterAppender("routingAppender", buildRoutingPlugin)
	_ = registry.RegisterAppender("rewrite", buildRewritePlugin)
	_ = registry.RegisterAppender("rewriteAppender", buildRewritePlugin)

	_ = registry.RegisterLayout("pattern", func(config LayoutBuildConfig) (Layout, error) {
		return NewPatternLayoutWithOptions(config.Pattern, config.Options)
	})
	_ = registry.RegisterLayout("text", func(_ LayoutBuildConfig) (Layout, error) {
		return TextLayout{}, nil
	})
	_ = registry.RegisterLayout("json", func(config LayoutBuildConfig) (Layout, error) {
		return NewJSONLayout(config.Options), nil
	})
	_ = registry.RegisterLayout("jsonTemplate", func(config LayoutBuildConfig) (Layout, error) {
		options := []JSONTemplateLayoutOption{
			WithJSONTemplateResolverRegistry(config.Registry),
			WithJSONTemplateLayoutOptions(config.Options),
		}
		if strings.TrimSpace(config.EventTemplateURI) != "" {
			return NewJSONTemplateLayoutFromFile(config.EventTemplateURI, options...)
		}
		return NewJSONTemplateLayout(config.EventTemplate, options...)
	})
	_ = registry.RegisterLayout("xml", func(config LayoutBuildConfig) (Layout, error) {
		return NewXMLLayout(config.Options), nil
	})
	_ = registry.RegisterLayout("xmlLayout", func(config LayoutBuildConfig) (Layout, error) {
		return NewXMLLayout(config.Options), nil
	})
	_ = registry.RegisterLayout("csv", func(config LayoutBuildConfig) (Layout, error) {
		return NewCSVLayout(config.Options), nil
	})
	_ = registry.RegisterLayout("csvLayout", func(config LayoutBuildConfig) (Layout, error) {
		return NewCSVLayout(config.Options), nil
	})
	_ = registry.RegisterLayout("gelf", func(config LayoutBuildConfig) (Layout, error) {
		return NewGELFLayout(config.Options), nil
	})
	_ = registry.RegisterLayout("gelfLayout", func(config LayoutBuildConfig) (Layout, error) {
		return NewGELFLayout(config.Options), nil
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
	_ = registry.RegisterLayout("yaml", func(config LayoutBuildConfig) (Layout, error) {
		return NewYAMLLayout(config.Options), nil
	})
	_ = registry.RegisterLayout("yamlLayout", func(config LayoutBuildConfig) (Layout, error) {
		return NewYAMLLayout(config.Options), nil
	})
	_ = registry.RegisterLayout("html", func(config LayoutBuildConfig) (Layout, error) {
		return NewHTMLLayout(config.Options), nil
	})
	_ = registry.RegisterLayout("htmlLayout", func(config LayoutBuildConfig) (Layout, error) {
		return NewHTMLLayout(config.Options), nil
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
	_ = registry.RegisterFilter("composite", buildCompositeFilterPlugin)
	_ = registry.RegisterFilter("compositeFilter", buildCompositeFilterPlugin)
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

func buildCompositeFilterPlugin(config FilterBuildConfig) (Filter, error) {
	if len(config.Filters) == 0 {
		return nil, fmt.Errorf("goark-log: filter %q composite requires filterRefs", config.Name)
	}
	return NewCompositeFilter(config.Filters...)
}

func buildMarkerFilterPlugin(config FilterBuildConfig) (Filter, error) {
	options, err := config.filterOptions()
	if err != nil {
		return nil, err
	}
	return NewMarkerFilter(textutil.FirstNonBlank(config.Marker, config.Value), options...)
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
	return NewThreadContextStackFilter(textutil.FirstNonBlank(config.Value, config.Text, config.Pattern), options...)
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
	return NewThrowableFilter(textutil.FirstNonBlank(config.Pattern, config.Text, config.Value), options...)
}

func buildStringMatchFilterPlugin(config FilterBuildConfig) (Filter, error) {
	options, err := config.filterOptions()
	if err != nil {
		return nil, err
	}
	return NewStringMatchFilter(textutil.FirstNonBlank(config.Text, config.Value, config.Pattern), options...)
}

func buildTimeFilterPlugin(config FilterBuildConfig) (Filter, error) {
	options, err := config.filterOptions()
	if err != nil {
		return nil, err
	}
	start := textutil.FirstNonBlank(config.Start, "00:00:00")
	end := textutil.FirstNonBlank(config.End, "23:59:59.999999999")
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
	level, err := ParseLevel(textutil.FirstNonBlank(config.Level, "warn"))
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
	defaultLevel, err := ParseLevel(textutil.FirstNonBlank(config.DefaultThreshold, config.Level, "error"))
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
