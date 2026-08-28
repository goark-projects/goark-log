package plugin

import (
	"strings"

	internallayout "goark.dev/log/internal/layout"
)

func registerBuiltInLayouts(registry *Registry) {
	registerLayoutAliases(registry, buildPatternLayoutPlugin, "pattern")
	registerLayoutAliases(registry, func(_ LayoutBuildConfig) (internallayout.Layout, error) {
		return internallayout.TextLayout{}, nil
	}, "text")
	registerLayoutAliases(registry, func(config LayoutBuildConfig) (internallayout.Layout, error) {
		return internallayout.NewJSONLayout(config.Options), nil
	}, "json")
	registerLayoutAliases(registry, func(config LayoutBuildConfig) (internallayout.Layout, error) {
		if config.Registry == nil {
			config.Registry = registry
		}
		return buildJSONTemplateLayoutPlugin(config)
	}, "jsonTemplate")
	registerLayoutAliases(registry, func(config LayoutBuildConfig) (internallayout.Layout, error) {
		return internallayout.NewXMLLayout(config.Options), nil
	}, "xml", "xmlLayout")
	registerLayoutAliases(registry, func(config LayoutBuildConfig) (internallayout.Layout, error) {
		return internallayout.NewCSVLayout(config.Options), nil
	}, "csv", "csvLayout")
	registerLayoutAliases(registry, func(config LayoutBuildConfig) (internallayout.Layout, error) {
		return internallayout.NewGELFLayout(config.Options), nil
	}, "gelf", "gelfLayout")
	registerLayoutAliases(registry, func(_ LayoutBuildConfig) (internallayout.Layout, error) {
		return internallayout.RFC5424Layout{}, nil
	}, "rfc5424", "rfc5424Layout")
	registerLayoutAliases(registry, func(_ LayoutBuildConfig) (internallayout.Layout, error) {
		return internallayout.SyslogLayout{}, nil
	}, "syslog", "syslogLayout")
	registerLayoutAliases(registry, func(config LayoutBuildConfig) (internallayout.Layout, error) {
		return internallayout.NewYAMLLayout(config.Options), nil
	}, "yaml", "yamlLayout")
	registerLayoutAliases(registry, func(config LayoutBuildConfig) (internallayout.Layout, error) {
		return internallayout.NewHTMLLayout(config.Options), nil
	}, "html", "htmlLayout")
}

func registerLayoutAliases(registry *Registry, factory LayoutFactory, aliases ...string) {
	for _, alias := range aliases {
		_ = registry.RegisterLayout(alias, factory)
	}
}

func buildPatternLayoutPlugin(config LayoutBuildConfig) (internallayout.Layout, error) {
	return internallayout.NewPatternLayoutWithOptions(config.Pattern, config.Options)
}

func buildJSONTemplateLayoutPlugin(config LayoutBuildConfig) (internallayout.Layout, error) {
	options := []internallayout.JSONTemplateLayoutOption{
		internallayout.WithJSONTemplateResolverLookup(jsonTemplateResolverLookup(config.Registry)),
		internallayout.WithJSONTemplateLayoutOptions(config.Options),
	}
	if strings.TrimSpace(config.EventTemplateURI) != "" {
		return internallayout.NewJSONTemplateLayoutFromFile(config.EventTemplateURI, options...)
	}
	return internallayout.NewJSONTemplateLayout(config.EventTemplate, options...)
}

func jsonTemplateResolverLookup(registry *Registry) internallayout.JSONTemplateResolverLookup {
	if registry == nil {
		return nil
	}
	return registry.JSONTemplateResolverFactory
}
