package filter

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"goark.dev/log/internal/logevent"
	"goark.dev/log/internal/logvalue"
)

// Event 是过滤器看到的事件快照。
type Event = logevent.Event

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

// ScriptEvaluator 是脚本过滤器的执行器契约。
type ScriptEvaluator interface {
	Evaluate(ctx context.Context, event Event) (bool, error)
}

// ScriptEvaluatorFunc 把函数适配为 ScriptEvaluator。
type ScriptEvaluatorFunc func(ctx context.Context, event Event) (bool, error)

// Evaluate 执行脚本判断函数。
func (f ScriptEvaluatorFunc) Evaluate(ctx context.Context, event Event) (bool, error) {
	if f == nil {
		return false, fmt.Errorf("goark-log: script evaluator func is nil")
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

// ScriptFilter 用调用方提供的执行器过滤日志事件。
type ScriptFilter struct {
	evaluator ScriptEvaluator
	onError   FilterDecision
	outcome   filterOutcome
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

// ScriptFilterOption 调整 ScriptFilter。
type ScriptFilterOption func(*scriptFilterSettings)

type scriptFilterSettings struct {
	onError FilterDecision
	outcome filterOutcome
}

// WithScriptFilterOnMatch 设置脚本匹配时的裁决。
func WithScriptFilterOnMatch(decision FilterDecision) ScriptFilterOption {
	return func(settings *scriptFilterSettings) {
		settings.outcome.onMatch = decision
	}
}

// WithScriptFilterOnMismatch 设置脚本不匹配时的裁决。
func WithScriptFilterOnMismatch(decision FilterDecision) ScriptFilterOption {
	return func(settings *scriptFilterSettings) {
		settings.outcome.onMismatch = decision
	}
}

// WithScriptFilterOnError 设置脚本执行失败时的裁决。
func WithScriptFilterOnError(decision FilterDecision) ScriptFilterOption {
	return func(settings *scriptFilterSettings) {
		settings.onError = decision
	}
}

// NewScriptFilter 创建脚本过滤器。
func NewScriptFilter(evaluator ScriptEvaluator, options ...ScriptFilterOption) (*ScriptFilter, error) {
	if evaluator == nil {
		return nil, fmt.Errorf("goark-log: script evaluator is nil")
	}
	settings := scriptFilterSettings{
		onError: FilterDeny,
		outcome: filterOutcome{
			onMatch:    FilterNeutral,
			onMismatch: FilterDeny,
		},
	}
	for _, option := range options {
		if option != nil {
			option(&settings)
		}
	}
	return &ScriptFilter{evaluator: evaluator, onError: settings.onError, outcome: settings.outcome}, nil
}

// Decide 执行脚本判断，脚本错误默认按拒绝处理。
func (f *ScriptFilter) Decide(ctx context.Context, event Event) FilterDecision {
	if f == nil || f.evaluator == nil {
		return FilterNeutral
	}
	matched, err := f.evaluator.Evaluate(ctx, event)
	if err != nil {
		return f.onError
	}
	return f.outcome.decide(matched)
}

func (o filterOutcome) decide(matched bool) FilterDecision {
	if matched {
		return o.onMatch
	}
	return o.onMismatch
}

func newSettings(onMatch FilterDecision, onMismatch FilterDecision, options ...FilterOption) *filterSettings {
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
	settings := newSettings(FilterNeutral, FilterDeny, options...)
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
	settings := newSettings(FilterNeutral, FilterDeny, options...)
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
	settings := newSettings(FilterNeutral, FilterDeny, options...)
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
		if configured, exists := f.thresholds[logvalue.String(value)]; exists {
			threshold = configured
		}
	}
	return f.outcome.decide(event.Level >= threshold)
}

// ParseFloat 解析配置中的浮点数。
func ParseFloat(value string, field string) (float64, error) {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0, fmt.Errorf("goark-log: %s is invalid", field)
	}
	return parsed, nil
}

// Normalize 校验并复制过滤器链。
func Normalize(scope string, filters []Filter) ([]Filter, error) {
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

// Append 合并过滤器链。
func Append(dst []Filter, src []Filter) []Filter {
	if len(src) == 0 {
		return dst
	}
	return append(dst, src...)
}

// Apply 按顺序执行过滤器链。
func Apply(ctx context.Context, filters []Filter, event Event) FilterDecision {
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
