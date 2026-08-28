package plugin

import (
	"fmt"
	"strings"
	"sync"

	internallayout "goark.dev/log/internal/layout"
	"goark.dev/log/internal/lookup"
	"goark.dev/log/internal/lookupguard"
	"goark.dev/log/internal/textutil"
)

// Registry 保存显式注册的日志插件。
type Registry struct {
	mu        sync.RWMutex
	appenders map[string]AppenderFactory
	layouts   map[string]LayoutFactory
	filters   map[string]FilterFactory
	lookups   map[string]LookupFunc
	resolvers map[string]internallayout.JSONTemplateResolverFactory
}

// NewRegistry 创建空插件注册表。
func NewRegistry() *Registry {
	return &Registry{
		appenders: make(map[string]AppenderFactory),
		layouts:   make(map[string]LayoutFactory),
		filters:   make(map[string]FilterFactory),
		lookups:   make(map[string]LookupFunc),
		resolvers: make(map[string]internallayout.JSONTemplateResolverFactory),
	}
}

// RegisterPlugins 把一组外部插件注册到当前注册表。
func (r *Registry) RegisterPlugins(registrars ...Registrar) error {
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
func (r *Registry) RegisterAppender(kind string, factory AppenderFactory) error {
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
func (r *Registry) RegisterLayout(kind string, factory LayoutFactory) error {
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
func (r *Registry) RegisterFilter(kind string, factory FilterFactory) error {
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
func (r *Registry) RegisterLookup(namespace string, lookup LookupFunc) error {
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
func (r *Registry) RegisterJSONTemplateResolver(kind string, factory internallayout.JSONTemplateResolverFactory) error {
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

// LookupResolver 返回包含注册 lookup 的独立解析器。
func (r *Registry) LookupResolver() *lookup.Resolver {
	resolver := lookup.NewResolver()
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

// AppenderFactory 返回指定类型的 appender 工厂。
func (r *Registry) AppenderFactory(kind string) (AppenderFactory, bool) {
	if r == nil {
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	factory, ok := r.appenders[textutil.NormalizeKind(kind)]
	return factory, ok
}

// LayoutFactory 返回指定类型的 layout 工厂。
func (r *Registry) LayoutFactory(kind string) (LayoutFactory, bool) {
	if r == nil {
		return nil, false
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

// FilterFactory 返回指定类型的 filter 工厂。
func (r *Registry) FilterFactory(kind string) (FilterFactory, bool) {
	if r == nil {
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	factory, ok := r.filters[textutil.NormalizeKind(kind)]
	return factory, ok
}

// JSONTemplateResolverFactory 返回指定类型的 JSON Template resolver 工厂。
func (r *Registry) JSONTemplateResolverFactory(kind string) (internallayout.JSONTemplateResolverFactory, bool) {
	if r == nil {
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	factory, ok := r.resolvers[textutil.NormalizeKind(kind)]
	return factory, ok
}
