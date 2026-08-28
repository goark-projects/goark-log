package configfile

import (
	"fmt"
	"io"
	"strings"

	"goark.dev/log/internal/configprops"
)

func decodePropertiesConfig(reader io.Reader, lookups *LookupResolver) (*fileConfig, error) {
	values, err := configprops.Read(reader)
	if err != nil {
		return nil, err
	}
	config, err := propertiesToFileConfig(values)
	if err != nil {
		return nil, err
	}
	return finalizeDecodedConfig(config, lookups)
}

func propertyAppenderRefs(value string) appenderRefs {
	values := configprops.List(value)
	refs := make(appenderRefs, 0, len(values))
	for _, ref := range values {
		refs = append(refs, appenderRefConfig{Ref: ref})
	}
	return refs
}

func applyAppenderRefProperty(refs *appenderRefs, key string, value string) error {
	id, field, ok := configprops.SplitID(key)
	if !ok {
		return nil
	}
	ref := findPropertyAppenderRef(refs, id)
	switch field {
	case "ref":
		ref.Ref = value
	case "level":
		ref.Level = value
	case "includeLocation", "include-location":
		parsed, err := configprops.Bool(value, key)
		if err != nil {
			return err
		}
		ref.IncludeLocation = &parsed
	case "filters", "filterRefs", "filter-refs":
		ref.FilterRefs = configprops.List(value)
	}
	return nil
}

func findPropertyAppenderRef(refs *appenderRefs, id string) *appenderRefConfig {
	for index := range *refs {
		if (*refs)[index].ID == id || ((*refs)[index].ID == "" && (*refs)[index].Ref == id) {
			if (*refs)[index].ID == "" {
				(*refs)[index].ID = id
			}
			return &(*refs)[index]
		}
	}
	*refs = append(*refs, appenderRefConfig{ID: id, Ref: id})
	return &(*refs)[len(*refs)-1]
}

func applyRewriteProperty(config *rewriteBuildConfig, key string, value string) error {
	switch {
	case strings.HasPrefix(key, "attrs."):
		attrKey := strings.TrimSpace(strings.TrimPrefix(key, "attrs."))
		if attrKey == "" {
			return fmt.Errorf("goark-log: properties rewrite attr key is empty")
		}
		if config.Attrs == nil {
			config.Attrs = make(map[string]string)
		}
		config.Attrs[attrKey] = value
	case strings.HasPrefix(key, "attributes."):
		attrKey := strings.TrimSpace(strings.TrimPrefix(key, "attributes."))
		if attrKey == "" {
			return fmt.Errorf("goark-log: properties rewrite attribute key is empty")
		}
		if config.Attributes == nil {
			config.Attributes = make(map[string]string)
		}
		config.Attributes[attrKey] = value
	case strings.HasPrefix(key, "properties."):
		attrKey := strings.TrimSpace(strings.TrimPrefix(key, "properties."))
		if attrKey == "" {
			return fmt.Errorf("goark-log: properties rewrite property key is empty")
		}
		if config.Properties == nil {
			config.Properties = make(map[string]string)
		}
		config.Properties[attrKey] = value
	case key == "remove" || key == "removeAttrs" || key == "remove-attrs":
		config.Remove = configprops.List(value)
	}
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
		parsed, err := configprops.Bool(value, key)
		if err != nil {
			return err
		}
		config.Compact = parsed
	case "eventEol", "event-eol":
		parsed, err := configprops.Bool(value, key)
		if err != nil {
			return err
		}
		config.EventEOL = parsed
	case "complete":
		parsed, err := configprops.Bool(value, key)
		if err != nil {
			return err
		}
		config.Complete = parsed
	case "includeStacktrace", "include-stacktrace":
		parsed, err := configprops.Bool(value, key)
		if err != nil {
			return err
		}
		config.IncludeStacktrace = parsed
	case "stacktraceAsString", "stacktrace-as-string":
		parsed, err := configprops.Bool(value, key)
		if err != nil {
			return err
		}
		config.StacktraceAsString = parsed
	case "propertiesAsList", "properties-as-list":
		parsed, err := configprops.Bool(value, key)
		if err != nil {
			return err
		}
		config.PropertiesAsList = parsed
	case "includeNullDelimiter", "include-null-delimiter":
		parsed, err := configprops.Bool(value, key)
		if err != nil {
			return err
		}
		config.IncludeNullDelimiter = parsed
	case "disableAnsi", "disable-ansi":
		parsed, err := configprops.Bool(value, key)
		if err != nil {
			return err
		}
		config.DisableANSI = parsed
	case "header":
		config.Header = value
	case "footer":
		config.Footer = value
	}
	return nil
}

func propertiesToFileConfig(values map[string]string) (fileConfig, error) {
	aliases := configprops.CollectAliases(values)
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

func applyLoggerProperty(config *fileConfig, aliases configprops.Aliases, key string, value string) error {
	id, field, ok := configprops.SplitID(key)
	if !ok {
		return nil
	}
	id = aliases.LoggerName(id)
	logger := config.Loggers[id]
	switch field {
	case "name":
		return nil
	case "level":
		logger.Level = value
	case "appenderRefs", "appender-refs", "refs":
		logger.AppenderRefs = propertyAppenderRefs(value)
	case "filters", "filterRefs", "filter-refs":
		logger.Filters = configprops.List(value)
	case "additivity":
		parsed, err := configprops.Bool(value, key)
		if err != nil {
			return err
		}
		logger.Additivity = &parsed
	case "includeLocation", "include-location":
		parsed, err := configprops.Bool(value, key)
		if err != nil {
			return err
		}
		logger.IncludeLocation = &parsed
	default:
		if strings.HasPrefix(field, "appenderRef.") {
			if err := applyAppenderRefProperty(&logger.AppenderRefs, strings.TrimPrefix(field, "appenderRef."), value); err != nil {
				return err
			}
		}
	}
	config.Loggers[id] = logger
	return nil
}
