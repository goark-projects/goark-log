package goarklog

import (
	"fmt"
	"strings"
	"sync"

	"goark.dev/log/internal/lookupguard"
	"goark.dev/log/internal/textutil"
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
