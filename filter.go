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
