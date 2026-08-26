package goarklog

import (
	"bufio"
	"fmt"
	"io"
	"sort"
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
		Properties:   make(map[string]string),
		CustomLevels: make(map[string]string),
		Appenders:    make(map[string]appenderConfig),
		Filters:      make(map[string]filterConfig),
		Loggers:      make(map[string]loggerConfig),
	}
	for key, value := range values {
		if err := applyProperty(&config, aliases, key, value); err != nil {
			return fileConfig{}, err
		}
	}
	if err := applyFilterKeyValuePairs(&config, values); err != nil {
		return fileConfig{}, err
	}
	if len(config.Properties) == 0 {
		config.Properties = nil
	}
	if len(config.CustomLevels) == 0 {
		config.CustomLevels = nil
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
		parsed, err := parsePropertyBool(value, key)
		if err != nil {
			return err
		}
		config.Enabled = &parsed
	case "queueSize", "queue-size":
		parsed, err := parsePropertyInt(value, key)
		if err != nil {
			return err
		}
		config.QueueSize = parsed
	case "batchSize", "batch-size":
		parsed, err := parsePropertyInt(value, key)
		if err != nil {
			return err
		}
		config.BatchSize = parsed
	case "overflowStrategy", "overflow-strategy":
		config.OverflowStrategy = value
	case "waitStrategy", "wait-strategy":
		config.WaitStrategy = value
	case "waitRetries", "wait-retries":
		parsed, err := parsePropertyInt(value, key)
		if err != nil {
			return err
		}
		config.WaitRetries = parsed
	case "sleepTime", "sleep-time":
		config.SleepTime = value
	case "timeout":
		config.Timeout = value
	case "includeLocation", "include-location":
		parsed, err := parsePropertyBool(value, key)
		if err != nil {
			return err
		}
		config.IncludeLocation = &parsed
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
	if strings.HasPrefix(field, "layout.") {
		if err := applyLayoutProperty(&appender.Layout, strings.TrimPrefix(field, "layout."), value); err != nil {
			return err
		}
		config.Appenders[id] = appender
		return nil
	}
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
	case "waitRetries", "wait-retries":
		parsed, err := parsePropertyInt(value, key)
		if err != nil {
			return err
		}
		appender.WaitRetries = parsed
	case "sleepTime", "sleep-time":
		appender.SleepTime = value
	case "timeout":
		appender.Timeout = value
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
	case "rolling.filePattern", "rolling.file-pattern":
		appender.Rolling.FilePattern = value
	case "rolling.maxSize", "rolling.max-size":
		appender.Rolling.MaxSize = value
	case "rolling.interval":
		appender.Rolling.Interval = value
	case "rolling.cron", "rolling.cronSchedule", "rolling.cron-schedule", "rolling.policies.cron.schedule", "rolling.policies.cronTriggeringPolicy.schedule", "rolling.policies.cron-triggering-policy.schedule":
		appender.Rolling.CronSchedule = value
	case "rolling.strategy.delete.maxCount", "rolling.strategy.delete.max-count":
		parsed, err := parsePropertyInt(value, key)
		if err != nil {
			return err
		}
		appender.Rolling.Strategy.Delete.MaxCount = &parsed
	case "rolling.strategy.delete.maxSize", "rolling.strategy.delete.max-size":
		appender.Rolling.Strategy.Delete.MaxSize = value
	case "rolling.strategy.delete.ifAccumulatedFileCount.exceeds", "rolling.strategy.delete.if-accumulated-file-count.exceeds":
		parsed, err := parsePropertyInt(value, key)
		if err != nil {
			return err
		}
		appender.Rolling.Strategy.Delete.IfAccumulatedFileCount.Exceeds = parsed
	case "rolling.strategy.delete.ifAccumulatedFileSize.exceeds", "rolling.strategy.delete.if-accumulated-file-size.exceeds":
		appender.Rolling.Strategy.Delete.IfAccumulatedFileSize.Exceeds = value
	case "rolling.strategy.type":
		appender.Rolling.Strategy.Type = value
	case "rolling.strategy.fileIndex", "rolling.strategy.file-index":
		appender.Rolling.Strategy.FileIndex = value
	case "rolling.directWrite", "rolling.direct-write", "rolling.strategy.directWrite", "rolling.strategy.direct-write":
		parsed, err := parsePropertyBool(value, key)
		if err != nil {
			return err
		}
		appender.Rolling.DirectWrite = parsed
	}
	config.Appenders[id] = appender
	return nil
}

