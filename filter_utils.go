package goarklog

import (
	"context"
	"fmt"
)

func (o filterOutcome) decide(matched bool) FilterDecision {
	if matched {
		return o.onMatch
	}
	return o.onMismatch
}

func newFilterSettings(onMatch FilterDecision, onMismatch FilterDecision, options ...FilterOption) *filterSettings {
	settings := &filterSettings{
		outcome: filterOutcome{
			onMatch:    onMatch,
			onMismatch: onMismatch,
		},
	}
	for _, option := range options {
		if option != nil {
			option(settings)
		}
	}
	return settings
}

func normalizeFilters(scope string, filters []Filter) ([]Filter, error) {
	if len(filters) == 0 {
		return nil, nil
	}
	out := make([]Filter, 0, len(filters))
	for index, filter := range filters {
		if filter == nil {
			return nil, fmt.Errorf("goark-log: %s filter %d is nil", scope, index)
		}
		out = append(out, filter)
	}
	return out, nil
}

func appendFilters(dst []Filter, src []Filter) []Filter {
	if len(src) == 0 {
		return dst
	}
	return append(dst, src...)
}

func applyFilters(ctx context.Context, filters []Filter, event Event) FilterDecision {
	for _, filter := range filters {
		switch filter.Decide(ctx, event) {
		case FilterDeny:
			return FilterDeny
		case FilterAccept:
			return FilterAccept
		}
	}
	return FilterNeutral
}
