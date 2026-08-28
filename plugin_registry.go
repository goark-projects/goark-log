package goarklog

import (
	"sync"

	internalplugin "goark.dev/log/internal/plugin"
)

// AppenderFactory 从配置构建 Appender。
type AppenderFactory = internalplugin.AppenderFactory

// LayoutFactory 从配置构建 Layout。
type LayoutFactory = internalplugin.LayoutFactory

// FilterFactory 从配置构建 Filter。
type FilterFactory = internalplugin.FilterFactory

// AppenderBuildConfig 是 appender 插件的构建输入。
type AppenderBuildConfig = internalplugin.AppenderBuildConfig

// RewriteBuildConfig 是 rewrite appender 的内置重写策略配置。
type RewriteBuildConfig = internalplugin.RewriteBuildConfig

// RollingBuildConfig 是滚动文件插件的构建输入。
type RollingBuildConfig = internalplugin.RollingBuildConfig

// RollingDeleteBuildConfig 是 YAML 删除动作的中间配置。
type RollingDeleteBuildConfig = internalplugin.RollingDeleteBuildConfig

// LayoutBuildConfig 是 layout 插件的构建输入。
type LayoutBuildConfig = internalplugin.LayoutBuildConfig

// FilterBuildConfig 是 filter 插件的构建输入。
type FilterBuildConfig = internalplugin.FilterBuildConfig

// PluginRegistrar 把一个外部模块的插件注册到指定注册表。
type PluginRegistrar = internalplugin.Registrar

// PluginRegistrarFunc 把函数适配为 PluginRegistrar。
type PluginRegistrarFunc = internalplugin.RegistrarFunc

// PluginRegistry 保存显式注册的日志插件。
type PluginRegistry = internalplugin.Registry

// PluginSet 是一组显式声明的 goark-log 插件注册项。
type PluginSet = internalplugin.PluginSet

// PluginSetOption 向 PluginSet 追加插件注册项。
type PluginSetOption = internalplugin.PluginSetOption

var (
	defaultRegistryOnce sync.Once
	defaultRegistry     *PluginRegistry
)

// NewPluginRegistry 创建包含内置插件的新注册表。
func NewPluginRegistry() *PluginRegistry {
	registry := internalplugin.NewRegistry()
	internalplugin.RegisterBuiltIns(registry)
	return registry
}

// DefaultPluginRegistry 返回进程默认插件注册表。
func DefaultPluginRegistry() *PluginRegistry {
	defaultRegistryOnce.Do(func() {
		defaultRegistry = NewPluginRegistry()
	})
	return defaultRegistry
}

// RegisterAppender 向默认插件注册表注册 appender。
func RegisterAppender(kind string, factory AppenderFactory) error {
	return DefaultPluginRegistry().RegisterAppender(kind, factory)
}

// RegisterPlugins 向默认插件注册表批量注册外部插件。
func RegisterPlugins(registrars ...PluginRegistrar) error {
	return DefaultPluginRegistry().RegisterPlugins(registrars...)
}

// RegisterLayout 向默认插件注册表注册 layout。
func RegisterLayout(kind string, factory LayoutFactory) error {
	return DefaultPluginRegistry().RegisterLayout(kind, factory)
}

// RegisterFilter 向默认插件注册表注册 filter。
func RegisterFilter(kind string, factory FilterFactory) error {
	return DefaultPluginRegistry().RegisterFilter(kind, factory)
}

// RegisterLookup 向默认插件注册表注册配置 lookup。
func RegisterLookup(namespace string, lookup LookupFunc) error {
	return DefaultPluginRegistry().RegisterLookup(namespace, lookup)
}

// RegisterJSONTemplateResolver 向默认插件注册表注册 JSON Template resolver。
func RegisterJSONTemplateResolver(kind string, factory JSONTemplateResolverFactory) error {
	return DefaultPluginRegistry().RegisterJSONTemplateResolver(kind, factory)
}

// NewPluginSet 创建可直接作为 PluginRegistrar 使用的插件集合。
func NewPluginSet(options ...PluginSetOption) PluginSet {
	return internalplugin.NewPluginSet(options...)
}

// WithPluginAppender 声明一个 appender 插件注册项。
func WithPluginAppender(kind string, factory AppenderFactory) PluginSetOption {
	return internalplugin.WithPluginAppender(kind, factory)
}

// WithPluginLayout 声明一个 layout 插件注册项。
func WithPluginLayout(kind string, factory LayoutFactory) PluginSetOption {
	return internalplugin.WithPluginLayout(kind, factory)
}

// WithPluginFilter 声明一个 filter 插件注册项。
func WithPluginFilter(kind string, factory FilterFactory) PluginSetOption {
	return internalplugin.WithPluginFilter(kind, factory)
}

// WithPluginLookup 声明一个配置 lookup 注册项。
func WithPluginLookup(namespace string, lookup LookupFunc) PluginSetOption {
	return internalplugin.WithPluginLookup(namespace, lookup)
}

// WithPluginJSONTemplateResolver 声明一个 JSON Template resolver 注册项。
func WithPluginJSONTemplateResolver(kind string, factory JSONTemplateResolverFactory) PluginSetOption {
	return internalplugin.WithPluginJSONTemplateResolver(kind, factory)
}
