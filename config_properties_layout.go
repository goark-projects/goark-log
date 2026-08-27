package goarklog

import (
	"fmt"
	"strings"
)

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
		config.Remove = propertyList(value)
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
	case "disableAnsi", "disable-ansi":
		parsed, err := parsePropertyBool(value, key)
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
