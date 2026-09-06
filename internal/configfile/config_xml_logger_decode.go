package configfile

import (
	"fmt"
	"strconv"
	"strings"

	"goark.dev/log/internal/configxml"
)

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

func (l xmlLogger) config(named bool) (loggerConfig, error) {
	if named && strings.TrimSpace(l.Name) == "" {
		return loggerConfig{}, fmt.Errorf("goark-log: XML logger name is empty")
	}
	appenderRefs, err := xmlAppenderRefs(l.AppenderRefs)
	if err != nil {
		return loggerConfig{}, fmt.Errorf("goark-log: XML logger %q: %w", l.Name, err)
	}
	includeLocation, err := configxml.BoolPointerStrict(l.IncludeLocation, "includeLocation")
	if err != nil {
		return loggerConfig{}, fmt.Errorf("goark-log: XML logger %q: %w", l.Name, err)
	}
	config := loggerConfig{
		Level:           l.Level,
		AppenderRefs:    appenderRefs,
		Filters:         xmlFilterRefs(l.FilterRefs),
		IncludeLocation: includeLocation,
	}
	if strings.TrimSpace(l.Additivity) != "" {
		value, err := strconv.ParseBool(strings.ToLower(strings.TrimSpace(l.Additivity)))
		if err != nil {
			return loggerConfig{}, fmt.Errorf("goark-log: XML logger %q additivity is invalid", l.Name)
		}
		config.Additivity = &value
	}
	return config, nil
}

func (l xmlLogger) empty() bool {
	return strings.TrimSpace(l.Name) == "" &&
		strings.TrimSpace(l.Level) == "" &&
		strings.TrimSpace(l.Additivity) == "" &&
		strings.TrimSpace(l.IncludeLocation) == "" &&
		len(l.AppenderRefs) == 0 &&
		len(l.FilterRefs) == 0
}

func (f xmlFilter) config(kind string) (filterConfig, error) {
	if strings.TrimSpace(f.Type) != "" {
		kind = f.Type
	}
	maxBurst, err := configxml.Int(f.MaxBurst, "maxBurst")
	if err != nil {
		return filterConfig{}, err
	}
	return filterConfig{
		Type:             kind,
		Level:            f.Level,
		MinLevel:         f.MinLevel,
		MaxLevel:         f.MaxLevel,
		Marker:           f.Marker,
		Text:             f.Text,
		Operator:         f.Operator,
		Start:            f.Start,
		End:              f.End,
		Timezone:         f.Timezone,
		Rate:             f.Rate,
		MaxBurst:         maxBurst,
		Field:            f.Field,
		Key:              f.Key,
		Value:            f.Value,
		DefaultThreshold: f.DefaultThreshold,
		Pattern:          f.Pattern,
		OnMatch:          f.OnMatch,
		OnMismatch:       f.OnMismatch,
		FilterRefs:       xmlFilterRefs(f.FilterRefs),
		KeyValuePair:     xmlKeyValuePairs(f.KeyValuePair),
	}, nil
}

func (c xmlAsyncLogger) config() (asyncLoggerConfig, error) {
	queueSize, err := configxml.Int(c.QueueSize, "queueSize")
	if err != nil {
		return asyncLoggerConfig{}, err
	}
	batchSize, err := configxml.Int(c.BatchSize, "batchSize")
	if err != nil {
		return asyncLoggerConfig{}, err
	}
	waitRetries, err := configxml.Int(c.WaitRetries, "waitRetries")
	if err != nil {
		return asyncLoggerConfig{}, err
	}
	return asyncLoggerConfig{
		Enabled:          configxml.BoolPointer(c.Enabled),
		QueueSize:        queueSize,
		BatchSize:        batchSize,
		OverflowStrategy: c.OverflowStrategy,
		WaitStrategy:     c.WaitStrategy,
		WaitRetries:      waitRetries,
		SleepTime:        c.SleepTime,
		Timeout:          c.Timeout,
		IncludeLocation:  configxml.BoolPointer(c.IncludeLocation),
	}, nil
}
