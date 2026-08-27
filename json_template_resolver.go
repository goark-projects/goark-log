package goarklog

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"goark.dev/log/internal/jsoncodec"
	"goark.dev/log/internal/textutil"
	"goark.dev/log/internal/timepattern"
)

func compileJSONTemplateResolver(raw json.RawMessage, registry *PluginRegistry, layoutOptions LayoutOptions) (JSONTemplateResolver, error) {
	var object map[string]json.RawMessage
	if err := jsoncodec.Unmarshal(raw, &object); err == nil {
		if resolverRaw, ok := object["$resolver"]; ok {
			var name string
			if err := jsoncodec.Unmarshal(resolverRaw, &name); err != nil {
				return nil, fmt.Errorf("$resolver must be a string")
			}
			return newJSONTemplateResolver(name, object, registry, layoutOptions)
		}
	}
	return rawJSONResolver{raw: append([]byte(nil), raw...)}, nil
}

func newJSONTemplateResolver(name string, options map[string]json.RawMessage, registry *PluginRegistry, layoutOptions LayoutOptions) (JSONTemplateResolver, error) {
	switch textutil.NormalizeKind(name) {
	case "timestamp", "time":
		format := jsonTemplateStringOption(options, "format")
		layout, unix := timepattern.Normalize(format)
		return timestampJSONResolver{layout: layout, unix: unix}, nil
	case "level":
		return levelJSONResolver{field: jsonTemplateStringOption(options, "field")}, nil
	case "logger", "loggername":
		return loggerJSONResolver{precision: jsonTemplateIntOption(options, "precision")}, nil
	case "message", "msg":
		return messageJSONResolver{}, nil
	case "thread":
		return threadJSONResolver{}, nil
	case "threadname":
		return threadJSONResolver{}, nil
	case "marker":
		return markerJSONResolver{}, nil
	case "throwable", "exception", "thrown":
		return throwableJSONResolver{
			field:              jsonTemplateStringOption(options, "field"),
			stacktraceAsString: layoutOptions.StacktraceAsString,
		}, nil
	case "rootcause":
		return throwableJSONResolver{field: "rootCause"}, nil
	case "stacktrace":
		field := "stackTrace"
		if layoutOptions.StacktraceAsString {
			field = "stackTraceAsString"
		}
		return throwableJSONResolver{field: field}, nil
	case "source", "location":
		return sourceJSONResolver{}, nil
	case "process":
		return processJSONResolver{}, nil
	case "contextstack", "ndc":
		return contextStackJSONResolver{}, nil
	case "mdc", "contextmap", "attrs":
		return attrsJSONResolver{
			flatten:          jsonTemplateBoolOption(options, "flatten"),
			propertiesAsList: layoutOptions.PropertiesAsList || jsonTemplateBoolOption(options, "propertiesAsList"),
		}, nil
	case "attr":
		key := jsonTemplateStringOption(options, "key")
		if strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("attr resolver requires key")
		}
		return attrJSONResolver{key: key}, nil
	case "endofbatch":
		return endOfBatchJSONResolver{}, nil
	default:
		if factory, ok := registry.jsonTemplateResolverFactory(name); ok {
			return factory(JSONTemplateResolverBuildConfig{Name: name, Options: copyJSONRawOptions(options)})
		}
		return nil, fmt.Errorf("unsupported resolver %q", name)
	}
}

func jsonTemplateStringOption(options map[string]json.RawMessage, key string) string {
	raw, ok := options[key]
	if !ok {
		return ""
	}
	var value string
	if err := jsoncodec.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return strings.TrimSpace(value)
}

func jsonTemplateBoolOption(options map[string]json.RawMessage, key string) bool {
	raw, ok := options[key]
	if !ok {
		return false
	}
	var value bool
	return jsoncodec.Unmarshal(raw, &value) == nil && value
}

func jsonTemplateIntOption(options map[string]json.RawMessage, key string) int {
	raw, ok := options[key]
	if !ok {
		return 0
	}
	var value int
	if err := jsoncodec.Unmarshal(raw, &value); err == nil {
		return value
	}
	var text string
	if err := jsoncodec.Unmarshal(raw, &text); err != nil {
		return 0
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil || parsed < 0 {
		return 0
	}
	return parsed
}

func copyJSONRawOptions(options map[string]json.RawMessage) map[string]json.RawMessage {
	copied := make(map[string]json.RawMessage, len(options))
	for key, raw := range options {
		copied[key] = append([]byte(nil), raw...)
	}
	return copied
}
