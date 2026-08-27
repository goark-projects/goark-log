package goarklog

import (
	"fmt"
	"strings"
	"sync"

	"goark.dev/log/internal/lookupguard"
	"goark.dev/log/internal/textutil"
)

// PluginRegistry 保存显式注册的日志插件。
type PluginRegistry struct {
	mu        sync.RWMutex
	appenders map[string]AppenderFactory
	layouts   map[string]LayoutFactory
	filters   map[string]FilterFactory
	lookups   map[string]LookupFunc
	resolvers map[string]JSONTemplateResolverFactory
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
		resolvers: make(map[string]JSONTemplateResolverFactory),
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

// RegisterPlugins 把一组外部插件注册到当前注册表。
func (r *PluginRegistry) RegisterPlugins(registrars ...PluginRegistrar) error {
	if r == nil {
		return fmt.Errorf("goark-log: plugin registry is nil")
	}
	for index, registrar := range registrars {
		if registrar == nil {
			continue
		}
		if err := registrar.RegisterLogPlugins(r); err != nil {
			return fmt.Errorf("goark-log: plugin registrar %d: %w", index, err)
		}
	}
	return nil
}

// RegisterAppender 注册 appender 插件。
func (r *PluginRegistry) RegisterAppender(kind string, factory AppenderFactory) error {
	if r == nil {
		return fmt.Errorf("goark-log: plugin registry is nil")
	}
	if factory == nil {
		return fmt.Errorf("goark-log: appender factory is nil")
	}
	kind = textutil.NormalizeKind(kind)
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
	kind = textutil.NormalizeKind(kind)
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
	kind = textutil.NormalizeKind(kind)
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
	if lookupguard.BlockedNamespace(namespace) {
		return fmt.Errorf("goark-log: lookup namespace %q is blocked by security policy", namespace)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lookups[namespace] = lookup
	return nil
}

// RegisterJSONTemplateResolver 注册 JSON Template resolver 插件。
func (r *PluginRegistry) RegisterJSONTemplateResolver(kind string, factory JSONTemplateResolverFactory) error {
	if r == nil {
		return fmt.Errorf("goark-log: plugin registry is nil")
	}
	if factory == nil {
		return fmt.Errorf("goark-log: JSON template resolver factory is nil")
	}
	kind = textutil.NormalizeKind(kind)
	if kind == "" {
		return fmt.Errorf("goark-log: JSON template resolver kind is empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.resolvers[kind] = factory
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
	factory, ok := r.appenders[textutil.NormalizeKind(kind)]
	return factory, ok
}

func (r *PluginRegistry) layoutFactory(kind string) (LayoutFactory, bool) {
	if r == nil {
		r = DefaultPluginRegistry()
	}
	kind = textutil.NormalizeKind(kind)
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
	factory, ok := r.filters[textutil.NormalizeKind(kind)]
	return factory, ok
}

func (r *PluginRegistry) jsonTemplateResolverFactory(kind string) (JSONTemplateResolverFactory, bool) {
	if r == nil {
		r = DefaultPluginRegistry()
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	factory, ok := r.resolvers[textutil.NormalizeKind(kind)]
	return factory, ok
}
