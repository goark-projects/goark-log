package goarklog

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

func (r *appenderRefs) UnmarshalYAML(node *yaml.Node) error {
	if node == nil || node.Kind == 0 {
		return nil
	}
	if node.Kind != yaml.SequenceNode {
		return fmt.Errorf("goark-log: appenderRefs must be a sequence")
	}
	refs := make([]appenderRefConfig, 0, len(node.Content))
	for index, item := range node.Content {
		switch item.Kind {
		case yaml.ScalarNode:
			var ref string
			if err := item.Decode(&ref); err != nil {
				return fmt.Errorf("goark-log: appenderRefs[%d]: %w", index, err)
			}
			refs = append(refs, appenderRefConfig{Ref: ref})
		case yaml.MappingNode:
			var ref appenderRefConfig
			if err := item.Decode(&ref); err != nil {
				return fmt.Errorf("goark-log: appenderRefs[%d]: %w", index, err)
			}
			refs = append(refs, ref)
		default:
			return fmt.Errorf("goark-log: appenderRefs[%d] must be a string or object", index)
		}
	}
	*r = refs
	return nil
}

func (r appenderRefs) strings() []string {
	if len(r) == 0 {
		return nil
	}
	refs := make([]string, 0, len(r))
	for _, ref := range r {
		if ref.hasControls() {
			continue
		}
		refs = append(refs, strings.TrimSpace(ref.Ref))
	}
	return refs
}

func (r appenderRefs) controls(filters map[string]Filter) ([]AppenderRef, error) {
	if len(r) == 0 {
		return nil, nil
	}
	controls := make([]AppenderRef, 0, len(r))
	for _, ref := range r {
		if !ref.hasControls() {
			continue
		}
		control, err := ref.build(filters)
		if err != nil {
			return nil, err
		}
		controls = append(controls, control)
	}
	return controls, nil
}

func (r appenderRefs) resolveLookups(lookups *LookupResolver) (appenderRefs, error) {
	if len(r) == 0 {
		return r, nil
	}
	out := make(appenderRefs, 0, len(r))
	for index, ref := range r {
		resolved, err := ref.resolveLookups(lookups)
		if err != nil {
			return nil, fmt.Errorf("appenderRefs[%d]: %w", index, err)
		}
		out = append(out, resolved)
	}
	return out, nil
}

func (c appenderRefConfig) hasControls() bool {
	return strings.TrimSpace(c.Level) != "" ||
		c.IncludeLocation != nil ||
		c.IncludeLocationKebab != nil ||
		len(c.Filters) > 0 ||
		len(c.FilterRefs) > 0 ||
		len(c.FilterRefsKebab) > 0
}

func (c appenderRefConfig) filterRefs() []string {
	return firstStringRefs(c.Filters, c.FilterRefs, c.FilterRefsKebab)
}

func (c appenderRefConfig) build(filters map[string]Filter) (AppenderRef, error) {
	ref := AppenderRef{Ref: strings.TrimSpace(c.Ref)}
	if strings.TrimSpace(c.Level) != "" {
		level, err := ParseLevel(c.Level)
		if err != nil {
			return AppenderRef{}, err
		}
		ref.Level = &level
	}
	if includeLocation := c.includeLocationPointer(); includeLocation != nil {
		value := *includeLocation
		ref.IncludeLocation = &value
	}
	resolved, err := resolveFilters(filters, c.filterRefs())
	if err != nil {
		return AppenderRef{}, err
	}
	ref.Filters = resolved
	return ref, nil
}

func (c appenderRefConfig) includeLocationPointer() *bool {
	if c.IncludeLocation != nil {
		value := *c.IncludeLocation
		return &value
	}
	if c.IncludeLocationKebab != nil {
		value := *c.IncludeLocationKebab
		return &value
	}
	return nil
}

func (c appenderRefConfig) resolveLookups(lookups *LookupResolver) (appenderRefConfig, error) {
	var err error
	if c.Ref, err = resolveStringLookup(lookups, c.Ref); err != nil {
		return appenderRefConfig{}, fmt.Errorf("ref: %w", err)
	}
	if c.Level, err = resolveStringLookup(lookups, c.Level); err != nil {
		return appenderRefConfig{}, fmt.Errorf("level: %w", err)
	}
	if c.Filters, err = resolveStringListLookups(lookups, c.Filters); err != nil {
		return appenderRefConfig{}, fmt.Errorf("filters: %w", err)
	}
	if c.FilterRefs, err = resolveStringListLookups(lookups, c.FilterRefs); err != nil {
		return appenderRefConfig{}, fmt.Errorf("filterRefs: %w", err)
	}
	if c.FilterRefsKebab, err = resolveStringListLookups(lookups, c.FilterRefsKebab); err != nil {
		return appenderRefConfig{}, fmt.Errorf("filter-refs: %w", err)
	}
	return c, nil
}
