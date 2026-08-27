package goarklog

import (
	"context"
	"fmt"
	"strings"
)

// DenyFilter 无条件拒绝事件。
type DenyFilter struct{}

// NewDenyFilter 创建无条件拒绝过滤器。
func NewDenyFilter() *DenyFilter {
	return &DenyFilter{}
}

func (*DenyFilter) Decide(context.Context, Event) FilterDecision {
	return FilterDeny
}

// CompositeFilter 按顺序组合多个过滤器。
type CompositeFilter struct {
	filters []Filter
}

// NewCompositeFilter 创建组合过滤器。
func NewCompositeFilter(filters ...Filter) (*CompositeFilter, error) {
	chain, err := normalizeFilters("composite", filters)
	if err != nil {
		return nil, err
	}
	return &CompositeFilter{filters: chain}, nil
}

func (f *CompositeFilter) Decide(ctx context.Context, event Event) FilterDecision {
	if f == nil {
		return FilterNeutral
	}
	return applyFilters(ctx, f.filters, event)
}

// MarkerFilter 按 marker 名称或父级 marker 匹配事件。
type MarkerFilter struct {
	name    string
	outcome filterOutcome
}

// NewMarkerFilter 创建 marker 过滤器。
func NewMarkerFilter(name string, options ...FilterOption) (*MarkerFilter, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("goark-log: marker filter name is empty")
	}
	settings := newFilterSettings(FilterNeutral, FilterDeny, options...)
	return &MarkerFilter{name: name, outcome: settings.outcome}, nil
}

func (f *MarkerFilter) Decide(_ context.Context, event Event) FilterDecision {
	if f == nil {
		return FilterNeutral
	}
	return f.outcome.decide(event.Marker != nil && event.Marker.Contains(f.name))
}

// NoMarkerFilter 匹配没有 marker 的事件。
type NoMarkerFilter struct {
	outcome filterOutcome
}

// NewNoMarkerFilter 创建无 marker 过滤器。
func NewNoMarkerFilter(options ...FilterOption) *NoMarkerFilter {
	settings := newFilterSettings(FilterNeutral, FilterDeny, options...)
	return &NoMarkerFilter{outcome: settings.outcome}
}

func (f *NoMarkerFilter) Decide(_ context.Context, event Event) FilterDecision {
	if f == nil {
		return FilterNeutral
	}
	return f.outcome.decide(event.Marker == nil)
}
