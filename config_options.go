package goarklog

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"goark.dev/log/internal/textutil"
)

func (c *fileConfig) options(registry *PluginRegistry) (Options, error) {
	if c == nil || c.empty() {
		return DefaultOptions(), nil
	}
	if err := c.registerCustomLevels(); err != nil {
		return Options{}, err
	}
	if err := c.validateAsyncLoggerConfig(); err != nil {
		return Options{}, err
	}
	if registry == nil {
		registry = DefaultPluginRegistry()
	}
	filters, err := c.buildFilters(registry)
	if err != nil {
		return Options{}, err
	}
	appenders, err := c.buildAppenders(filters, registry)
	if err != nil {
		return Options{}, err
	}
	globalFilters, err := resolveFilters(filters, c.filterRefs())
	if err != nil {
		_ = closeAppenderList(appenders)
		return Options{}, err
	}
	rootFilters, err := resolveFilters(filters, c.Root.filterRefs())
	if err != nil {
		_ = closeAppenderList(appenders)
		return Options{}, fmt.Errorf("goark-log: root: %w", err)
	}
	rootLevel, err := ParseLevel(c.Root.Level)
	if err != nil {
		_ = closeAppenderList(appenders)
		return Options{}, err
	}
	options := Options{
		Appenders: appenders,
		Filters:   globalFilters,
		Async:     c.asyncLoggerOptions(),
		Root: RootLogger{
			Level:           rootLevel,
			AppenderRefs:    c.Root.refs(),
			Filters:         rootFilters,
			IncludeLocation: c.Root.includeLocation(),
		},
	}
	options.Root.AppenderRefControls, err = c.Root.appenderRefControls(filters)
	if err != nil {
		_ = closeAppenderList(appenders)
		return Options{}, fmt.Errorf("goark-log: root: %w", err)
	}
	loggerNames := textutil.SortedKeys(c.Loggers)
	for _, name := range loggerNames {
		loggerConfig := c.Loggers[name]
		var level *slog.Level
		if strings.TrimSpace(loggerConfig.Level) != "" {
			parsed, err := ParseLevel(loggerConfig.Level)
			if err != nil {
				_ = closeAppenderList(appenders)
				return Options{}, fmt.Errorf("goark-log: logger %q: %w", name, err)
			}
			level = &parsed
		}
		rule := LoggerRule{
			Name:            name,
			Level:           level,
			AppenderRefs:    loggerConfig.refs(),
			IncludeLocation: loggerConfig.includeLocationPointer(),
		}
		rule.AppenderRefControls, err = loggerConfig.appenderRefControls(filters)
		if err != nil {
			_ = closeAppenderList(appenders)
			return Options{}, fmt.Errorf("goark-log: logger %q: %w", name, err)
		}
		rule.Filters, err = resolveFilters(filters, loggerConfig.filterRefs())
		if err != nil {
			_ = closeAppenderList(appenders)
			return Options{}, fmt.Errorf("goark-log: logger %q: %w", name, err)
		}
		if loggerConfig.Additivity != nil {
			rule.Additivity = *loggerConfig.Additivity
			rule.AdditivitySet = true
		}
		options.Loggers = append(options.Loggers, rule)
	}
	return options, nil
}

func (c *fileConfig) validateAsyncLoggerConfig() error {
	used := 0
	for _, candidate := range []asyncLoggerConfig{c.AsyncLogger, c.AsyncLoggerKebab, c.Async} {
		if !candidate.empty() {
			used++
		}
	}
	if used > 1 {
		return fmt.Errorf("goark-log: config must use only one of asyncLogger, async-logger, or async")
	}
	config := c.asyncLoggerConfig()
	if config.empty() {
		return nil
	}
	if config.queueSize() < 0 {
		return fmt.Errorf("goark-log: asyncLogger queueSize must be >= 0")
	}
	if config.batchSize() < 0 {
		return fmt.Errorf("goark-log: asyncLogger batchSize must be >= 0")
	}
	if _, err := ParseAsyncOverflowStrategy(config.overflowStrategy()); err != nil {
		return fmt.Errorf("goark-log: asyncLogger: %w", err)
	}
	if _, err := ParseAsyncWaitStrategy(config.waitStrategy()); err != nil {
		return fmt.Errorf("goark-log: asyncLogger: %w", err)
	}
	if err := validateAsyncWaitOptions(config.waitOptions()); err != nil {
		return fmt.Errorf("goark-log: asyncLogger: %w", err)
	}
	return nil
}

