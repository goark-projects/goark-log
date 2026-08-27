package goarklog

import (
	"fmt"
	"strconv"
	"strings"

	"goark.dev/log/internal/configxml"
)

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
