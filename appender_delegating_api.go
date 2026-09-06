package goarklog

import internaldelegate "goark.dev/log/internal/delegating"

// FailoverAppender 在主 appender 写入失败时按顺序尝试备用 appender。
type FailoverAppender = internaldelegate.FailoverAppender

// FailoverOption 调整 FailoverAppender。
type FailoverOption = internaldelegate.FailoverOption

// RouteKeyFunc 从事件中计算路由键。
type RouteKeyFunc = internaldelegate.RouteKeyFunc

// RoutingAppender 按事件属性或自定义函数选择下游 appender。
type RoutingAppender = internaldelegate.RoutingAppender

// RoutingOption 调整 RoutingAppender。
type RoutingOption = internaldelegate.RoutingOption

// RewritePolicy 在写出前重写事件快照。
type RewritePolicy = internaldelegate.RewritePolicy

// RewriteAppender 在写出前执行事件重写。
type RewriteAppender = internaldelegate.RewriteAppender

// RewriteOption 调整 RewriteAppender。
type RewriteOption = internaldelegate.RewriteOption

// WithFailoverName 设置 failover appender 名称。
func WithFailoverName(name string) FailoverOption {
	return internaldelegate.WithFailoverName(name)
}

// WithFailoverCloseChildren 设置关闭 failover 时是否关闭下游 appender。
func WithFailoverCloseChildren(enabled bool) FailoverOption {
	return internaldelegate.WithFailoverCloseChildren(enabled)
}

// NewFailoverAppender 创建失败转移 appender。
func NewFailoverAppender(primary Appender, failovers []Appender, options ...FailoverOption) (*FailoverAppender, error) {
	return internaldelegate.NewFailoverAppender(primary, delegatingAppenders(failovers), options...)
}

// WithRoutingName 设置 routing appender 名称。
func WithRoutingName(name string) RoutingOption {
	return internaldelegate.WithRoutingName(name)
}

// WithRoutingAttrKey 设置按事件属性取路由键。
func WithRoutingAttrKey(key string) RoutingOption {
	return internaldelegate.WithRoutingAttrKey(key)
}

// WithRoutingDefault 设置未命中路由时的默认 appender。
func WithRoutingDefault(route Appender) RoutingOption {
	return internaldelegate.WithRoutingDefault(route)
}

// WithRoutingKeyFunc 设置自定义路由键函数。
func WithRoutingKeyFunc(routeKeyFunc RouteKeyFunc) RoutingOption {
	return internaldelegate.WithRoutingKeyFunc(routeKeyFunc)
}

// WithRoutingCloseChildren 设置关闭 routing 时是否关闭下游 appender。
func WithRoutingCloseChildren(enabled bool) RoutingOption {
	return internaldelegate.WithRoutingCloseChildren(enabled)
}

// NewRoutingAppender 创建路由 appender。
func NewRoutingAppender(routes map[string]Appender, options ...RoutingOption) (*RoutingAppender, error) {
	return internaldelegate.NewRoutingAppender(delegatingAppenderMap(routes), options...)
}

// WithRewriteName 设置 rewrite appender 名称。
func WithRewriteName(name string) RewriteOption {
	return internaldelegate.WithRewriteName(name)
}

// WithRewriteCloseDelegate 设置关闭 rewrite 时是否关闭下游 appender。
func WithRewriteCloseDelegate(enabled bool) RewriteOption {
	return internaldelegate.WithRewriteCloseDelegate(enabled)
}

// NewRewriteAppender 创建事件重写 appender。
func NewRewriteAppender(delegate Appender, policy RewritePolicy, options ...RewriteOption) (*RewriteAppender, error) {
	return internaldelegate.NewRewriteAppender(delegate, policy, options...)
}

func delegatingAppenders(appenders []Appender) []internaldelegate.Appender {
	if len(appenders) == 0 {
		return nil
	}
	converted := make([]internaldelegate.Appender, 0, len(appenders))
	for _, appender := range appenders {
		converted = append(converted, appender)
	}
	return converted
}

func delegatingAppenderMap(routes map[string]Appender) map[string]internaldelegate.Appender {
	if len(routes) == 0 {
		return nil
	}
	converted := make(map[string]internaldelegate.Appender, len(routes))
	for key, appender := range routes {
		converted[key] = appender
	}
	return converted
}
