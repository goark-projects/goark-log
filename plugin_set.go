package goarklog

import "fmt"

// PluginSet 是一组显式声明的 goark-log 插件注册项。
type PluginSet struct {
	appenders []appenderPlugin
	layouts   []layoutPlugin
	filters   []filterPlugin
	lookups   []lookupPlugin
	resolvers []jsonTemplateResolverPlugin
}

type appenderPlugin struct {
	kind    string
	factory AppenderFactory
}

type layoutPlugin struct {
	kind    string
	factory LayoutFactory
}

type filterPlugin struct {
	kind    string
	factory FilterFactory
}

type lookupPlugin struct {
	namespace string
	lookup    LookupFunc
}

type jsonTemplateResolverPlugin struct {
	kind    string
	factory JSONTemplateResolverFactory
}

// PluginSetOption 向 PluginSet 追加插件注册项。
type PluginSetOption func(*PluginSet)

// NewPluginSet 创建可直接作为 PluginRegistrar 使用的插件集合。
func NewPluginSet(options ...PluginSetOption) PluginSet {
	set := PluginSet{}
	for _, option := range options {
		if option != nil {
			option(&set)
		}
	}
	return set
}

// WithPluginAppender 声明一个 appender 插件注册项。
func WithPluginAppender(kind string, factory AppenderFactory) PluginSetOption {
	return func(set *PluginSet) {
		set.appenders = append(set.appenders, appenderPlugin{kind: kind, factory: factory})
	}
}

// WithPluginLayout 声明一个 layout 插件注册项。
func WithPluginLayout(kind string, factory LayoutFactory) PluginSetOption {
	return func(set *PluginSet) {
		set.layouts = append(set.layouts, layoutPlugin{kind: kind, factory: factory})
	}
}

// WithPluginFilter 声明一个 filter 插件注册项。
func WithPluginFilter(kind string, factory FilterFactory) PluginSetOption {
	return func(set *PluginSet) {
		set.filters = append(set.filters, filterPlugin{kind: kind, factory: factory})
	}
}

// WithPluginLookup 声明一个配置 lookup 注册项。
func WithPluginLookup(namespace string, lookup LookupFunc) PluginSetOption {
	return func(set *PluginSet) {
		set.lookups = append(set.lookups, lookupPlugin{namespace: namespace, lookup: lookup})
	}
}

// WithPluginJSONTemplateResolver 声明一个 JSON Template resolver 注册项。
func WithPluginJSONTemplateResolver(kind string, factory JSONTemplateResolverFactory) PluginSetOption {
	return func(set *PluginSet) {
		set.resolvers = append(set.resolvers, jsonTemplateResolverPlugin{kind: kind, factory: factory})
	}
}

// RegisterLogPlugins 把插件集合注册到指定注册表。
func (s PluginSet) RegisterLogPlugins(registry *PluginRegistry) error {
	if registry == nil {
		return fmt.Errorf("goark-log: plugin registry is nil")
	}
	for _, plugin := range s.appenders {
		if err := registry.RegisterAppender(plugin.kind, plugin.factory); err != nil {
			return fmt.Errorf("goark-log: register appender plugin %q: %w", plugin.kind, err)
		}
	}
	for _, plugin := range s.layouts {
		if err := registry.RegisterLayout(plugin.kind, plugin.factory); err != nil {
			return fmt.Errorf("goark-log: register layout plugin %q: %w", plugin.kind, err)
		}
	}
	for _, plugin := range s.filters {
		if err := registry.RegisterFilter(plugin.kind, plugin.factory); err != nil {
			return fmt.Errorf("goark-log: register filter plugin %q: %w", plugin.kind, err)
		}
	}
	for _, plugin := range s.lookups {
		if err := registry.RegisterLookup(plugin.namespace, plugin.lookup); err != nil {
			return fmt.Errorf("goark-log: register lookup plugin %q: %w", plugin.namespace, err)
		}
	}
	for _, plugin := range s.resolvers {
		if err := registry.RegisterJSONTemplateResolver(plugin.kind, plugin.factory); err != nil {
			return fmt.Errorf("goark-log: register JSON template resolver plugin %q: %w", plugin.kind, err)
		}
	}
	return nil
}
