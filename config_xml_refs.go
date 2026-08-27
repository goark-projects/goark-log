package goarklog

import (
	"fmt"
	"strings"

	"goark.dev/log/internal/configxml"
)

func xmlAppenderType(element string, configured string) string {
	if strings.TrimSpace(configured) != "" {
		return configured
	}
	switch normalizeKind(element) {
	case "rollingfile":
		return "rollingFile"
	default:
		return element
	}
}

func xmlConsoleTarget(target string) string {
	switch strings.ToUpper(strings.TrimSpace(target)) {
	case "SYSTEM_OUT", "STDOUT":
		return "stdout"
	case "SYSTEM_ERR", "STDERR":
		return "stderr"
	default:
		return target
	}
}

func xmlAppenderRefs(refs []xmlAppenderRef) (appenderRefs, error) {
	out := make(appenderRefs, 0, len(refs))
	for _, ref := range refs {
		includeLocation, err := configxml.BoolPointerStrict(ref.IncludeLocation, "includeLocation")
		if err != nil {
			return nil, fmt.Errorf("AppenderRef %q: %w", ref.Ref, err)
		}
		out = append(out, appenderRefConfig{
			Ref:             ref.Ref,
			Level:           ref.Level,
			IncludeLocation: includeLocation,
			FilterRefs:      xmlFilterRefs(ref.FilterRefs),
		})
	}
	return out, nil
}

func xmlFilterRefsFromAppenderRefs(refs []xmlAppenderRef) []string {
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		out = append(out, strings.TrimSpace(ref.Ref))
	}
	return out
}

func xmlRoutes(routes []xmlRoute) map[string]string {
	if len(routes) == 0 {
		return nil
	}
	out := make(map[string]string, len(routes))
	for _, route := range routes {
		key := strings.TrimSpace(route.Key)
		if key == "" {
			continue
		}
		out[key] = firstNonBlank(route.Ref, route.AppenderRef.Ref)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func xmlFilterRefs(refs []xmlFilterRef) []string {
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		out = append(out, strings.TrimSpace(ref.Ref))
	}
	return out
}

func xmlKeyValuePairMap(pairs []xmlKeyValuePair) map[string]string {
	if len(pairs) == 0 {
		return nil
	}
	out := make(map[string]string, len(pairs))
	for _, pair := range pairs {
		key := strings.TrimSpace(pair.Key)
		if key != "" {
			out[key] = pair.Value
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func xmlKeyValuePairs(pairs []xmlKeyValuePair) []keyValuePairConfig {
	out := make([]keyValuePairConfig, 0, len(pairs))
	for _, pair := range pairs {
		out = append(out, keyValuePairConfig{
			Key:   pair.Key,
			Value: pair.Value,
		})
	}
	return out
}

func xmlRemoveAttrs(values []xmlRemoveAttr) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if key := firstNonBlank(value.Key, value.Name); key != "" {
			out = append(out, key)
		}
	}
	return out
}
