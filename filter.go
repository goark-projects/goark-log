package goarklog

import (
	"context"
	"fmt"
	"strings"
)

// FilterDecision 表示过滤器对日志事件的裁决。
type FilterDecision uint8

const (
	FilterNeutral FilterDecision = iota
	FilterAccept
	FilterDeny
)

// Filter 是日志事件过滤器。实现必须并发安全。
type Filter interface {
	Decide(ctx context.Context, event Event) FilterDecision
}

// FilterFunc 把普通函数适配为 Filter。
type FilterFunc func(ctx context.Context, event Event) FilterDecision

func (f FilterFunc) Decide(ctx context.Context, event Event) FilterDecision {
	if f == nil {
		return FilterNeutral
	}
	return f(ctx, event)
}

// ParseFilterDecision 解析过滤器裁决名称。
func ParseFilterDecision(value string) (FilterDecision, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "neutral":
		return FilterNeutral, nil
	case "accept":
		return FilterAccept, nil
	case "deny":
		return FilterDeny, nil
	default:
		return FilterNeutral, fmt.Errorf("goark-log: unsupported filter decision %q", value)
	}
}

type filterOutcome struct {
	onMatch    FilterDecision
	onMismatch FilterDecision
}

type filterSettings struct {
	outcome filterOutcome
}

// FilterOption 调整内置过滤器的匹配结果。
type FilterOption func(*filterSettings)

// WithFilterOnMatch 设置匹配时的裁决。
func WithFilterOnMatch(decision FilterDecision) FilterOption {
	return func(settings *filterSettings) {
		settings.outcome.onMatch = decision
	}
}

// WithFilterOnMismatch 设置不匹配时的裁决。
func WithFilterOnMismatch(decision FilterDecision) FilterOption {
	return func(settings *filterSettings) {
		settings.outcome.onMismatch = decision
	}
}

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
