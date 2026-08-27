package goarklog

import (
	"context"
	"fmt"
	"strings"

	"goark.dev/log/internal/logvalue"
)

// MapFilterOperator 指定 map 过滤器多条件关系。
type MapFilterOperator string

const (
	MapFilterAnd MapFilterOperator = "and"
	MapFilterOr  MapFilterOperator = "or"
)

// ParseMapFilterOperator 解析 map 过滤器条件关系。
func ParseMapFilterOperator(value string) (MapFilterOperator, error) {
	switch MapFilterOperator(strings.ToLower(strings.TrimSpace(value))) {
	case "", MapFilterAnd:
		return MapFilterAnd, nil
	case MapFilterOr:
		return MapFilterOr, nil
	default:
		return "", fmt.Errorf("goark-log: unsupported map filter operator %q", value)
	}
}

// MapFilter 按事件属性键值匹配。
type MapFilter struct {
	values   map[string]string
	operator MapFilterOperator
	outcome  filterOutcome
}

// MapFilterOption 调整 MapFilter。
type MapFilterOption func(*mapFilterSettings)

type mapFilterSettings struct {
	operator MapFilterOperator
	base     filterSettings
}

// WithMapFilterOperator 设置多个键值条件的组合关系。
func WithMapFilterOperator(operator MapFilterOperator) MapFilterOption {
	return func(settings *mapFilterSettings) {
		settings.operator = operator
	}
}

// WithMapFilterOnMatch 设置匹配时裁决。
func WithMapFilterOnMatch(decision FilterDecision) MapFilterOption {
	return func(settings *mapFilterSettings) {
		settings.base.outcome.onMatch = decision
	}
}

// WithMapFilterOnMismatch 设置不匹配时裁决。
func WithMapFilterOnMismatch(decision FilterDecision) MapFilterOption {
	return func(settings *mapFilterSettings) {
		settings.base.outcome.onMismatch = decision
	}
}

// NewMapFilter 创建属性 map 过滤器。
func NewMapFilter(values map[string]string, options ...MapFilterOption) (*MapFilter, error) {
	settings := mapFilterSettings{
		operator: MapFilterAnd,
		base:     *newFilterSettings(FilterNeutral, FilterDeny),
	}
	for _, option := range options {
		if option != nil {
			option(&settings)
		}
	}
	if _, err := ParseMapFilterOperator(string(settings.operator)); err != nil {
		return nil, err
	}
	normalized := make(map[string]string, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("goark-log: map filter key is empty")
		}
		normalized[key] = value
	}
	if len(normalized) == 0 {
		return nil, fmt.Errorf("goark-log: map filter requires at least one value")
	}
	return &MapFilter{values: normalized, operator: settings.operator, outcome: settings.base.outcome}, nil
}

func (f *MapFilter) Decide(_ context.Context, event Event) FilterDecision {
	if f == nil {
		return FilterNeutral
	}
	matched := f.matches(event)
	return f.outcome.decide(matched)
}

func (f *MapFilter) matches(event Event) bool {
	if f.operator == MapFilterOr {
		for key, want := range f.values {
			value, ok := event.Attr(key)
			if ok && logvalue.String(value) == want {
				return true
			}
		}
		return false
	}
	for key, want := range f.values {
		value, ok := event.Attr(key)
		if !ok || logvalue.String(value) != want {
			return false
		}
	}
	return true
}

// ThreadContextMapFilter 是 MDC 场景下的 MapFilter 别名实现。
type ThreadContextMapFilter struct {
	*MapFilter
}

// NewThreadContextMapFilter 创建 MDC 键值过滤器。
func NewThreadContextMapFilter(values map[string]string, options ...MapFilterOption) (*ThreadContextMapFilter, error) {
	filter, err := NewMapFilter(values, options...)
	if err != nil {
		return nil, err
	}
	return &ThreadContextMapFilter{MapFilter: filter}, nil
}
