package goarklog

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func decodePropertiesConfig(reader io.Reader, lookups *LookupResolver) (*fileConfig, error) {
	values, err := readProperties(reader)
	if err != nil {
		return nil, err
	}
	config, err := propertiesToFileConfig(values)
	if err != nil {
		return nil, err
	}
	return finalizeDecodedConfig(config, lookups)
}

func readProperties(reader io.Reader) (map[string]string, error) {
	scanner := bufio.NewScanner(reader)
	values := make(map[string]string)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}
		key, value, ok := cutProperty(line)
		if !ok {
			return nil, fmt.Errorf("goark-log: properties line %d is invalid", lineNumber)
		}
		values[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return values, nil
}

func cutProperty(line string) (string, string, bool) {
	for _, separator := range []string{"=", ":"} {
		key, value, ok := strings.Cut(line, separator)
		if ok {
			return key, value, true
		}
	}
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return "", "", false
	}
	return fields[0], strings.Join(fields[1:], " "), true
}

func propertiesToFileConfig(values map[string]string) (fileConfig, error) {
	aliases := collectPropertyAliases(values)
	config := fileConfig{
		Properties: make(map[string]string),
		Appenders:  make(map[string]appenderConfig),
		Filters:    make(map[string]filterConfig),
		Loggers:    make(map[string]loggerConfig),
	}
	for key, value := range values {
		if err := applyProperty(&config, aliases, key, value); err != nil {
			return fileConfig{}, err
		}
	}
	if len(config.Properties) == 0 {
		config.Properties = nil
	}
	return config, nil
}

type propertyAliases struct {
	appenders map[string]string
	loggers   map[string]string
}

func collectPropertyAliases(values map[string]string) propertyAliases {
	aliases := propertyAliases{
		appenders: make(map[string]string),
		loggers:   make(map[string]string),
	}
	for key, value := range values {
		if strings.HasPrefix(key, "appender.") {
			id, field, ok := splitPropertyID(strings.TrimPrefix(key, "appender."))
			if ok && field == "name" && strings.TrimSpace(value) != "" {
				aliases.appenders[id] = strings.TrimSpace(value)
			}
		}
		if strings.HasPrefix(key, "logger.") {
			id, field, ok := splitPropertyID(strings.TrimPrefix(key, "logger."))
			if ok && field == "name" && strings.TrimSpace(value) != "" {
				aliases.loggers[id] = strings.TrimSpace(value)
			}
		}
	}
	return aliases
}

func (a propertyAliases) appenderName(id string) string {
	if name := strings.TrimSpace(a.appenders[id]); name != "" {
		return name
	}
	return id
}

func (a propertyAliases) loggerName(id string) string {
	if name := strings.TrimSpace(a.loggers[id]); name != "" {
		return name
	}
	return id
}

func applyProperty(config *fileConfig, aliases propertyAliases, key string, value string) error {
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
		config.Root.Filters = propertyList(value)
	case strings.HasPrefix(key, "property."):
		name := strings.TrimPrefix(key, "property.")
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("goark-log: properties key %q has empty property name", key)
		}
		config.Properties[name] = value
	case strings.HasPrefix(key, "appender."):
		return applyAppenderProperty(config, aliases, strings.TrimPrefix(key, "appender."), value)
	case strings.HasPrefix(key, "logger."):
		return applyLoggerProperty(config, aliases, strings.TrimPrefix(key, "logger."), value)
	case strings.HasPrefix(key, "filter."):
		return applyFilterProperty(config, strings.TrimPrefix(key, "filter."), value)
	}
	return nil
}

