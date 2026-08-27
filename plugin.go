package goarklog

// AppenderFactory 从配置构建 Appender。
type AppenderFactory func(config AppenderBuildConfig) (Appender, error)

// LayoutFactory 从配置构建 Layout。
type LayoutFactory func(config LayoutBuildConfig) (Layout, error)

// FilterFactory 从配置构建 Filter。
type FilterFactory func(config FilterBuildConfig) (Filter, error)

// PluginRegistrar 把一个外部模块的插件注册到指定注册表。
type PluginRegistrar interface {
	RegisterLogPlugins(registry *PluginRegistry) error
}

// PluginRegistrarFunc 把函数适配为 PluginRegistrar。
type PluginRegistrarFunc func(registry *PluginRegistry) error

// RegisterLogPlugins 执行插件注册函数。
func (f PluginRegistrarFunc) RegisterLogPlugins(registry *PluginRegistry) error {
	if f == nil {
		return nil
	}
	return f(registry)
}
