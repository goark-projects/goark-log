package router

import "strings"

func routePlanFromConfig(config *runtimeConfig, name string) Plan {
	name = strings.TrimSpace(name)
	for _, logger := range config.loggers {
		if !loggerMatches(name, logger.name) {
			continue
		}
		return Plan{Route: logger.route, GlobalFilters: config.globalFilters}
	}
	return Plan{Route: config.root, GlobalFilters: config.globalFilters}
}
