package goarklog

import (
	"context"
	"log/slog"
	"sort"
	"strings"
)

type rewriteBuildConfig struct {
	Attrs            map[string]string `yaml:"attrs"`
	Attributes       map[string]string `yaml:"attributes"`
	Properties       map[string]string `yaml:"properties"`
	Remove           []string          `yaml:"remove"`
	RemoveAttrs      []string          `yaml:"removeAttrs"`
	RemoveAttrsKebab []string          `yaml:"remove-attrs"`
}

func (c rewriteBuildConfig) attrs() map[string]string {
	attrs := mergeStringMaps(copyStringMap(c.Attrs), c.Attributes)
	return mergeStringMaps(attrs, c.Properties)
}

func (c rewriteBuildConfig) removeKeys() []string {
	return firstStringRefs(c.Remove, c.RemoveAttrs, c.RemoveAttrsKebab)
}

func (c *rewriteBuildConfig) resolveLookups(lookups *LookupResolver) error {
	var err error
	if c.Attrs, err = resolveStringMapLookups(lookups, c.Attrs); err != nil {
		return err
	}
	if c.Attributes, err = resolveStringMapLookups(lookups, c.Attributes); err != nil {
		return err
	}
	if c.Properties, err = resolveStringMapLookups(lookups, c.Properties); err != nil {
		return err
	}
	if c.Remove, err = resolveStringListLookups(lookups, c.Remove); err != nil {
		return err
	}
	if c.RemoveAttrs, err = resolveStringListLookups(lookups, c.RemoveAttrs); err != nil {
		return err
	}
	if c.RemoveAttrsKebab, err = resolveStringListLookups(lookups, c.RemoveAttrsKebab); err != nil {
		return err
	}
	return nil
}

func newAttributeRewritePolicy(config RewriteBuildConfig) RewritePolicy {
	additions := rewriteAdditions(config.Attrs)
	removals := rewriteRemovalSet(config.RemoveAttrs)
	if len(additions) == 0 && len(removals) == 0 {
		return nil
	}
	return func(_ context.Context, event Event) (Event, error) {
		rewritten := event
		attrs := make([]slog.Attr, 0, len(event.Attrs)+len(additions))
		for _, attr := range event.Attrs {
			if _, remove := removals[attr.Key]; remove {
				continue
			}
			attrs = append(attrs, attr)
		}
		attrs = append(attrs, additions...)
		rewritten.Attrs = attrs
		return rewritten, nil
	}
}

func rewriteAdditions(values map[string]string) []slog.Attr {
	if len(values) == 0 {
		return nil
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		key = strings.TrimSpace(key)
		if key != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	attrs := make([]slog.Attr, 0, len(keys))
	for _, key := range keys {
		attrs = append(attrs, slog.String(key, values[key]))
	}
	return attrs
}

func rewriteRemovalSet(keys []string) map[string]struct{} {
	if len(keys) == 0 {
		return nil
	}
	removals := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key != "" {
			removals[key] = struct{}{}
		}
	}
	return removals
}
