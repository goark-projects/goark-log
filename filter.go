package goarklog

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
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

// RegexFilterField 指定 RegexFilter 的匹配目标。
type RegexFilterField string

const (
	RegexFieldMessage RegexFilterField = "message"
	RegexFieldLogger  RegexFilterField = "logger"
	RegexFieldAttr    RegexFilterField = "attr"
)

// RegexFilterOption 调整正则过滤器。
type RegexFilterOption func(*regexFilterSettings)

type regexFilterSettings struct {
	field RegexFilterField
	key   string
	base  filterSettings
}

// WithRegexField 设置正则匹配字段。
func WithRegexField(field RegexFilterField) RegexFilterOption {
	return func(settings *regexFilterSettings) {
		settings.field = field
	}
}

// WithRegexAttrKey 设置正则匹配的属性键。
func WithRegexAttrKey(key string) RegexFilterOption {
	return func(settings *regexFilterSettings) {
		settings.key = key
	}
}

// WithRegexOnMatch 设置正则匹配时的裁决。
func WithRegexOnMatch(decision FilterDecision) RegexFilterOption {
	return func(settings *regexFilterSettings) {
		settings.base.outcome.onMatch = decision
	}
}

// WithRegexOnMismatch 设置正则不匹配时的裁决。
func WithRegexOnMismatch(decision FilterDecision) RegexFilterOption {
	return func(settings *regexFilterSettings) {
		settings.base.outcome.onMismatch = decision
	}
}

// RegexFilter 按消息、logger 名称或属性值执行正则匹配。
type RegexFilter struct {
	field   RegexFilterField
	key     string
	pattern *regexp.Regexp
	outcome filterOutcome
}

// NewRegexFilter 创建正则过滤器。
func NewRegexFilter(pattern string, options ...RegexFilterOption) (*RegexFilter, error) {
	if strings.TrimSpace(pattern) == "" {
		return nil, fmt.Errorf("goark-log: regex filter pattern is empty")
	}
	settings := regexFilterSettings{
		field: RegexFieldMessage,
		base:  *newFilterSettings(FilterNeutral, FilterDeny),
	}
	for _, option := range options {
		if option != nil {
			option(&settings)
		}
	}
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("goark-log: compile regex filter pattern %q: %w", pattern, err)
	}
	filter := &RegexFilter{
		field:   settings.field,
		key:     strings.TrimSpace(settings.key),
		pattern: compiled,
		outcome: settings.base.outcome,
	}
	if err := filter.validate(); err != nil {
		return nil, err
	}
	return filter, nil
}

func (f *RegexFilter) Decide(_ context.Context, event Event) FilterDecision {
	if f == nil || f.pattern == nil {
		return FilterNeutral
	}
	value, ok := f.value(event)
	return f.outcome.decide(ok && f.pattern.MatchString(value))
}

func (f *RegexFilter) validate() error {
	switch f.field {
	case RegexFieldMessage, RegexFieldLogger:
		return nil
	case RegexFieldAttr:
		if f.key == "" {
			return fmt.Errorf("goark-log: regex attr filter key is empty")
		}
		return nil
	default:
		return fmt.Errorf("goark-log: unsupported regex filter field %q", f.field)
	}
}

func (f *RegexFilter) value(event Event) (string, bool) {
	switch f.field {
	case RegexFieldMessage:
		return event.Message, true
	case RegexFieldLogger:
		return event.Logger, true
	case RegexFieldAttr:
		value, ok := event.Attr(f.key)
		if !ok {
			return "", false
		}
		return attrValueString(value), true
	default:
		return "", false
	}
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
	return f.outcome.decide(ok && attrValueString(value) == f.value)
}

// FilteredAppender 为任意 appender 增加过滤器链。
type FilteredAppender struct {
	delegate Appender
	filters  []Filter
}

// NewFilteredAppender 创建带过滤器链的 appender。
func NewFilteredAppender(delegate Appender, filters ...Filter) (*FilteredAppender, error) {
	if delegate == nil {
		return nil, fmt.Errorf("goark-log: filtered appender delegate is nil")
	}
	chain, err := normalizeFilters("appender "+delegate.Name(), filters)
	if err != nil {
		return nil, err
	}
	return &FilteredAppender{delegate: delegate, filters: chain}, nil
}

func (a *FilteredAppender) Name() string {
	if a == nil || a.delegate == nil {
		return ""
	}
	return a.delegate.Name()
}

func (a *FilteredAppender) Append(ctx context.Context, event Event) error {
	if a == nil || a.delegate == nil {
		return fmt.Errorf("goark-log: filtered appender is nil")
	}
	if applyFilters(ctx, a.filters, event) == FilterDeny {
		return nil
	}
	return a.delegate.Append(ctx, event)
}

func (a *FilteredAppender) Close() error {
	if a == nil || a.delegate == nil {
		return nil
	}
	return a.delegate.Close()
}

func isAsyncAppender(appender Appender) bool {
	switch value := appender.(type) {
	case *AsyncAppender:
		return true
	case *FilteredAppender:
		return isAsyncAppender(value.delegate)
	default:
		return false
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