func (c loggerConfig) includeLocation() bool {
	if includeLocation := c.includeLocationPointer(); includeLocation != nil {
		return *includeLocation
	}
	return false
}

func (c loggerConfig) includeLocationPointer() *bool {
	if c.IncludeLocation != nil {
		value := *c.IncludeLocation
		return &value
	}
	if c.IncludeLocationKebab != nil {
		value := *c.IncludeLocationKebab
		return &value
	}
	return nil
}

func (c *fileConfig) registerCustomLevels() error {
	levels := mergeStringMaps(copyStringMap(c.CustomLevels), c.CustomLevelsKebab)
	for name, value := range levels {
		parsed, err := parseCustomLevelValue(value)
		if err != nil {
			return fmt.Errorf("goark-log: custom level %q: %w", name, err)
		}
		if err := RegisterLevel(name, slog.Level(parsed)); err != nil {
			return fmt.Errorf("goark-log: custom level %q: %w", name, err)
		}
	}
	return nil
}

func parseCustomLevelValue(value string) (int, error) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("value %q is invalid", value)
	}
	return parsed, nil
}

func (c *fileConfig) asyncLoggerOptions() AsyncLoggerOptions {
	config := c.asyncLoggerConfig()
	if config.empty() {
		return AsyncLoggerOptions{}
	}
	options := AsyncLoggerOptions{
		QueueSize:        config.queueSize(),
		BatchSize:        config.batchSize(),
		OverflowStrategy: AsyncOverflowStrategy(config.overflowStrategy()),
		WaitStrategy:     AsyncWaitStrategy(config.waitStrategy()),
		WaitOptions:      config.waitOptions(),
		IncludeLocation:  config.includeLocation(),
	}
	if config.Enabled != nil {
		options.Enabled = *config.Enabled
	}
	return options
}

func (c *fileConfig) asyncLoggerConfig() asyncLoggerConfig {
	for _, candidate := range []asyncLoggerConfig{c.AsyncLogger, c.AsyncLoggerKebab, c.Async} {
		if !candidate.empty() {
			return candidate
		}
	}
	return asyncLoggerConfig{}
}

func (c asyncLoggerConfig) queueSize() int {
	if c.QueueSize != 0 {
		return c.QueueSize
	}
	return c.QueueSizeKebab
}

func (c asyncLoggerConfig) batchSize() int {
	if c.BatchSize != 0 {
		return c.BatchSize
	}
	return c.BatchSizeKebab
}

func (c asyncLoggerConfig) overflowStrategy() string {
	return textutil.FirstNonBlank(c.OverflowStrategy, c.OverflowStrategyKebab)
}

func (c asyncLoggerConfig) waitStrategy() string {
	return textutil.FirstNonBlank(c.WaitStrategy, c.WaitStrategyKebab)
}

func (c asyncLoggerConfig) waitOptions() AsyncWaitOptions {
	return AsyncWaitOptions{
		Retries:   textutil.FirstNonZero(c.WaitRetries, c.WaitRetriesKebab),
		SleepTime: textutil.OptionalDuration(textutil.FirstNonBlank(c.SleepTime, c.SleepTimeKebab)),
		Timeout:   textutil.OptionalDuration(c.Timeout),
	}
}

func (c asyncLoggerConfig) includeLocation() bool {
	if c.IncludeLocation != nil {
		return *c.IncludeLocation
	}
	if c.IncludeLocationKebab != nil {
		return *c.IncludeLocationKebab
	}
	return false
}
