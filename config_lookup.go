package goarklog

import (
	"fmt"
	"strings"
)

func (c *fileConfig) resolveLookups(lookups *LookupResolver) error {
	if c == nil {
		return nil
	}
	if lookups == nil {
		lookups = NewLookupResolver()
	}
	var err error
	if c.Status, err = resolveStringLookup(lookups, c.Status); err != nil {
		return fmt.Errorf("goark-log: status: %w", err)
	}
	if c.MonitorInterval, err = resolveStringLookup(lookups, c.MonitorInterval); err != nil {
		return fmt.Errorf("goark-log: monitorInterval: %w", err)
	}
	if c.MonitorKebab, err = resolveStringLookup(lookups, c.MonitorKebab); err != nil {
		return fmt.Errorf("goark-log: monitor-interval: %w", err)
	}
	if err := c.resolveProperties(lookups); err != nil {
		return err
	}
	if err := c.resolveCustomLevelLookups(lookups); err != nil {
		return err
	}
	c.FilterRefs, err = resolveStringListLookups(lookups, c.FilterRefs)
	if err != nil {
		return fmt.Errorf("goark-log: filterRefs: %w", err)
	}
	c.FilterRefsKebab, err = resolveStringListLookups(lookups, c.FilterRefsKebab)
	if err != nil {
		return fmt.Errorf("goark-log: filter-refs: %w", err)
	}
	if err := c.AsyncLogger.resolveLookups(lookups); err != nil {
		return fmt.Errorf("goark-log: asyncLogger: %w", err)
	}
	if err := c.AsyncLoggerKebab.resolveLookups(lookups); err != nil {
		return fmt.Errorf("goark-log: async-logger: %w", err)
	}
	if err := c.Async.resolveLookups(lookups); err != nil {
		return fmt.Errorf("goark-log: async: %w", err)
	}
	for name, spec := range c.Appenders {
		if err := spec.resolveLookups(lookups); err != nil {
			return fmt.Errorf("goark-log: appender %q: %w", name, err)
		}
		c.Appenders[name] = spec
	}
	for name, spec := range c.Filters {
		if err := spec.resolveLookups(lookups); err != nil {
			return fmt.Errorf("goark-log: filter %q: %w", name, err)
		}
		c.Filters[name] = spec
	}
	if err := c.Root.resolveLookups(lookups); err != nil {
		return fmt.Errorf("goark-log: root: %w", err)
	}
	for name, spec := range c.Loggers {
		if err := spec.resolveLookups(lookups); err != nil {
			return fmt.Errorf("goark-log: logger %q: %w", name, err)
		}
		c.Loggers[name] = spec
	}
	return nil
}

func (c *loggerConfig) resolveLookups(lookups *LookupResolver) error {
	var err error
	if c.Level, err = resolveStringLookup(lookups, c.Level); err != nil {
		return fmt.Errorf("level: %w", err)
	}
	if c.AppenderRefs, err = c.AppenderRefs.resolveLookups(lookups); err != nil {
		return fmt.Errorf("appenderRefs: %w", err)
	}
	if c.AppenderRefsKebab, err = c.AppenderRefsKebab.resolveLookups(lookups); err != nil {
		return fmt.Errorf("appender-refs: %w", err)
	}
	if c.Refs, err = c.Refs.resolveLookups(lookups); err != nil {
		return fmt.Errorf("refs: %w", err)
	}
	if c.Filters, err = resolveStringListLookups(lookups, c.Filters); err != nil {
		return fmt.Errorf("filters: %w", err)
	}
	if c.FilterRefs, err = resolveStringListLookups(lookups, c.FilterRefs); err != nil {
		return fmt.Errorf("filterRefs: %w", err)
	}
	if c.FilterRefsKebab, err = resolveStringListLookups(lookups, c.FilterRefsKebab); err != nil {
		return fmt.Errorf("filter-refs: %w", err)
	}
	return nil
}

