package goarklog

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"goark.dev/log/internal/logvalue"
	"goark.dev/log/internal/textutil"
)

// FailoverAppender 在主 appender 写入失败时按顺序尝试备用 appender。
type FailoverAppender struct {
	name          string
	primary       Appender
	failovers     []Appender
	closeChildren bool
}

// FailoverOption 调整 FailoverAppender。
type FailoverOption func(*FailoverAppender)

// WithFailoverName 设置 failover appender 名称。
func WithFailoverName(name string) FailoverOption {
	return func(appender *FailoverAppender) {
		appender.name = name
	}
}

// WithFailoverCloseChildren 设置关闭 failover 时是否关闭下游 appender。
func WithFailoverCloseChildren(enabled bool) FailoverOption {
	return func(appender *FailoverAppender) {
		appender.closeChildren = enabled
	}
}

// NewFailoverAppender 创建失败转移 appender。
func NewFailoverAppender(primary Appender, failovers []Appender, options ...FailoverOption) (*FailoverAppender, error) {
	appender := &FailoverAppender{
		name:          "failover",
		primary:       primary,
		failovers:     append([]Appender(nil), failovers...),
		closeChildren: true,
	}
	for _, option := range options {
		if option != nil {
			option(appender)
		}
	}
	if strings.TrimSpace(appender.name) == "" {
		return nil, fmt.Errorf("goark-log: failover appender name is empty")
	}
	if appender.primary == nil {
		return nil, fmt.Errorf("goark-log: failover primary appender is nil")
	}
	if len(appender.failovers) == 0 {
		return nil, fmt.Errorf("goark-log: failover appender requires at least one failover")
	}
	for index, failover := range appender.failovers {
		if failover == nil {
			return nil, fmt.Errorf("goark-log: failover appender %d is nil", index)
		}
	}
	return appender, nil
}

func (a *FailoverAppender) Name() string {
	if a == nil || a.name == "" {
		return "failover"
	}
	return a.name
}

func (a *FailoverAppender) Append(ctx context.Context, event Event) error {
	if a == nil || a.primary == nil {
		return fmt.Errorf("goark-log: failover appender is nil")
	}
	ctx = normalizeContext(ctx)
	primaryErr := a.primary.Append(ctx, event)
	if primaryErr == nil {
		return nil
	}
	var joined error = primaryErr
	for _, failover := range a.failovers {
		if err := failover.Append(ctx, event); err == nil {
			return nil
		} else {
			joined = errors.Join(joined, err)
		}
	}
	return joined
}

func (a *FailoverAppender) Close() error {
	if a == nil || !a.closeChildren {
		return nil
	}
	return closeUniqueAppenders(append([]Appender{a.primary}, a.failovers...))
}

// RouteKeyFunc 从事件中计算路由键。
type RouteKeyFunc func(ctx context.Context, event Event) string

// RoutingAppender 按事件属性或自定义函数选择下游 appender。
type RoutingAppender struct {
	name          string
	routes        map[string]Appender
	defaultRoute  Appender
	attrKey       string
	routeKeyFunc  RouteKeyFunc
	closeChildren bool
}

// RoutingOption 调整 RoutingAppender。
type RoutingOption func(*RoutingAppender)

// WithRoutingName 设置 routing appender 名称。
func WithRoutingName(name string) RoutingOption {
	return func(appender *RoutingAppender) {
		appender.name = name
	}
}

// WithRoutingAttrKey 设置按事件属性取路由键。
func WithRoutingAttrKey(key string) RoutingOption {
	return func(appender *RoutingAppender) {
		appender.attrKey = key
	}
}

// WithRoutingDefault 设置未命中路由时的默认 appender。
func WithRoutingDefault(route Appender) RoutingOption {
	return func(appender *RoutingAppender) {
		appender.defaultRoute = route
	}
}

// WithRoutingKeyFunc 设置自定义路由键函数。
func WithRoutingKeyFunc(routeKeyFunc RouteKeyFunc) RoutingOption {
	return func(appender *RoutingAppender) {
		appender.routeKeyFunc = routeKeyFunc
	}
}

