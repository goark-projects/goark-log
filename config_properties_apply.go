package goarklog

import (
	"fmt"
	"strings"

	"goark.dev/log/internal/configprops"
)

func applyProperty(config *fileConfig, aliases configprops.Aliases, key string, value string) error {
	switch {
	case key == "status":
		config.Status = value
	case key == "monitorInterval" || key == "monitor-interval":
		config.MonitorInterval = value
	case key == "rootLogger.level" || key == "root.level":
		config.Root.Level = value
	case key == "rootLogger.appenderRefs" || key == "root.appenderRefs":
		config.Root.AppenderRefs = propertyAppenderRefs(value)
	case key == "rootLogger.filters" || key == "root.filters":
		config.Root.Filters = configprops.List(value)
	case key == "rootLogger.includeLocation" || key == "rootLogger.include-location" || key == "root.includeLocation" || key == "root.include-location":
		parsed, err := configprops.Bool(value, key)
		if err != nil {
			return err
		}
		config.Root.IncludeLocation = &parsed
	case strings.HasPrefix(key, "rootLogger.appenderRef."):
		return applyAppenderRefProperty(&config.Root.AppenderRefs, strings.TrimPrefix(key, "rootLogger.appenderRef."), value)
	case strings.HasPrefix(key, "root.appenderRef."):
		return applyAppenderRefProperty(&config.Root.AppenderRefs, strings.TrimPrefix(key, "root.appenderRef."), value)
	case strings.HasPrefix(key, "property."):
		name := strings.TrimPrefix(key, "property.")
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("goark-log: properties key %q has empty property name", key)
		}
		config.Properties[name] = value
	case strings.HasPrefix(key, "customLevel."):
		name := strings.TrimPrefix(key, "customLevel.")
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("goark-log: properties key %q has empty custom level name", key)
		}
		config.CustomLevels[name] = value
	case strings.HasPrefix(key, "custom-level."):
		name := strings.TrimPrefix(key, "custom-level.")
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("goark-log: properties key %q has empty custom level name", key)
		}
		config.CustomLevels[name] = value
	case strings.HasPrefix(key, "asyncLogger."):
		return applyAsyncLoggerProperty(&config.AsyncLogger, strings.TrimPrefix(key, "asyncLogger."), value)
	case strings.HasPrefix(key, "async-logger."):
		return applyAsyncLoggerProperty(&config.AsyncLoggerKebab, strings.TrimPrefix(key, "async-logger."), value)
	case strings.HasPrefix(key, "async."):
		return applyAsyncLoggerProperty(&config.Async, strings.TrimPrefix(key, "async."), value)
	case strings.HasPrefix(key, "appender."):
		return applyAppenderProperty(config, aliases, strings.TrimPrefix(key, "appender."), value)
	case strings.HasPrefix(key, "logger."):
		return applyLoggerProperty(config, aliases, strings.TrimPrefix(key, "logger."), value)
	case strings.HasPrefix(key, "filter."):
		return applyFilterProperty(config, strings.TrimPrefix(key, "filter."), value)
	}
	return nil
}

func applyAsyncLoggerProperty(config *asyncLoggerConfig, key string, value string) error {
	switch key {
	case "enabled":
		parsed, err := configprops.Bool(value, key)
		if err != nil {
			return err
		}
		config.Enabled = &parsed
	case "queueSize", "queue-size":
		parsed, err := configprops.Int(value, key)
		if err != nil {
			return err
		}
		config.QueueSize = parsed
	case "batchSize", "batch-size":
		parsed, err := configprops.Int(value, key)
		if err != nil {
			return err
		}
		config.BatchSize = parsed
	case "overflowStrategy", "overflow-strategy":
		config.OverflowStrategy = value
	case "waitStrategy", "wait-strategy":
		config.WaitStrategy = value
	case "waitRetries", "wait-retries":
		parsed, err := configprops.Int(value, key)
		if err != nil {
			return err
		}
		config.WaitRetries = parsed
	case "sleepTime", "sleep-time":
		config.SleepTime = value
	case "timeout":
		config.Timeout = value
	case "includeLocation", "include-location":
		parsed, err := configprops.Bool(value, key)
		if err != nil {
			return err
		}
		config.IncludeLocation = &parsed
	}
	return nil
}