func (c *filterConfig) resolveLookups(lookups *LookupResolver) error {
	var err error
	if c.Type, err = resolveStringLookup(lookups, c.Type); err != nil {
		return fmt.Errorf("type: %w", err)
	}
	if c.Level, err = resolveStringLookup(lookups, c.Level); err != nil {
		return fmt.Errorf("level: %w", err)
	}
	if c.MinLevel, err = resolveStringLookup(lookups, c.MinLevel); err != nil {
		return fmt.Errorf("minLevel: %w", err)
	}
	if c.MinLevelKebab, err = resolveStringLookup(lookups, c.MinLevelKebab); err != nil {
		return fmt.Errorf("min-level: %w", err)
	}
	if c.MaxLevel, err = resolveStringLookup(lookups, c.MaxLevel); err != nil {
		return fmt.Errorf("maxLevel: %w", err)
	}
	if c.MaxLevelKebab, err = resolveStringLookup(lookups, c.MaxLevelKebab); err != nil {
		return fmt.Errorf("max-level: %w", err)
	}
	if c.Marker, err = resolveStringLookup(lookups, c.Marker); err != nil {
		return fmt.Errorf("marker: %w", err)
	}
	if c.Text, err = resolveStringLookup(lookups, c.Text); err != nil {
		return fmt.Errorf("text: %w", err)
	}
	if c.Operator, err = resolveStringLookup(lookups, c.Operator); err != nil {
		return fmt.Errorf("operator: %w", err)
	}
	if c.Start, err = resolveStringLookup(lookups, c.Start); err != nil {
		return fmt.Errorf("start: %w", err)
	}
	if c.End, err = resolveStringLookup(lookups, c.End); err != nil {
		return fmt.Errorf("end: %w", err)
	}
	if c.Timezone, err = resolveStringLookup(lookups, c.Timezone); err != nil {
		return fmt.Errorf("timezone: %w", err)
	}
	if c.Rate, err = resolveStringLookup(lookups, c.Rate); err != nil {
		return fmt.Errorf("rate: %w", err)
	}
	if c.Field, err = resolveStringLookup(lookups, c.Field); err != nil {
		return fmt.Errorf("field: %w", err)
	}
	if c.Key, err = resolveStringLookup(lookups, c.Key); err != nil {
		return fmt.Errorf("key: %w", err)
	}
	if c.Value, err = resolveStringLookup(lookups, c.Value); err != nil {
		return fmt.Errorf("value: %w", err)
	}
	if c.Values, err = resolveStringMapLookups(lookups, c.Values); err != nil {
		return fmt.Errorf("values: %w", err)
	}
	if c.Thresholds, err = resolveStringMapLookups(lookups, c.Thresholds); err != nil {
		return fmt.Errorf("thresholds: %w", err)
	}
	if c.Filters, err = resolveStringListLookups(lookups, c.Filters); err != nil {
		return fmt.Errorf("filters: %w", err)
	}
	if c.FilterRefs, err = resolveStringListLookups(lookups, c.FilterRefs); err != nil {
		return fmt.Errorf("filterRefs: %w", err)
	}
	if c.FilterRefsKebab, err = resolveStringListLookups(lookups, c.FilterRefsKebab); err != nil {
		return fmt.Errorf("filter-refs: %w", err)
	}
	if c.KeyValuePair, err = resolveKeyValuePairLookups(lookups, c.KeyValuePair); err != nil {
		return fmt.Errorf("KeyValuePair: %w", err)
	}
	if c.KeyValuePairs, err = resolveKeyValuePairLookups(lookups, c.KeyValuePairs); err != nil {
		return fmt.Errorf("keyValuePairs: %w", err)
	}
	if c.KeyValuePairsKebab, err = resolveKeyValuePairLookups(lookups, c.KeyValuePairsKebab); err != nil {
		return fmt.Errorf("key-value-pairs: %w", err)
	}
	if c.DefaultThreshold, err = resolveStringLookup(lookups, c.DefaultThreshold); err != nil {
		return fmt.Errorf("defaultThreshold: %w", err)
	}
	if c.DefaultKebab, err = resolveStringLookup(lookups, c.DefaultKebab); err != nil {
		return fmt.Errorf("default-threshold: %w", err)
	}
	if c.Pattern, err = resolveStringLookup(lookups, c.Pattern); err != nil {
		return fmt.Errorf("pattern: %w", err)
	}
	if c.OnMatch, err = resolveStringLookup(lookups, c.OnMatch); err != nil {
		return fmt.Errorf("onMatch: %w", err)
	}
	if c.OnMatchKebab, err = resolveStringLookup(lookups, c.OnMatchKebab); err != nil {
		return fmt.Errorf("on-match: %w", err)
	}
	if c.OnMismatch, err = resolveStringLookup(lookups, c.OnMismatch); err != nil {
		return fmt.Errorf("onMismatch: %w", err)
	}
	if c.OnMismatchKebab, err = resolveStringLookup(lookups, c.OnMismatchKebab); err != nil {
		return fmt.Errorf("on-mismatch: %w", err)
	}
	return nil
}

