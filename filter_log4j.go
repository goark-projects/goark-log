package goarklog

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"
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
			if ok && attrValueString(value) == want {
				return true
			}
		}
		return false
	}
	for key, want := range f.values {
		value, ok := event.Attr(key)
		if !ok || attrValueString(value) != want {
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

// TimeFilter 按一天内的时间区间匹配事件。
type TimeFilter struct {
	start    time.Duration
	end      time.Duration
	location *time.Location
	outcome  filterOutcome
}

// NewTimeFilter 创建时间区间过滤器。
func NewTimeFilter(start string, end string, options ...FilterOption) (*TimeFilter, error) {
	return newTimeFilter(start, end, nil, options...)
}

// NewTimeFilterInLocation 创建固定时区的时间区间过滤器。
func NewTimeFilterInLocation(start string, end string, location *time.Location, options ...FilterOption) (*TimeFilter, error) {
	if location == nil {
		return nil, fmt.Errorf("goark-log: time filter location is nil")
	}
	return newTimeFilter(start, end, location, options...)
}

func newTimeFilter(start string, end string, location *time.Location, options ...FilterOption) (*TimeFilter, error) {
	startTime, err := parseTimeOfDay(start)
	if err != nil {
		return nil, fmt.Errorf("goark-log: time filter start: %w", err)
	}
	endTime, err := parseTimeOfDay(end)
	if err != nil {
		return nil, fmt.Errorf("goark-log: time filter end: %w", err)
	}
	settings := newFilterSettings(FilterNeutral, FilterDeny, options...)
	return &TimeFilter{start: startTime, end: endTime, location: location, outcome: settings.outcome}, nil
}

func (f *TimeFilter) Decide(_ context.Context, event Event) FilterDecision {
	if f == nil {
		return FilterNeutral
	}
	when := event.Time
	if when.IsZero() {
		when = time.Now()
	}
	if f.location != nil {
		when = when.In(f.location)
	}
	value := time.Duration(when.Hour())*time.Hour +
		time.Duration(when.Minute())*time.Minute +
		time.Duration(when.Second())*time.Second +
		time.Duration(when.Nanosecond())
	if f.start <= f.end {
		return f.outcome.decide(value >= f.start && value <= f.end)
	}
	return f.outcome.decide(value >= f.start || value <= f.end)
}

func parseTimeOfDay(value string) (time.Duration, error) {
	text := strings.TrimSpace(value)
	if text == "" {
		return 0, fmt.Errorf("time is empty")
	}
	layouts := []string{"15:04:05.999999999", "15:04:05", "15:04"}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, text)
		if err == nil {
			return time.Duration(parsed.Hour())*time.Hour +
				time.Duration(parsed.Minute())*time.Minute +
				time.Duration(parsed.Second())*time.Second +
				time.Duration(parsed.Nanosecond()), nil
		}
	}
	return 0, fmt.Errorf("invalid time %q", value)
}

// BurstFilter 对低优先级日志做令牌桶限流。
type BurstFilter struct {
	level    slog.Level
	rate     float64
	maxBurst float64
	outcome  filterOutcome

	mu     sync.Mutex
	tokens float64
	last   time.Time
}

// NewBurstFilter 创建突发限流过滤器。
func NewBurstFilter(level slog.Level, ratePerSecond float64, maxBurst int, options ...FilterOption) (*BurstFilter, error) {
	if ratePerSecond <= 0 {
		return nil, fmt.Errorf("goark-log: burst filter rate must be > 0")
	}
	if maxBurst <= 0 {
		return nil, fmt.Errorf("goark-log: burst filter maxBurst must be > 0")
	}
	settings := newFilterSettings(FilterNeutral, FilterDeny, options...)
	return &BurstFilter{
		level:    level,
		rate:     ratePerSecond,
		maxBurst: float64(maxBurst),
		outcome:  settings.outcome,
		tokens:   float64(maxBurst),
		last:     time.Now(),
	}, nil
}

func (f *BurstFilter) Decide(_ context.Context, event Event) FilterDecision {
	if f == nil {
		return FilterNeutral
	}
	if event.Level > f.level {
		return FilterNeutral
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(f.last).Seconds()
	f.last = now
	f.tokens += elapsed * f.rate
	if f.tokens > f.maxBurst {
		f.tokens = f.maxBurst
	}
	if f.tokens >= 1 {
		f.tokens--
		return f.outcome.onMatch
	}
	return f.outcome.onMismatch
}

// DynamicThresholdFilter 按事件属性动态选择级别阈值。
type DynamicThresholdFilter struct {
	key              string
	defaultThreshold slog.Level
	thresholds       map[string]slog.Level
	outcome          filterOutcome
}

// NewDynamicThresholdFilter 创建动态级别阈值过滤器。
func NewDynamicThresholdFilter(key string, defaultThreshold slog.Level, thresholds map[string]slog.Level, options ...FilterOption) (*DynamicThresholdFilter, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("goark-log: dynamic threshold filter key is empty")
	}
	copied := make(map[string]slog.Level, len(thresholds))
	for value, level := range thresholds {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, fmt.Errorf("goark-log: dynamic threshold filter value is empty")
		}
		copied[value] = level
	}
	settings := newFilterSettings(FilterNeutral, FilterDeny, options...)
	return &DynamicThresholdFilter{
		key:              key,
		defaultThreshold: defaultThreshold,
		thresholds:       copied,
		outcome:          settings.outcome,
	}, nil
}

func (f *DynamicThresholdFilter) Decide(_ context.Context, event Event) FilterDecision {
	if f == nil {
		return FilterNeutral
	}
	threshold := f.defaultThreshold
	if value, ok := event.Attr(f.key); ok {
		if configured, exists := f.thresholds[attrValueString(value)]; exists {
			threshold = configured
		}
	}
	return f.outcome.decide(event.Level >= threshold)
}

func parseFloat(value string, field string) (float64, error) {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0, fmt.Errorf("goark-log: %s is invalid", field)
	}
	return parsed, nil
}