func applyLayoutProperty(config *layoutConfig, key string, value string) error {
	switch key {
	case "type":
		config.Type = value
	case "pattern":
		config.Pattern = value
	case "eventTemplate", "event-template":
		config.EventTemplate = value
	case "eventTemplateUri", "event-template-uri", "eventTemplatePath", "event-template-path":
		config.EventTemplateURI = value
	case "compact":
		parsed, err := parsePropertyBool(value, key)
		if err != nil {
			return err
		}
		config.Compact = parsed
	case "eventEol", "event-eol":
		parsed, err := parsePropertyBool(value, key)
		if err != nil {
			return err
		}
		config.EventEOL = parsed
	case "complete":
		parsed, err := parsePropertyBool(value, key)
		if err != nil {
			return err
		}
		config.Complete = parsed
	case "includeStacktrace", "include-stacktrace":
		parsed, err := parsePropertyBool(value, key)
		if err != nil {
			return err
		}
		config.IncludeStacktrace = parsed
	case "stacktraceAsString", "stacktrace-as-string":
		parsed, err := parsePropertyBool(value, key)
		if err != nil {
			return err
		}
		config.StacktraceAsString = parsed
	case "propertiesAsList", "properties-as-list":
		parsed, err := parsePropertyBool(value, key)
		if err != nil {
			return err
		}
		config.PropertiesAsList = parsed
	case "includeNullDelimiter", "include-null-delimiter":
		parsed, err := parsePropertyBool(value, key)
		if err != nil {
			return err
		}
		config.IncludeNullDelimiter = parsed
	case "header":
		config.Header = value
	case "footer":
		config.Footer = value
	}
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
	switch {
	case field == "type":
		filter.Type = value
	case field == "level":
		filter.Level = value
	case field == "minLevel" || field == "min-level":
		filter.MinLevel = value
	case field == "maxLevel" || field == "max-level":
		filter.MaxLevel = value
	case field == "marker":
		filter.Marker = value
	case field == "text":
		filter.Text = value
	case field == "operator":
		filter.Operator = value
	case field == "start":
		filter.Start = value
	case field == "end":
		filter.End = value
	case field == "timezone":
		filter.Timezone = value
	case field == "rate":
		filter.Rate = value
	case field == "maxBurst" || field == "max-burst":
		parsed, err := parsePropertyInt(value, key)
		if err != nil {
			return err
		}
		filter.MaxBurst = parsed
	case field == "field":
		filter.Field = value
	case field == "key":
		filter.Key = value
	case field == "value":
		filter.Value = value
	case strings.HasPrefix(field, "values."):
		mapKey := strings.TrimSpace(strings.TrimPrefix(field, "values."))
		if mapKey == "" {
			return fmt.Errorf("goark-log: properties filter.%s.%s has empty values key", id, field)
		}
		if filter.Values == nil {
			filter.Values = make(map[string]string)
		}
		filter.Values[mapKey] = value
	case strings.HasPrefix(field, "thresholds."):
		mapKey := strings.TrimSpace(strings.TrimPrefix(field, "thresholds."))
		if mapKey == "" {
			return fmt.Errorf("goark-log: properties filter.%s.%s has empty thresholds key", id, field)
		}
		if filter.Thresholds == nil {
			filter.Thresholds = make(map[string]string)
		}
		filter.Thresholds[mapKey] = value
	case field == "filters" || field == "filterRefs" || field == "filter-refs":
		filter.FilterRefs = propertyList(value)
	case field == "defaultThreshold" || field == "default-threshold":
		filter.DefaultThreshold = value
	case field == "pattern":
		filter.Pattern = value
	case field == "onMatch" || field == "on-match":
		filter.OnMatch = value
	case field == "onMismatch" || field == "on-mismatch":
		filter.OnMismatch = value
	}
	config.Filters[id] = filter
	return nil
}

type propertyFilterPair struct {
	key      string
	value    string
	hasKey   bool
	hasValue bool
}

func applyFilterKeyValuePairs(config *fileConfig, values map[string]string) error {
	pairsByFilter := make(map[string]map[string]propertyFilterPair)
	for key, value := range values {
		filterID, pairID, field, ok := splitFilterPairProperty(key)
		if !ok {
			continue
		}
		pairs := pairsByFilter[filterID]
		if pairs == nil {
			pairs = make(map[string]propertyFilterPair)
			pairsByFilter[filterID] = pairs
		}
		pair := pairs[pairID]
		switch field {
		case "key":
			pair.key = value
			pair.hasKey = true
		case "value":
			pair.value = value
			pair.hasValue = true
		}
		pairs[pairID] = pair
	}
	filterIDs := make([]string, 0, len(pairsByFilter))
	for filterID := range pairsByFilter {
		filterIDs = append(filterIDs, filterID)
	}
	sort.Strings(filterIDs)
	for _, filterID := range filterIDs {
		filter := config.Filters[filterID]
		pairIDs := make([]string, 0, len(pairsByFilter[filterID]))
		for pairID := range pairsByFilter[filterID] {
			pairIDs = append(pairIDs, pairID)
		}
		sort.Strings(pairIDs)
		for _, pairID := range pairIDs {
			pair := pairsByFilter[filterID][pairID]
			if !pair.hasKey && !pair.hasValue {
				continue
			}
			if !pair.hasKey || strings.TrimSpace(pair.key) == "" || !pair.hasValue {
				return fmt.Errorf("goark-log: properties filter.%s.%s requires key and value", filterID, pairID)
			}
			filter.KeyValuePair = append(filter.KeyValuePair, keyValuePairConfig{
				Key:   pair.key,
				Value: pair.value,
			})
		}
		config.Filters[filterID] = filter
	}
	return nil
}

func splitFilterPairProperty(key string) (string, string, string, bool) {
	if !strings.HasPrefix(key, "filter.") {
		return "", "", "", false
	}
	filterID, field, ok := splitPropertyID(strings.TrimPrefix(key, "filter."))
	if !ok {
		return "", "", "", false
	}
	pairID, pairField, ok := splitPropertyID(field)
	if !ok {
		return "", "", "", false
	}
	normalized := normalizeKind(pairID)
	if !strings.HasPrefix(normalized, "keyvaluepair") && !strings.HasPrefix(normalized, "kv") {
		return "", "", "", false
	}
	switch strings.ToLower(pairField) {
	case "key", "value":
		return filterID, pairID, strings.ToLower(pairField), true
	default:
		return "", "", "", false
	}
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
