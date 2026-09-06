package router

import (
	"log/slog"
	"sort"
	"strings"
)

func applyLevelOverrides(base *runtimeConfig, levels map[string]slog.Level) *runtimeConfig {
	if base == nil || len(levels) == 0 {
		return base
	}
	config := *base
	config.loggers = append([]loggerRuntime(nil), base.loggers...)
	names := make([]string, 0, len(levels))
	for name := range levels {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		level := levels[name]
		if name == "ROOT" {
			continue
		}
		found := false
		for index := range config.loggers {
			if config.loggers[index].name != name {
				continue
			}
			config.loggers[index].configuredLevel = levelPointer(level)
			config.loggers[index].route.Level = level
			found = true
			break
		}
		if found {
			continue
		}
		route := routePlanFromConfig(base, name).Route
		config.loggers = append(
			config.loggers,
			loggerRuntime{name: name, configuredLevel: levelPointer(level), route: route},
		)
	}
	sort.Slice(config.loggers, func(i, j int) bool {
		return loggerSpecificity(config.loggers[i].name) > loggerSpecificity(config.loggers[j].name)
	})
	config.root.Level = effectiveRootLevel(base.root.Level, levels)
	for index := range config.loggers {
		config.loggers[index].route.Level = effectiveLoggerLevel(
			config.root.Level,
			config.loggers[index].name,
			config.loggers,
		)
	}
	return &config
}

func effectiveRootLevel(configured slog.Level, levels map[string]slog.Level) slog.Level {
	if level, exists := levels["ROOT"]; exists {
		return level
	}
	return configured
}

func effectiveLoggerLevel(root slog.Level, name string, loggers []loggerRuntime) slog.Level {
	for _, logger := range loggers {
		if logger.configuredLevel != nil && loggerMatches(name, logger.name) {
			return *logger.configuredLevel
		}
	}
	return root
}

func normalizeLevelName(name string) string {
	name = strings.TrimSpace(name)
	if strings.EqualFold(name, "root") {
		return "ROOT"
	}
	return name
}

func cloneLevel(level *slog.Level) *slog.Level {
	if level == nil {
		return nil
	}
	return levelPointer(*level)
}

func levelPointer(level slog.Level) *slog.Level {
	copy := level
	return &copy
}

func loggerMatches(name string, rule string) bool {
	return name == rule || strings.HasPrefix(name, rule+".")
}

func loggerSpecificity(name string) int {
	return strings.Count(name, ".")*1024 + len(name)
}

func loggerLevel(root slog.Level, level *slog.Level) slog.Level {
	if level == nil {
		return root
	}
	return *level
}
