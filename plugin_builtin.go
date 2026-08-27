package goarklog

import (
	"strings"
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
