package goarklog

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"goark.dev/log/internal/configvalue"
	"goark.dev/log/internal/textutil"
	"gopkg.in/yaml.v3"
)

func loadConfigFile(ctx context.Context, path string, lookups *LookupResolver) (*fileConfig, error) {
	if ctx == nil {
		return nil, fmt.Errorf("goark-log: context is nil")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	format, err := configFormat(path)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("goark-log: open config file %q: %w", path, err)
	}
	defer file.Close()
	config, err := decodeConfig(file, format, lookups)
	if err != nil {
		return nil, fmt.Errorf("goark-log: parse config file %q: %w", path, err)
	}
	return config, nil
}

// ParseByteSize 解析日志滚动大小。
func ParseByteSize(value string) (int64, error) {
	return configvalue.ByteSize(value)
}

// ParseRollingInterval 解析时间滚动间隔。
func ParseRollingInterval(value string) (time.Duration, error) {
	return configvalue.RollingInterval(value)
}

// ParseRollingMaxAge 解析滚动档案最大保留时间。
func ParseRollingMaxAge(value string) (time.Duration, error) {
	return configvalue.RollingMaxAge(value)
}

// ParseMonitorInterval 解析配置监控间隔；纯数字按秒处理。
func ParseMonitorInterval(value string) (time.Duration, error) {
	return configvalue.MonitorInterval(value)
}

func decodeConfig(reader io.Reader, format string, lookups *LookupResolver) (*fileConfig, error) {
	switch format {
	case "yaml", "json":
		return decodeStructuredConfig(reader, lookups)
	case "xml":
		return decodeXMLConfig(reader, lookups)
	case "properties":
		return decodePropertiesConfig(reader, lookups)
	default:
		return nil, fmt.Errorf("goark-log: unsupported config format %q", format)
	}
}

func decodeStructuredConfig(reader io.Reader, lookups *LookupResolver) (*fileConfig, error) {
	var config fileConfig
	decoder := yaml.NewDecoder(reader)
	decoder.KnownFields(true)
	if err := decoder.Decode(&config); err != nil {
		if errors.Is(err, io.EOF) {
			return &fileConfig{}, nil
		}
		return nil, err
	}
	return finalizeDecodedConfig(config, lookups)
}

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

func finalizeDecodedConfig(config fileConfig, lookups *LookupResolver) (*fileConfig, error) {
	effective, err := config.effective()
	if err != nil {
		return nil, err
	}
	if lookups == nil {
		lookups = NewLookupResolver()
	}
	if err := effective.resolveLookups(lookups.clone()); err != nil {
		return nil, err
	}
	return effective, nil
}

func configFormat(path string) (string, error) {
	switch strings.ToLower(strings.TrimPrefix(filepathExt(path), ".")) {
	case "yml", "yaml":
		return "yaml", nil
	case "json":
		return "json", nil
	case "xml":
		return "xml", nil
	case "toml":
		return "toml", nil
	case "properties":
		return "properties", nil
	default:
		return "", fmt.Errorf("goark-log: unsupported config file extension for %q", path)
	}
}

func filepathExt(path string) string {
	index := strings.LastIndexByte(path, '.')
	if index < 0 {
		return ""
	}
	return path[index:]
}

func (c *fileConfig) monitorInterval() (time.Duration, error) {
	if c == nil {
		return 0, nil
	}
	return ParseMonitorInterval(textutil.FirstNonBlank(c.MonitorInterval, c.MonitorKebab))
}