// WithRoutingCloseChildren 设置关闭 routing 时是否关闭下游 appender。
func WithRoutingCloseChildren(enabled bool) RoutingOption {
	return func(appender *RoutingAppender) {
		appender.closeChildren = enabled
	}
}

// NewRoutingAppender 创建路由 appender。
func NewRoutingAppender(routes map[string]Appender, options ...RoutingOption) (*RoutingAppender, error) {
	appender := &RoutingAppender{
		name:    "routing",
		attrKey: "route",
		routes:  make(map[string]Appender, len(routes)),
	}
	for key, route := range routes {
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("goark-log: routing route key is empty")
		}
		if route == nil {
			return nil, fmt.Errorf("goark-log: routing route %q appender is nil", key)
		}
		appender.routes[key] = route
	}
	for _, option := range options {
		if option != nil {
			option(appender)
		}
	}
	if strings.TrimSpace(appender.name) == "" {
		return nil, fmt.Errorf("goark-log: routing appender name is empty")
	}
	if len(appender.routes) == 0 && appender.defaultRoute == nil {
		return nil, fmt.Errorf("goark-log: routing appender requires routes or default route")
	}
	return appender, nil
}

func (a *RoutingAppender) Name() string {
	if a == nil || a.name == "" {
		return "routing"
	}
	return a.name
}

func (a *RoutingAppender) Append(ctx context.Context, event Event) error {
	if a == nil {
		return fmt.Errorf("goark-log: routing appender is nil")
	}
	ctx = normalizeContext(ctx)
	key := a.routeKey(ctx, event)
	appender := a.routes[key]
	if appender == nil {
		appender = a.defaultRoute
	}
	if appender == nil {
		return nil
	}
	return appender.Append(ctx, event)
}

func (a *RoutingAppender) Close() error {
	if a == nil || !a.closeChildren {
		return nil
	}
	appenders := make([]Appender, 0, len(a.routes)+1)
	if a.defaultRoute != nil {
		appenders = append(appenders, a.defaultRoute)
	}
	for _, appender := range a.routes {
		appenders = append(appenders, appender)
	}
	return closeUniqueAppenders(appenders)
}

func (a *RoutingAppender) routeKey(ctx context.Context, event Event) string {
	if a.routeKeyFunc != nil {
		return strings.TrimSpace(a.routeKeyFunc(ctx, event))
	}
	if strings.TrimSpace(a.attrKey) == "" {
		return ""
	}
	value, ok := event.Attr(a.attrKey)
	if !ok {
		return ""
	}
	return strings.TrimSpace(logvalue.String(value))
}

// RewritePolicy 在写出前重写事件快照。
type RewritePolicy func(ctx context.Context, event Event) (Event, error)

type rewriteBuildConfig struct {
	Attrs            map[string]string `yaml:"attrs"`
	Attributes       map[string]string `yaml:"attributes"`
	Properties       map[string]string `yaml:"properties"`
	Remove           []string          `yaml:"remove"`
	RemoveAttrs      []string          `yaml:"removeAttrs"`
	RemoveAttrsKebab []string          `yaml:"remove-attrs"`
}

func (c rewriteBuildConfig) attrs() map[string]string {
	attrs := mergeStringMaps(copyStringMap(c.Attrs), c.Attributes)
	return mergeStringMaps(attrs, c.Properties)
}

func (c rewriteBuildConfig) removeKeys() []string {
	return textutil.FirstTrimmedStrings(c.Remove, c.RemoveAttrs, c.RemoveAttrsKebab)
}

func (c *rewriteBuildConfig) resolveLookups(lookups *LookupResolver) error {
	var err error
	if c.Attrs, err = resolveStringMapLookups(lookups, c.Attrs); err != nil {
		return err
	}
	if c.Attributes, err = resolveStringMapLookups(lookups, c.Attributes); err != nil {
		return err
	}
	if c.Properties, err = resolveStringMapLookups(lookups, c.Properties); err != nil {
		return err
	}
	if c.Remove, err = resolveStringListLookups(lookups, c.Remove); err != nil {
		return err
	}
	if c.RemoveAttrs, err = resolveStringListLookups(lookups, c.RemoveAttrs); err != nil {
		return err
	}
	if c.RemoveAttrsKebab, err = resolveStringListLookups(lookups, c.RemoveAttrsKebab); err != nil {
		return err
	}
	return nil
}

