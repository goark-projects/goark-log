package goarklog

// RegisterAppender 向默认插件注册表注册 appender。
func RegisterAppender(kind string, factory AppenderFactory) error {
	return DefaultPluginRegistry().RegisterAppender(kind, factory)
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
