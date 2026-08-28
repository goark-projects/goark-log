package configfile

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"time"

	internalasyncappender "goark.dev/log/internal/asyncappender"
	internalasync "goark.dev/log/internal/asyncruntime"
	"goark.dev/log/internal/configvalue"
	internalfileappender "goark.dev/log/internal/fileappender"
	logfilter "goark.dev/log/internal/filter"
	internallayout "goark.dev/log/internal/layout"
	configlevel "goark.dev/log/internal/level"
	"goark.dev/log/internal/lookup"
	internalplugin "goark.dev/log/internal/plugin"
	internalrouter "goark.dev/log/internal/router"
)

type Appender = internalrouter.Appender
type AppenderRef = internalrouter.AppenderRef
type Filter = logfilter.Filter
type Layout = internallayout.Layout
type LayoutOptions = internallayout.LayoutOptions
type AsyncOverflowStrategy = internalasync.OverflowStrategy
type AsyncWaitStrategy = internalasync.WaitStrategy
type AsyncWaitOptions = internalasync.WaitOptions
type AsyncLoggerOptions = internalasync.LoggerOptions
type PluginRegistry = internalplugin.Registry
type AppenderBuildConfig = internalplugin.AppenderBuildConfig
type LayoutBuildConfig = internalplugin.LayoutBuildConfig
type FilterBuildConfig = internalplugin.FilterBuildConfig
type RewriteBuildConfig = internalplugin.RewriteBuildConfig
type RollingBuildConfig = internalplugin.RollingBuildConfig
type RollingDeleteBuildConfig = internalplugin.RollingDeleteBuildConfig
type LookupResolver = lookup.Resolver
type RootLogger = internalrouter.RootLogger
type LoggerRule = internalrouter.LoggerRule

// Config 是配置文件解析后的内部模型，不属于公共 API。
type Config = fileConfig

// LayoutConfig 是布局插件构建所需的内部配置片段。
type LayoutConfig = layoutConfig

// Options 是配置文件构建出的运行期配置，根包会转换成公共 Options。
type Options struct {
	Appenders []Appender
	Filters   []Filter
	Root      RootLogger
	Loggers   []LoggerRule
	Async     AsyncLoggerOptions
}

var (
	defaultRegistryOnce sync.Once
	defaultRegistry     *PluginRegistry
)

// Load 读取并解析一个配置文件。
func Load(ctx context.Context, path string, lookups *LookupResolver) (*Config, error) {
	return loadConfigFile(ctx, path, lookups)
}

// Decode 按指定格式解析配置。
func Decode(reader io.Reader, format string, lookups *LookupResolver) (*Config, error) {
	return decodeConfig(reader, format, lookups)
}

// DecodeStructured 解析 YAML 或 JSON 配置。
func DecodeStructured(reader io.Reader, lookups *LookupResolver) (*Config, error) {
	return decodeStructuredConfig(reader, lookups)
}

// DecodeXML 解析 Log4j2 风格 XML 配置。
func DecodeXML(reader io.Reader, lookups *LookupResolver) (*Config, error) {
	return decodeXMLConfig(reader, lookups)
}

// DecodeProperties 解析 properties 配置。
func DecodeProperties(reader io.Reader, lookups *LookupResolver) (*Config, error) {
	return decodePropertiesConfig(reader, lookups)
}

// Options 构建 Handler 运行期配置。
func (c *fileConfig) Options(registry *PluginRegistry) (Options, error) {
	return c.options(registry)
}

// MonitorIntervalDuration 返回配置文件监控间隔。
func (c *fileConfig) MonitorIntervalDuration() (time.Duration, error) {
	return c.monitorInterval()
}

// BuildFilters 构建并返回具名过滤器表。
func (c *fileConfig) BuildFilters(registry *PluginRegistry) (map[string]Filter, error) {
	return c.buildFilters(registry)
}

// BuildLayout 构建布局，供配置测试覆盖插件注册路径。
func BuildLayout(config LayoutConfig, registry *PluginRegistry) (Layout, error) {
	return buildLayout(config, registry)
}

func NewLookupResolver() *LookupResolver {
	return lookup.NewResolver()
}

func NewPluginRegistry() *PluginRegistry {
	registry := internalplugin.NewRegistry()
	internalplugin.RegisterBuiltIns(registry)
	return registry
}

func DefaultPluginRegistry() *PluginRegistry {
	defaultRegistryOnce.Do(func() {
		defaultRegistry = NewPluginRegistry()
	})
	return defaultRegistry
}

func DefaultOptions() Options {
	return Options{
		Appenders: []Appender{internalfileappender.NewConsoleAppender()},
		Root: RootLogger{
			Level:        slog.LevelInfo,
			AppenderRefs: []string{"console"},
		},
	}
}

func RegisterLevel(name string, level slog.Level) error {
	return configlevel.RegisterDefault(name, level)
}

func ParseLevel(value string) (slog.Level, error) {
	return configlevel.ParseDefault(value)
}

func ParseMonitorInterval(value string) (time.Duration, error) {
	return configvalue.MonitorInterval(value)
}

func ParseAsyncOverflowStrategy(value string) (AsyncOverflowStrategy, error) {
	return internalasync.ParseOverflowStrategy(value)
}

func ParseAsyncWaitStrategy(value string) (AsyncWaitStrategy, error) {
	return internalasync.ParseWaitStrategy(value)
}

func validateAsyncWaitOptions(options AsyncWaitOptions) error {
	return internalasync.ValidateWaitOptions(options)
}

func NewFilteredAppender(delegate Appender, filters ...Filter) (*internalrouter.FilteredAppender, error) {
	return internalrouter.NewFilteredAppender(delegate, filters...)
}

func closeAppenderList(appenders []Appender) error {
	return internalrouter.CloseAppenders(appenders, isAsyncAppender)
}

func isAsyncAppender(appender Appender) bool {
	switch value := appender.(type) {
	case *internalasyncappender.Appender:
		return true
	case *internalrouter.FilteredAppender:
		return isAsyncAppender(value.Delegate())
	default:
		return false
	}
}

func configFormat(path string) (string, error) {
	switch strings.ToLower(strings.TrimPrefix(filepath.Ext(path), ".")) {
	case "yml", "yaml":
		return "yaml", nil
	case "json":
		return "json", nil
	case "xml":
		return "xml", nil
	case "toml":
		return "toml", nil
	case "properties":
		return "properties", nil
	default:
		return "", unsupportedConfigFormatError(path)
	}
}

func unsupportedConfigFormatError(path string) error {
	return fmt.Errorf("goark-log: unsupported config file extension for %q", path)
}