func applyAppenderProperty(config *fileConfig, aliases propertyAliases, key string, value string) error {
	id, field, ok := splitPropertyID(key)
	if !ok {
		return nil
	}
	id = aliases.appenderName(id)
	appender := config.Appenders[id]
	switch field {
	case "name":
		return nil
	case "type":
		appender.Type = value
	case "target":
		appender.Target = value
	case "url":
		appender.URL = value
	case "method":
		appender.Method = value
	case "address":
		appender.Address = value
	case "network":
		appender.Network = value
	case "facility":
		appender.Facility = value
	case "appName", "app-name":
		appender.AppName = value
	case "connectTimeout", "connect-timeout":
		appender.ConnectTimeout = value
	case "writeTimeout", "write-timeout":
		appender.WriteTimeout = value
	case "fileName", "file-name", "path":
		appender.FileName = value
	case "appenderRefs", "appender-refs", "refs":
		appender.AppenderRefs = propertyAppenderRefs(value)
	case "queueSize", "queue-size":
		parsed, err := parsePropertyInt(value, key)
		if err != nil {
			return err
		}
		appender.QueueSize = parsed
	case "overflowStrategy", "overflow-strategy":
		appender.OverflowStrategy = value
	case "waitStrategy", "wait-strategy":
		appender.WaitStrategy = value
	case "bufferSize", "buffer-size":
		appender.BufferSize = value
	case "flushOnWrite", "flush-on-write":
		parsed, err := parsePropertyBool(value, key)
		if err != nil {
			return err
		}
		appender.FlushOnWrite = parsed
	case "filters", "filterRefs", "filter-refs":
		appender.Filters = propertyList(value)
	case "layout.type":
		appender.Layout.Type = value
	case "layout.pattern":
		appender.Layout.Pattern = value
	case "layout.eventTemplate", "layout.event-template":
		appender.Layout.EventTemplate = value
	case "rolling.filePattern", "rolling.file-pattern":
		appender.Rolling.FilePattern = value
	case "rolling.maxSize", "rolling.max-size":
		appender.Rolling.MaxSize = value
	case "rolling.interval":
		appender.Rolling.Interval = value
	}
	config.Appenders[id] = appender
	return nil
}

func applyLoggerProperty(config *fileConfig, aliases propertyAliases, key string, value string) error {
	id, field, ok := splitPropertyID(key)
	if !ok {
		return nil
	}
	id = aliases.loggerName(id)
	logger := config.Loggers[id]
	switch field {
	case "name":
		return nil
	case "level":
		logger.Level = value
	case "appenderRefs", "appender-refs", "refs":
		logger.AppenderRefs = propertyAppenderRefs(value)
	case "filters", "filterRefs", "filter-refs":
		logger.Filters = propertyList(value)
	case "additivity":
		parsed, err := parsePropertyBool(value, key)
		if err != nil {
			return err
		}
		logger.Additivity = &parsed
	}
	config.Loggers[id] = logger
	return nil
}

func applyFilterProperty(config *fileConfig, key string, value string) error {
	id, field, ok := splitPropertyID(key)
	if !ok {
		return nil
	}
	filter := config.Filters[id]
	switch field {
	case "type":
		filter.Type = value
	case "level":
		filter.Level = value
	case "minLevel", "min-level":
		filter.MinLevel = value
	case "maxLevel", "max-level":
		filter.MaxLevel = value
	case "field":
		filter.Field = value
	case "key":
		filter.Key = value
	case "value":
		filter.Value = value
	case "pattern":
		filter.Pattern = value
	case "onMatch", "on-match":
		filter.OnMatch = value
	case "onMismatch", "on-mismatch":
		filter.OnMismatch = value
	}
	config.Filters[id] = filter
	return nil
}

func splitPropertyID(key string) (string, string, bool) {
	id, field, ok := strings.Cut(key, ".")
	id = strings.TrimSpace(id)
	field = strings.TrimSpace(field)
	if !ok || id == "" || field == "" {
		return "", "", false
	}
	return id, field, true
}

func propertyAppenderRefs(value string) appenderRefs {
	values := propertyList(value)
	refs := make(appenderRefs, 0, len(values))
	for _, ref := range values {
		refs = append(refs, appenderRefConfig{Ref: ref})
	}
	return refs
}

func propertyList(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';'
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func parsePropertyInt(value string, field string) (int, error) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("goark-log: properties %s is invalid", field)
	}
	return parsed, nil
}

func parsePropertyBool(value string, field string) (bool, error) {
	parsed, err := strconv.ParseBool(strings.ToLower(strings.TrimSpace(value)))
	if err != nil {
		return false, fmt.Errorf("goark-log: properties %s is invalid", field)
	}
	return parsed, nil
}
