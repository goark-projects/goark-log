package goarklog

import (
	"strings"
)

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
	case "includeLocation", "include-location":
		parsed, err := parsePropertyBool(value, key)
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
