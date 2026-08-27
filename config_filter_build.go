package goarklog

import (
	"fmt"
	"strings"
)

func (c loggerConfig) refs() []string {
	return c.appenderRefs().strings()
}

func (c loggerConfig) appenderRefs() appenderRefs {
	return firstAppenderRefs(c.AppenderRefs, c.AppenderRefsKebab, c.Refs)
}

func (c loggerConfig) appenderRefControls(filters map[string]Filter) ([]AppenderRef, error) {
	return c.appenderRefs().controls(filters)
}

func (c loggerConfig) filterRefs() []string {
	return firstStringRefs(c.Filters, c.FilterRefs, c.FilterRefsKebab)
}

func (c filterConfig) filterRefs() []string {
	return firstStringRefs(c.Filters, c.FilterRefs, c.FilterRefsKebab)
}

func (c *fileConfig) filterRefs() []string {
	if c == nil {
		return nil
	}
	return firstStringRefs(c.FilterRefs, c.FilterRefsKebab)
}

func (c *fileConfig) buildFilters(registry *PluginRegistry) (map[string]Filter, error) {
	if len(c.Filters) == 0 {
		return nil, nil
	}
	names := sortedFilterNames(c.Filters)
	filters := make(map[string]Filter, len(c.Filters))
	visiting := make(map[string]bool, len(c.Filters))
	for _, name := range names {
		filter, err := c.buildFilter(name, registry, filters, visiting)
		if err != nil {
			return nil, err
		}
		filters[name] = filter
	}
	return filters, nil
}

func (c *fileConfig) buildFilter(name string, registry *PluginRegistry, filters map[string]Filter, visiting map[string]bool) (Filter, error) {
	if filter, ok := filters[name]; ok {
		return filter, nil
	}
	if visiting[name] {
		return nil, fmt.Errorf("goark-log: filter %q has cyclic filterRefs", name)
	}
	spec, ok := c.Filters[name]
	if !ok {
		return nil, fmt.Errorf("goark-log: filter %q is not configured", name)
	}
	visiting[name] = true
	defer delete(visiting, name)
	nested, err := c.resolveNestedFilters(spec.filterRefs(), registry, filters, visiting)
	if err != nil {
		return nil, fmt.Errorf("goark-log: filter %q: %w", name, err)
	}
	filter, err := buildFilter(name, spec, nested, registry)
	if err != nil {
		return nil, err
	}
	filters[name] = filter
	return filter, nil
}

func (c *fileConfig) resolveNestedFilters(refs []string, registry *PluginRegistry, filters map[string]Filter, visiting map[string]bool) ([]Filter, error) {
	if len(refs) == 0 {
		return nil, nil
	}
	nested := make([]Filter, 0, len(refs))
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			return nil, fmt.Errorf("filter ref is empty")
		}
		filter, err := c.buildFilter(ref, registry, filters, visiting)
		if err != nil {
			return nil, err
		}
		nested = append(nested, filter)
	}
	return nested, nil
}

func buildFilter(name string, spec filterConfig, nested []Filter, registry *PluginRegistry) (Filter, error) {
	if normalizeKind(spec.Type) == "" {
		return nil, fmt.Errorf("goark-log: filter %q type is empty", name)
	}
	factory, ok := registry.filterFactory(spec.Type)
	if !ok {
		return nil, fmt.Errorf("goark-log: unsupported filter %q type %q", name, spec.Type)
	}
	config := spec.filterBuildConfig(name)
	config.Filters = nested
	filter, err := factory(config)
	if err != nil {
		return nil, fmt.Errorf("goark-log: filter %q: %w", name, err)
	}
	return filter, nil
}

func parseFilterDecisionOrDefault(value string, fallback FilterDecision) (FilterDecision, error) {
	if strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	return ParseFilterDecision(value)
}

func parseRegexFilterField(value string) (RegexFilterField, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "message", "msg":
		return RegexFieldMessage, nil
	case "logger", "name":
		return RegexFieldLogger, nil
	case "attr", "attribute":
		return RegexFieldAttr, nil
	default:
		return "", fmt.Errorf("unsupported regex filter field %q", value)
	}
}

