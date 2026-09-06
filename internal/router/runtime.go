package router

import (
	"fmt"
	"sort"
	"strings"

	logfilter "goark.dev/log/internal/filter"
)

func buildRuntimeConfig(options Options) (*runtimeConfig, error) {
	if len(options.Appenders) == 0 {
		return nil, fmt.Errorf("goark-log: requires at least one appender")
	}
	globalFilters, err := logfilter.Normalize("global", options.Filters)
	if err != nil {
		return nil, err
	}
	appenderByName := make(map[string]Appender, len(options.Appenders))
	all := make([]Appender, 0, len(options.Appenders))
	for _, appender := range options.Appenders {
		if appender == nil {
			return nil, fmt.Errorf("goark-log: appender is nil")
		}
		name := strings.TrimSpace(appender.Name())
		if name == "" {
			return nil, fmt.Errorf("goark-log: appender name is empty")
		}
		if _, exists := appenderByName[name]; exists {
			return nil, fmt.Errorf("goark-log: duplicate appender %q", name)
		}
		appenderByName[name] = appender
		all = append(all, appender)
	}
	rootRefs := mergeAppenderRefs(options.Root.AppenderRefs, options.Root.AppenderRefControls)
	if len(rootRefs) == 0 {
		rootRefs = []AppenderRef{{Ref: all[0].Name()}}
	}
	rootAppenders, err := resolveAppenderControls(appenderByName, rootRefs)
	if err != nil {
		return nil, err
	}
	rootFilters, err := logfilter.Normalize("root", options.Root.Filters)
	if err != nil {
		return nil, err
	}
	config := &runtimeConfig{
		root: Route{
			Level:           options.Root.Level,
			Appenders:       rootAppenders,
			Filters:         rootFilters,
			IncludeLocation: routeRequiresLocation(options.Root.IncludeLocation, rootAppenders),
		},
		globalFilters:   globalFilters,
		all:             all,
		isAsyncAppender: options.IsAsyncAppender,
	}
	config.includeLocation = config.root.IncludeLocation
	for _, rule := range options.Loggers {
		err := appendLoggerRuntime(config, appenderByName, rootFilters, options.Root, rule)
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(config.loggers, func(i, j int) bool {
		return loggerSpecificity(config.loggers[i].name) > loggerSpecificity(config.loggers[j].name)
	})
	for index := range config.loggers {
		config.loggers[index].route.Level = effectiveLoggerLevel(
			config.root.Level,
			config.loggers[index].name,
			config.loggers,
		)
	}
	return config, nil
}

func appendLoggerRuntime(
	config *runtimeConfig,
	appenderByName map[string]Appender,
	rootFilters []Filter,
	root RootLogger,
	rule LoggerRule,
) error {
	name := strings.TrimSpace(rule.Name)
	if name == "" {
		return fmt.Errorf("goark-log: logger name is empty")
	}
	additivity := true
	if rule.AdditivitySet {
		additivity = rule.Additivity
	}
	appenders, err := resolveAppenderControls(
		appenderByName,
		mergeAppenderRefs(rule.AppenderRefs, rule.AppenderRefControls),
	)
	if err != nil {
		return fmt.Errorf("goark-log: logger %q: %w", name, err)
	}
	if !additivity && len(appenders) == 0 {
		return fmt.Errorf("goark-log: logger %q disables additivity but has no appender", name)
	}
	filters, err := logfilter.Normalize("logger "+name, rule.Filters)
	if err != nil {
		return err
	}
	effectiveFilters := append([]Filter(nil), filters...)
	if additivity {
		appenders = appendUniqueAppenderControls(appenders, config.root.Appenders)
		effectiveFilters = logfilter.Append(effectiveFilters, rootFilters)
	}
	includeLocation := root.IncludeLocation
	if rule.IncludeLocation != nil {
		includeLocation = *rule.IncludeLocation
	}
	loggerRoute := Route{
		Level:           loggerLevel(root.Level, rule.Level),
		Appenders:       appenders,
		Filters:         effectiveFilters,
		IncludeLocation: routeRequiresLocation(includeLocation, appenders),
	}
	if loggerRoute.IncludeLocation {
		config.includeLocation = true
	}
	config.loggers = append(config.loggers, loggerRuntime{
		name:            name,
		configuredLevel: cloneLevel(rule.Level),
		route:           loggerRoute,
	})
	return nil
}

func routeRequiresLocation(includeLocation bool, appenders []AppenderControl) bool {
	if includeLocation {
		return true
	}
	for _, appender := range appenders {
		if appender.requiresLocation() {
			return true
		}
	}
	return false
}
