package plugin

import (
	"goark.dev/log/internal/asyncruntime"
	logfilter "goark.dev/log/internal/filter"
	internallayout "goark.dev/log/internal/layout"
	"goark.dev/log/internal/lookup"
	internalrouter "goark.dev/log/internal/router"
)

// AppenderFactory 从配置构建 Appender。
type AppenderFactory func(config AppenderBuildConfig) (internalrouter.Appender, error)

// LayoutFactory 从配置构建 Layout。
type LayoutFactory func(config LayoutBuildConfig) (internallayout.Layout, error)

// FilterFactory 从配置构建 Filter。
type FilterFactory func(config FilterBuildConfig) (logfilter.Filter, error)

// LookupFunc 根据键解析配置变量。
type LookupFunc = lookup.Func

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
	Layout           internallayout.Layout
	AppenderRefs     []string
	Delegates        []internalrouter.Appender
	Routes           map[string]internalrouter.Appender
	DefaultRoute     internalrouter.Appender
	RouteKey         string
	QueueSize        int
	BatchSize        int
	OverflowStrategy string
	WaitStrategy     string
	WaitOptions      asyncruntime.WaitOptions
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
	Options          internallayout.LayoutOptions
	Registry         *Registry
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
	Filters          []logfilter.Filter
	DefaultThreshold string
	Pattern          string
	OnMatch          string
	OnMismatch       string
}
