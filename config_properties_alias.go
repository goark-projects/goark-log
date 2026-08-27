package goarklog

import (
	"strings"
)

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