func (c *asyncLoggerConfig) resolveLookups(lookups *LookupResolver) error {
	var err error
	if c.OverflowStrategy, err = resolveStringLookup(lookups, c.OverflowStrategy); err != nil {
		return fmt.Errorf("overflowStrategy: %w", err)
	}
	if c.OverflowStrategyKebab, err = resolveStringLookup(lookups, c.OverflowStrategyKebab); err != nil {
		return fmt.Errorf("overflow-strategy: %w", err)
	}
	if c.WaitStrategy, err = resolveStringLookup(lookups, c.WaitStrategy); err != nil {
		return fmt.Errorf("waitStrategy: %w", err)
	}
	if c.WaitStrategyKebab, err = resolveStringLookup(lookups, c.WaitStrategyKebab); err != nil {
		return fmt.Errorf("wait-strategy: %w", err)
	}
	if c.SleepTime, err = resolveStringLookup(lookups, c.SleepTime); err != nil {
		return fmt.Errorf("sleepTime: %w", err)
	}
	if c.SleepTimeKebab, err = resolveStringLookup(lookups, c.SleepTimeKebab); err != nil {
		return fmt.Errorf("sleep-time: %w", err)
	}
	if c.Timeout, err = resolveStringLookup(lookups, c.Timeout); err != nil {
		return fmt.Errorf("timeout: %w", err)
	}
	return nil
}

func (c *fileConfig) resolveProperties(lookups *LookupResolver) error {
	if len(c.Properties) == 0 {
		return nil
	}
	resolved := make(map[string]string, len(c.Properties))
	for key, value := range c.Properties {
		key = strings.TrimSpace(key)
		if key == "" {
			return fmt.Errorf("goark-log: property name is empty")
		}
		out, err := resolveStringLookup(lookups, value)
		if err != nil {
			return fmt.Errorf("goark-log: property %q: %w", key, err)
		}
		resolved[key] = out
	}
	c.Properties = resolved
	lookups.Register("prop", func(key string) (string, bool) {
		value, ok := resolved[strings.TrimSpace(key)]
		return value, ok
	})
	lookups.Register("property", func(key string) (string, bool) {
		value, ok := resolved[strings.TrimSpace(key)]
		return value, ok
	})
	return nil
}

func resolveStringLookup(lookups *LookupResolver, value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return value, nil
	}
	return lookups.Resolve(value)
}

func resolveStringListLookups(lookups *LookupResolver, values []string) ([]string, error) {
	if len(values) == 0 {
		return values, nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		resolved, err := resolveStringLookup(lookups, value)
		if err != nil {
			return nil, err
		}
		out = append(out, resolved)
	}
	return out, nil
}

func resolveStringMapLookups(lookups *LookupResolver, values map[string]string) (map[string]string, error) {
	if len(values) == 0 {
		return values, nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		resolvedKey, err := resolveStringLookup(lookups, key)
		if err != nil {
			return nil, err
		}
		resolvedValue, err := resolveStringLookup(lookups, value)
		if err != nil {
			return nil, err
		}
		out[resolvedKey] = resolvedValue
	}
	return out, nil
}

func resolveKeyValuePairLookups(lookups *LookupResolver, values []keyValuePairConfig) ([]keyValuePairConfig, error) {
	if len(values) == 0 {
		return values, nil
	}
	out := make([]keyValuePairConfig, 0, len(values))
	for index, value := range values {
		resolved, err := value.resolveLookups(lookups)
		if err != nil {
			return nil, fmt.Errorf("%d: %w", index, err)
		}
		if strings.TrimSpace(resolved.Key) == "" {
			return nil, fmt.Errorf("%d: key is empty", index)
		}
		out = append(out, resolved)
	}
	return out, nil
}

func (c keyValuePairConfig) resolveLookups(lookups *LookupResolver) (keyValuePairConfig, error) {
	var err error
	if c.Key, err = resolveStringLookup(lookups, c.Key); err != nil {
		return keyValuePairConfig{}, fmt.Errorf("key: %w", err)
	}
	if c.Value, err = resolveStringLookup(lookups, c.Value); err != nil {
		return keyValuePairConfig{}, fmt.Errorf("value: %w", err)
	}
	return c, nil
}

func (c *fileConfig) resolveCustomLevelLookups(lookups *LookupResolver) error {
	if c == nil {
		return nil
	}
	resolved, err := resolveStringMapLookups(lookups, c.CustomLevels)
	if err != nil {
		return fmt.Errorf("goark-log: customLevels: %w", err)
	}
	c.CustomLevels = resolved
	resolvedKebab, err := resolveStringMapLookups(lookups, c.CustomLevelsKebab)
	if err != nil {
		return fmt.Errorf("goark-log: custom-levels: %w", err)
	}
	c.CustomLevelsKebab = resolvedKebab
	return nil
}
