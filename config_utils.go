package goarklog

import (
	"sort"
	"strings"
	"time"
)

func firstAppenderRefs(groups ...appenderRefs) appenderRefs {
	for _, refs := range groups {
		if len(refs) > 0 {
			out := make(appenderRefs, len(refs))
			copy(out, refs)
			return out
		}
	}
	return nil
}

func firstStringRefs(groups ...[]string) []string {
	for _, refs := range groups {
		if len(refs) > 0 {
			out := make([]string, len(refs))
			for index, ref := range refs {
				out[index] = strings.TrimSpace(ref)
			}
			return out
		}
	}
	return nil
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstNonZero(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func parseOptionalDuration(value string) time.Duration {
	text := strings.TrimSpace(value)
	if text == "" {
		return 0
	}
	duration, err := time.ParseDuration(text)
	if err != nil {
		return -1
	}
	return duration
}

func sortedAppenderNames(appenders map[string]appenderConfig) []string {
	names := make([]string, 0, len(appenders))
	for name := range appenders {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sortedLoggerNames(loggers map[string]loggerConfig) []string {
	names := make([]string, 0, len(loggers))
	for name := range loggers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sortedFilterNames(filters map[string]filterConfig) []string {
	names := make([]string, 0, len(filters))
	for name := range filters {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func normalizeKind(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "")
	value = strings.ReplaceAll(value, "_", "")
	return value
}