// RewriteAppender 在写出前执行事件重写。
type RewriteAppender struct {
	name          string
	delegate      Appender
	policy        RewritePolicy
	closeDelegate bool
}

// RewriteOption 调整 RewriteAppender。
type RewriteOption func(*RewriteAppender)

// WithRewriteName 设置 rewrite appender 名称。
func WithRewriteName(name string) RewriteOption {
	return func(appender *RewriteAppender) {
		appender.name = name
	}
}

// WithRewriteCloseDelegate 设置关闭 rewrite 时是否关闭下游 appender。
func WithRewriteCloseDelegate(enabled bool) RewriteOption {
	return func(appender *RewriteAppender) {
		appender.closeDelegate = enabled
	}
}

// NewRewriteAppender 创建事件重写 appender。
func NewRewriteAppender(delegate Appender, policy RewritePolicy, options ...RewriteOption) (*RewriteAppender, error) {
	appender := &RewriteAppender{
		name:          "rewrite",
		delegate:      delegate,
		policy:        policy,
		closeDelegate: true,
	}
	for _, option := range options {
		if option != nil {
			option(appender)
		}
	}
	if strings.TrimSpace(appender.name) == "" {
		return nil, fmt.Errorf("goark-log: rewrite appender name is empty")
	}
	if appender.delegate == nil {
		return nil, fmt.Errorf("goark-log: rewrite delegate appender is nil")
	}
	return appender, nil
}

func (a *RewriteAppender) Name() string {
	if a == nil || a.name == "" {
		return "rewrite"
	}
	return a.name
}

func (a *RewriteAppender) Append(ctx context.Context, event Event) error {
	if a == nil || a.delegate == nil {
		return fmt.Errorf("goark-log: rewrite appender is nil")
	}
	ctx = normalizeContext(ctx)
	if a.policy != nil {
		rewritten, err := a.policy(ctx, event)
		if err != nil {
			return err
		}
		event = rewritten
	}
	return a.delegate.Append(ctx, event)
}

func (a *RewriteAppender) Close() error {
	if a == nil || a.delegate == nil || !a.closeDelegate {
		return nil
	}
	return a.delegate.Close()
}

func newAttributeRewritePolicy(config RewriteBuildConfig) RewritePolicy {
	additions := rewriteAdditions(config.Attrs)
	removals := rewriteRemovalSet(config.RemoveAttrs)
	if len(additions) == 0 && len(removals) == 0 {
		return nil
	}
	return func(_ context.Context, event Event) (Event, error) {
		rewritten := event
		attrs := make([]slog.Attr, 0, len(event.Attrs)+len(additions))
		for _, attr := range event.Attrs {
			if _, remove := removals[attr.Key]; remove {
				continue
			}
			attrs = append(attrs, attr)
		}
		attrs = append(attrs, additions...)
		rewritten.Attrs = attrs
		return rewritten, nil
	}
}

func rewriteAdditions(values map[string]string) []slog.Attr {
	if len(values) == 0 {
		return nil
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		key = strings.TrimSpace(key)
		if key != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	attrs := make([]slog.Attr, 0, len(keys))
	for _, key := range keys {
		attrs = append(attrs, slog.String(key, values[key]))
	}
	return attrs
}

func rewriteRemovalSet(keys []string) map[string]struct{} {
	if len(keys) == 0 {
		return nil
	}
	removals := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key != "" {
			removals[key] = struct{}{}
		}
	}
	return removals
}

func closeUniqueAppenders(appenders []Appender) error {
	var joined error
	closed := make(map[string]struct{}, len(appenders))
	for _, appender := range appenders {
		if appender == nil {
			continue
		}
		name := appender.Name()
		if _, exists := closed[name]; exists {
			continue
		}
		closed[name] = struct{}{}
		joined = errors.Join(joined, appender.Close())
	}
	return joined
}