func (c filterConfig) minLevel() string {
	return firstNonBlank(c.MinLevel, c.MinLevelKebab)
}

func (c filterConfig) maxLevel() string {
	return firstNonBlank(c.MaxLevel, c.MaxLevelKebab)
}

func (c filterConfig) maxBurst() int {
	if c.MaxBurst != 0 {
		return c.MaxBurst
	}
	return c.MaxBurstKebab
}

func (c filterConfig) defaultThreshold() string {
	return firstNonBlank(c.DefaultThreshold, c.DefaultKebab)
}

func (c filterConfig) values() map[string]string {
	values := copyStringMap(c.Values)
	if isDynamicThresholdFilterKind(c.Type) {
		return values
	}
	return mergeStringMaps(values, c.keyValuePairs())
}

func (c filterConfig) thresholds() map[string]string {
	thresholds := copyStringMap(c.Thresholds)
	if !isDynamicThresholdFilterKind(c.Type) {
		return thresholds
	}
	return mergeStringMaps(thresholds, c.keyValuePairs())
}

func (c filterConfig) keyValuePairs() map[string]string {
	pairs := make(map[string]string)
	for _, groups := range [][]keyValuePairConfig{c.KeyValuePair, c.KeyValuePairs, c.KeyValuePairsKebab} {
		for _, pair := range groups {
			key := strings.TrimSpace(pair.Key)
			if key == "" {
				continue
			}
			pairs[key] = pair.Value
		}
	}
	if len(pairs) == 0 {
		return nil
	}
	return pairs
}

func (c filterConfig) filterBuildConfig(name string) FilterBuildConfig {
	return FilterBuildConfig{
		Name:             name,
		Type:             c.Type,
		Level:            c.Level,
		MinLevel:         c.minLevel(),
		MaxLevel:         c.maxLevel(),
		Marker:           c.Marker,
		Text:             c.Text,
		Operator:         c.Operator,
		Start:            c.Start,
		End:              c.End,
		Timezone:         c.Timezone,
		Rate:             c.Rate,
		MaxBurst:         c.maxBurst(),
		Field:            c.Field,
		Key:              c.Key,
		Value:            c.Value,
		Values:           c.values(),
		Thresholds:       c.thresholds(),
		DefaultThreshold: c.defaultThreshold(),
		Pattern:          c.Pattern,
		OnMatch:          firstNonBlank(c.OnMatch, c.OnMatchKebab),
		OnMismatch:       firstNonBlank(c.OnMismatch, c.OnMismatchKebab),
	}
}

func copyStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func mergeStringMaps(base map[string]string, overlay map[string]string) map[string]string {
	if len(overlay) == 0 {
		return base
	}
	if base == nil {
		base = make(map[string]string, len(overlay))
	}
	for key, value := range overlay {
		base[key] = value
	}
	return base
}

func isDynamicThresholdFilterKind(value string) bool {
	switch normalizeKind(value) {
	case "dynamicthreshold", "dynamicthresholdfilter":
		return true
	default:
		return false
	}
}

func wrapAppenderFilters(name string, appender Appender, refs []string, filters map[string]Filter) (Appender, error) {
	resolved, err := resolveFilters(filters, refs)
	if err != nil {
		return nil, fmt.Errorf("goark-log: appender %q: %w", name, err)
	}
	if len(resolved) == 0 {
		return appender, nil
	}
	return NewFilteredAppender(appender, resolved...)
}

func resolveFilters(filters map[string]Filter, refs []string) ([]Filter, error) {
	if len(refs) == 0 {
		return nil, nil
	}
	resolved := make([]Filter, 0, len(refs))
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			return nil, fmt.Errorf("filter ref is empty")
		}
		filter, ok := filters[ref]
		if !ok {
			return nil, fmt.Errorf("filter %q is not configured", ref)
		}
		resolved = append(resolved, filter)
	}
	return resolved, nil
}
