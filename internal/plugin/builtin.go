package plugin

// RegisterBuiltIns 注册 goark-log 内置 appender、layout 和 filter 插件。
func RegisterBuiltIns(registry *Registry) {
	registerBuiltInAppenders(registry)
	registerBuiltInLayouts(registry)
	registerBuiltInFilters(registry)
}

func registerBuiltInAppenders(registry *Registry) {
	registerAppenderAliases(registry, buildConsolePlugin, "console")
	registerAppenderAliases(registry, buildFilePlugin, "file")
	registerAppenderAliases(registry, buildJSONPlugin, "json", "jsonDirect", "jsonWriter")
	registerAppenderAliases(registry, buildRollingPlugin, "rolling", "rollingFile")
	registerAppenderAliases(registry, buildAsyncPlugin, "async")
	registerAppenderAliases(registry, buildFailoverPlugin, "failover", "failoverAppender")
	registerAppenderAliases(registry, buildRoutingPlugin, "routing", "routingAppender")
	registerAppenderAliases(registry, buildRewritePlugin, "rewrite", "rewriteAppender")
}

func registerAppenderAliases(registry *Registry, factory AppenderFactory, aliases ...string) {
	for _, alias := range aliases {
		_ = registry.RegisterAppender(alias, factory)
	}
}

func registerBuiltInFilters(registry *Registry) {
	registerFilterAliases(registry, buildThresholdFilterPlugin, "threshold", "thresholdFilter")
	registerFilterAliases(registry, buildLevelFilterPlugin, "level", "levelFilter")
	registerFilterAliases(registry, buildLevelRangeFilterPlugin, "levelRange", "levelRangeFilter")
	registerFilterAliases(registry, buildRegexFilterPlugin, "regex", "regexFilter")
	registerFilterAliases(registry, buildAttrFilterPlugin, "attr", "attribute", "attrFilter", "attributeFilter")
	registerFilterAliases(registry, buildDenyFilterPlugin, "deny", "denyAll", "denyFilter", "denyAllFilter")
	registerFilterAliases(registry, buildCompositeFilterPlugin, "composite", "compositeFilter")
	registerFilterAliases(registry, buildMarkerFilterPlugin, "marker", "markerFilter")
	registerFilterAliases(registry, buildNoMarkerFilterPlugin, "noMarker", "noMarkerFilter")
	registerFilterAliases(registry, buildMapFilterPlugin, "map", "mapFilter")
	registerFilterAliases(registry, buildThreadContextMapFilterPlugin, "threadContextMap", "threadContextMapFilter")
	registerFilterAliases(registry, buildThreadContextStackFilterPlugin, "threadContextStack", "threadContextStackFilter")
	registerFilterAliases(registry, buildStructuredDataFilterPlugin, "structuredData", "structuredDataFilter")
	registerFilterAliases(registry, buildThrowableFilterPlugin, "throwable", "throwableFilter")
	registerFilterAliases(registry, buildStringMatchFilterPlugin, "stringMatch", "stringMatchFilter")
	registerFilterAliases(registry, buildTimeFilterPlugin, "time", "timeFilter")
	registerFilterAliases(registry, buildBurstFilterPlugin, "burst", "burstFilter")
	registerFilterAliases(registry, buildDynamicThresholdFilterPlugin, "dynamicThreshold", "dynamicThresholdFilter")
}

func registerFilterAliases(registry *Registry, factory FilterFactory, aliases ...string) {
	for _, alias := range aliases {
		_ = registry.RegisterFilter(alias, factory)
	}
}
