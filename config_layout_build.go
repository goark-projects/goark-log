package goarklog

import (
	"fmt"
)

func buildLayout(config layoutConfig, registry *PluginRegistry) (Layout, error) {
	factory, ok := registry.layoutFactory(config.Type)
	if !ok {
		return nil, fmt.Errorf("unsupported layout type %q", config.Type)
	}
	return factory(LayoutBuildConfig{
		Type:             config.Type,
		Pattern:          config.Pattern,
		EventTemplate:    config.eventTemplate(),
		EventTemplateURI: config.eventTemplateURI(),
		Options:          config.options(),
		Registry:         registry,
	})
}

func (c layoutConfig) eventTemplate() string {
	return firstNonBlank(c.EventTemplate, c.EventTemplateKebab)
}

func (c layoutConfig) eventTemplateURI() string {
	return firstNonBlank(c.EventTemplateURI, c.EventTemplateURIKebab, c.EventTemplatePath, c.EventTemplatePathKebab)
}

func (c layoutConfig) options() LayoutOptions {
	return LayoutOptions{
		Compact:              c.Compact,
		EventEOL:             c.EventEOL || c.EventEOLKebab,
		Complete:             c.Complete,
		IncludeStacktrace:    c.IncludeStacktrace || c.IncludeStacktraceKebab,
		StacktraceAsString:   c.StacktraceAsString || c.StacktraceAsStringKebab,
		PropertiesAsList:     c.PropertiesAsList || c.PropertiesAsListKebab,
		IncludeNullDelimiter: c.IncludeNullDelimiter || c.IncludeNullDelimiterKebab,
		DisableANSI:          c.DisableANSI || c.DisableANSIKebab,
		Header:               c.Header,
		Footer:               c.Footer,
	}
}
