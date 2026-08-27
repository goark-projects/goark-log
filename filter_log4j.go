package goarklog

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"goark.dev/log/internal/logvalue"
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

// AttrFilter 按属性键和值过滤日志事件。
type AttrFilter struct {
	key     string
	value   string
	outcome filterOutcome
}

// NewAttrFilter 创建属性过滤器。
func NewAttrFilter(key string, value string, options ...FilterOption) (*AttrFilter, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("goark-log: attr filter key is empty")
	}
	settings := newFilterSettings(FilterNeutral, FilterDeny, options...)
	return &AttrFilter{
		key:     key,
		value:   value,
		outcome: settings.outcome,
	}, nil
}

func (f *AttrFilter) Decide(_ context.Context, event Event) FilterDecision {
	if f == nil {
		return FilterNeutral
	}
	value, ok := event.Attr(f.key)
	return f.outcome.decide(ok && logvalue.String(value) == f.value)
}

// StringMatchFilter 按消息子串匹配事件。
type StringMatchFilter struct {
	text    string
	outcome filterOutcome
}

// NewStringMatchFilter 创建消息子串过滤器。
func NewStringMatchFilter(text string, options ...FilterOption) (*StringMatchFilter, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("goark-log: string match filter text is empty")
	}
	settings := newFilterSettings(FilterNeutral, FilterDeny, options...)
	return &StringMatchFilter{text: text, outcome: settings.outcome}, nil
}

func (f *StringMatchFilter) Decide(_ context.Context, event Event) FilterDecision {
	if f == nil {
		return FilterNeutral
	}
	return f.outcome.decide(strings.Contains(event.Message, f.text))
}

// ThresholdFilter 按日志级别下限过滤。
type ThresholdFilter struct {
	level   slog.Level
	outcome filterOutcome
}

// NewThresholdFilter 创建按级别下限过滤的过滤器。
func NewThresholdFilter(level slog.Level, options ...FilterOption) *ThresholdFilter {
	settings := newFilterSettings(FilterNeutral, FilterDeny, options...)
	return &ThresholdFilter{level: level, outcome: settings.outcome}
}

func (f *ThresholdFilter) Decide(_ context.Context, event Event) FilterDecision {
	if f == nil {
		return FilterNeutral
	}
	return f.outcome.decide(event.Level >= f.level)
}

// LevelFilter 按单个日志级别过滤。
type LevelFilter struct {
	level   slog.Level
	outcome filterOutcome
}

// NewLevelFilter 创建按单个级别匹配的过滤器。
func NewLevelFilter(level slog.Level, options ...FilterOption) *LevelFilter {
	settings := newFilterSettings(FilterNeutral, FilterDeny, options...)
	return &LevelFilter{level: level, outcome: settings.outcome}
}

func (f *LevelFilter) Decide(_ context.Context, event Event) FilterDecision {
	if f == nil {
		return FilterNeutral
	}
	return f.outcome.decide(event.Level == f.level)
}

// LevelRangeFilter 按日志级别闭区间过滤。
type LevelRangeFilter struct {
	min     slog.Level
	max     slog.Level
	outcome filterOutcome
}

// NewLevelRangeFilter 创建级别区间过滤器。
func NewLevelRangeFilter(min slog.Level, max slog.Level, options ...FilterOption) (*LevelRangeFilter, error) {
	if min > max {
		return nil, fmt.Errorf("goark-log: filter level range min must be <= max")
	}
	settings := newFilterSettings(FilterNeutral, FilterDeny, options...)
	return &LevelRangeFilter{min: min, max: max, outcome: settings.outcome}, nil
}

func (f *LevelRangeFilter) Decide(_ context.Context, event Event) FilterDecision {
	if f == nil {
		return FilterNeutral
	}
	return f.outcome.decide(event.Level >= f.min && event.Level <= f.max)
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

// ThreadContextStackFilter 按 context stack/NDC 值匹配事件。
type ThreadContextStackFilter struct {
	value   string
	outcome filterOutcome
}

// NewThreadContextStackFilter 创建 context stack 过滤器。
func NewThreadContextStackFilter(value string, options ...FilterOption) (*ThreadContextStackFilter, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, fmt.Errorf("goark-log: thread context stack filter value is empty")
	}
	settings := newFilterSettings(FilterNeutral, FilterDeny, options...)
	return &ThreadContextStackFilter{value: value, outcome: settings.outcome}, nil
}

// Decide 判断事件 context stack 是否包含目标值。
func (f *ThreadContextStackFilter) Decide(_ context.Context, event Event) FilterDecision {
	if f == nil {
		return FilterNeutral
	}
	for _, value := range event.ContextStack {
		if value == f.value {
			return f.outcome.onMatch
		}
	}
	return f.outcome.onMismatch
}

// ThrowableFilter 按异常文本匹配事件。
type ThrowableFilter struct {
	pattern *regexp.Regexp
	outcome filterOutcome
}

// NewThrowableFilter 创建异常文本过滤器。
func NewThrowableFilter(pattern string, options ...FilterOption) (*ThrowableFilter, error) {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return nil, fmt.Errorf("goark-log: throwable filter pattern is empty")
	}
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("goark-log: throwable filter pattern %q is invalid: %w", pattern, err)
	}
	settings := newFilterSettings(FilterNeutral, FilterDeny, options...)
	return &ThrowableFilter{pattern: compiled, outcome: settings.outcome}, nil
}

// Decide 判断事件异常文本是否匹配。
func (f *ThrowableFilter) Decide(_ context.Context, event Event) FilterDecision {
	if f == nil {
		return FilterNeutral
	}
	return f.outcome.decide(f.pattern.MatchString(eventErrorString(event)))
}

// StructuredDataFilter 是面向结构化属性的 MapFilter 别名。
type StructuredDataFilter struct {
	*MapFilter
}

// NewStructuredDataFilter 创建结构化属性过滤器。
func NewStructuredDataFilter(values map[string]string, options ...MapFilterOption) (*StructuredDataFilter, error) {
	filter, err := NewMapFilter(values, options...)
	if err != nil {
		return nil, err
	}
	return &StructuredDataFilter{MapFilter: filter}, nil
}
