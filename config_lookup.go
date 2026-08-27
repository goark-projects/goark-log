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
