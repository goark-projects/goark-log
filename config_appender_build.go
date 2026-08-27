package goarklog

import (
	"fmt"
	"strings"

	"goark.dev/log/internal/textutil"
)

func (c *fileConfig) buildAppenders(filters map[string]Filter, registry *PluginRegistry) ([]Appender, error) {
	if len(c.Appenders) == 0 {
		return DefaultOptions().Appenders, nil
	}
	appenderNames := textutil.SortedKeys(c.Appenders)
	built := make(map[string]Appender, len(c.Appenders))
	appenders := make([]Appender, 0, len(c.Appenders))
	compositeNames := make([]string, 0)
	for _, name := range appenderNames {
		spec := c.Appenders[name]
		if isCompositeAppenderKind(spec.Type) {
			compositeNames = append(compositeNames, name)
			continue
		}
		appender, err := buildConcreteAppender(name, spec, filters, registry)
		if err != nil {
			_ = closeAppenderList(appenders)
			return nil, err
		}
		built[name] = appender
		appenders = append(appenders, appender)
	}
	for len(compositeNames) > 0 {
		progress := false
		remaining := compositeNames[:0]
		for _, name := range compositeNames {
			appender, waiting, err := buildCompositeAppender(name, c.Appenders[name], c.Appenders, built, filters, registry)
			if err != nil {
				_ = closeAppenderList(appenders)
				return nil, err
			}
			if waiting {
				remaining = append(remaining, name)
				continue
			}
			built[name] = appender
			appenders = append(appenders, appender)
			progress = true
		}
		if !progress {
			_ = closeAppenderList(appenders)
			return nil, fmt.Errorf("goark-log: appender dependencies are unresolved: %s", strings.Join(remaining, ", "))
		}
		compositeNames = remaining
	}
	return appenders, nil
}

func buildConcreteAppender(name string, spec appenderConfig, filters map[string]Filter, registry *PluginRegistry) (Appender, error) {
	layout, err := buildLayout(spec.Layout, registry)
	if err != nil {
		return nil, fmt.Errorf("goark-log: appender %q: %w", name, err)
	}
	kind := textutil.NormalizeKind(spec.Type)
	if kind == "" {
		return nil, fmt.Errorf("goark-log: appender %q type is empty", name)
	}
	factory, ok := registry.appenderFactory(kind)
	if !ok {
		return nil, fmt.Errorf("goark-log: unsupported appender %q type %q", name, spec.Type)
	}
	appender, err := factory(spec.appenderBuildConfig(name, layout, nil, nil, nil))
	if err != nil {
		return nil, err
	}
	wrapped, err := wrapAppenderFilters(name, appender, spec.filterRefs(), filters)
	if err != nil {
		_ = appender.Close()
		return nil, err
	}
	return wrapped, nil
}
