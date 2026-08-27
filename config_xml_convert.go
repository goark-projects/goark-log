package goarklog

import (
	"fmt"
	"strings"
)

func (c xmlConfig) appenders(file *fileConfig) error {
	groups := [][]xmlAppender{
		c.Appenders.Console,
		c.Appenders.File,
		c.Appenders.RollingFile,
		c.Appenders.Async,
		c.Appenders.Failover,
		c.Appenders.Routing,
		c.Appenders.Rewrite,
		c.Appenders.HTTP,
		c.Appenders.Socket,
		c.Appenders.Syslog,
	}
	for _, group := range groups {
		for _, item := range group {
			name, appender, err := item.config()
			if err != nil {
				return err
			}
			file.Appenders[name] = appender
		}
	}
	return nil
}

func (c xmlConfig) filters(file *fileConfig) error {
	groups := []struct {
		kind    string
		filters []xmlFilter
	}{
		{kind: "threshold", filters: c.Filters.Threshold},
		{kind: "level", filters: c.Filters.Level},
		{kind: "levelRange", filters: c.Filters.Range},
		{kind: "regex", filters: c.Filters.Regex},
		{kind: "attr", filters: c.Filters.Attr},
		{kind: "attr", filters: c.Filters.Attribute},
		{kind: "deny", filters: c.Filters.Deny},
		{kind: "denyAll", filters: c.Filters.DenyAll},
		{kind: "composite", filters: c.Filters.Composite},
		{kind: "marker", filters: c.Filters.Marker},
		{kind: "noMarker", filters: c.Filters.NoMarker},
		{kind: "map", filters: c.Filters.Map},
		{kind: "threadContextMap", filters: c.Filters.ThreadContextMap},
		{kind: "threadContextStack", filters: c.Filters.ThreadContextStack},
		{kind: "structuredData", filters: c.Filters.StructuredData},
		{kind: "throwable", filters: c.Filters.Throwable},
		{kind: "stringMatch", filters: c.Filters.StringMatch},
		{kind: "time", filters: c.Filters.Time},
		{kind: "burst", filters: c.Filters.Burst},
		{kind: "dynamicThreshold", filters: c.Filters.DynamicThreshold},
	}
	for _, group := range groups {
		for _, item := range group.filters {
			name := strings.TrimSpace(item.Name)
			if name == "" {
				return fmt.Errorf("goark-log: XML filter name is empty")
			}
			config, err := item.config(group.kind)
			if err != nil {
				return fmt.Errorf("goark-log: XML filter %q: %w", name, err)
			}
			file.Filters[name] = config
		}
	}
	return nil
}

func (c xmlConfig) loggers(file *fileConfig) error {
	if !c.Loggers.Root.empty() {
		root, err := c.Loggers.Root.config(false)
		if err != nil {
			return err
		}
		file.Root = root
	}
	for _, logger := range c.Loggers.Logger {
		config, err := logger.config(true)
		if err != nil {
			return err
		}
		file.Loggers[strings.TrimSpace(logger.Name)] = config
	}
	return nil
}
