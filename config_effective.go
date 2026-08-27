package goarklog

import (
	"fmt"
	"strings"
)

func (c *fileConfig) effective() (*fileConfig, error) {
	topLevelUsed := !c.withoutWrappers().empty()
	wrappers := 0
	if c.Goark.Log != nil {
		wrappers++
	}
	if c.Configuration != nil {
		wrappers++
	}
	if wrappers == 0 {
		return c, nil
	}
	if topLevelUsed {
		return nil, fmt.Errorf("goark-log: config must use either top-level fields, configuration, or goark.log")
	}
	if wrappers > 1 {
		return nil, fmt.Errorf("goark-log: config must use only one wrapper: configuration or goark.log")
	}
	if c.Configuration != nil {
		return c.Configuration, nil
	}
	return c.Goark.Log, nil
}

func (c *fileConfig) withoutWrappers() *fileConfig {
	if c == nil {
		return nil
	}
	return &fileConfig{
		Status:            c.Status,
		MonitorInterval:   c.MonitorInterval,
		MonitorKebab:      c.MonitorKebab,
		Properties:        c.Properties,
		CustomLevels:      c.CustomLevels,
		CustomLevelsKebab: c.CustomLevelsKebab,
		Appenders:         c.Appenders,
		Filters:           c.Filters,
		FilterRefs:        c.FilterRefs,
		FilterRefsKebab:   c.FilterRefsKebab,
		AsyncLogger:       c.AsyncLogger,
		AsyncLoggerKebab:  c.AsyncLoggerKebab,
		Async:             c.Async,
		Root:              c.Root,
		Loggers:           c.Loggers,
	}
}

func (c *fileConfig) empty() bool {
	if c == nil {
		return true
	}
	return len(c.Appenders) == 0 &&
		strings.TrimSpace(c.Status) == "" &&
		strings.TrimSpace(c.MonitorInterval) == "" &&
		strings.TrimSpace(c.MonitorKebab) == "" &&
		len(c.Properties) == 0 &&
		len(c.CustomLevels) == 0 &&
		len(c.CustomLevelsKebab) == 0 &&
		len(c.Filters) == 0 &&
		len(c.FilterRefs) == 0 &&
		len(c.FilterRefsKebab) == 0 &&
		c.AsyncLogger.empty() &&
		c.AsyncLoggerKebab.empty() &&
		c.Async.empty() &&
		c.Root.empty() &&
		len(c.Loggers) == 0
}

func (c loggerConfig) empty() bool {
	return strings.TrimSpace(c.Level) == "" &&
		len(c.AppenderRefs) == 0 &&
		len(c.AppenderRefsKebab) == 0 &&
		len(c.Refs) == 0 &&
		len(c.Filters) == 0 &&
		len(c.FilterRefs) == 0 &&
		len(c.FilterRefsKebab) == 0 &&
		c.Additivity == nil &&
		c.IncludeLocation == nil &&
		c.IncludeLocationKebab == nil
}

func (c asyncLoggerConfig) empty() bool {
	return c.Enabled == nil &&
		c.QueueSize == 0 &&
		c.QueueSizeKebab == 0 &&
		c.BatchSize == 0 &&
		c.BatchSizeKebab == 0 &&
		strings.TrimSpace(c.OverflowStrategy) == "" &&
		strings.TrimSpace(c.OverflowStrategyKebab) == "" &&
		strings.TrimSpace(c.WaitStrategy) == "" &&
		strings.TrimSpace(c.WaitStrategyKebab) == "" &&
		c.WaitRetries == 0 &&
		c.WaitRetriesKebab == 0 &&
		strings.TrimSpace(c.SleepTime) == "" &&
		strings.TrimSpace(c.SleepTimeKebab) == "" &&
		strings.TrimSpace(c.Timeout) == "" &&
		c.IncludeLocation == nil &&
		c.IncludeLocationKebab == nil
}
