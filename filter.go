package goarklog

import (
	"context"
	"log/slog"
	"time"

	logfilter "goark.dev/log/internal/filter"
)

// FilterDecision 表示过滤器对日志事件的裁决。
type FilterDecision = logfilter.FilterDecision

const (
	FilterNeutral = logfilter.FilterNeutral
	FilterAccept  = logfilter.FilterAccept
	FilterDeny    = logfilter.FilterDeny
)

// Filter 是日志事件过滤器。实现必须并发安全。
type Filter = logfilter.Filter

// FilterFunc 把普通函数适配为 Filter。
type FilterFunc = logfilter.FilterFunc

// ScriptEvaluator 是脚本过滤器的执行器契约。
type ScriptEvaluator = logfilter.ScriptEvaluator

// ScriptEvaluatorFunc 把函数适配为 ScriptEvaluator。
type ScriptEvaluatorFunc = logfilter.ScriptEvaluatorFunc

// ScriptFilter 用调用方提供的执行器过滤日志事件。
type ScriptFilter = logfilter.ScriptFilter

// FilterOption 调整内置过滤器的匹配结果。
type FilterOption = logfilter.FilterOption

// ScriptFilterOption 调整 ScriptFilter。
type ScriptFilterOption = logfilter.ScriptFilterOption

// TimeFilter 按一天内的时间区间匹配事件。
type TimeFilter = logfilter.TimeFilter

// BurstFilter 对低优先级日志做令牌桶限流。
type BurstFilter = logfilter.BurstFilter

// DynamicThresholdFilter 按事件属性动态选择级别阈值。
type DynamicThresholdFilter = logfilter.DynamicThresholdFilter

// DenyFilter 无条件拒绝事件。
type DenyFilter = logfilter.DenyFilter

// CompositeFilter 按顺序组合多个过滤器。
type CompositeFilter = logfilter.CompositeFilter

// AttrFilter 按属性键和值过滤日志事件。
type AttrFilter = logfilter.AttrFilter

// StringMatchFilter 按消息子串匹配事件。
type StringMatchFilter = logfilter.StringMatchFilter

// ThresholdFilter 按日志级别下限过滤。
type ThresholdFilter = logfilter.ThresholdFilter

// LevelFilter 按单个日志级别过滤。
type LevelFilter = logfilter.LevelFilter

// LevelRangeFilter 按日志级别闭区间过滤。
type LevelRangeFilter = logfilter.LevelRangeFilter

// MarkerFilter 按 marker 名称或父级 marker 匹配事件。
type MarkerFilter = logfilter.MarkerFilter

// NoMarkerFilter 匹配没有 marker 的事件。
type NoMarkerFilter = logfilter.NoMarkerFilter

// ThreadContextStackFilter 按 context stack/NDC 值匹配事件。
type ThreadContextStackFilter = logfilter.ThreadContextStackFilter

// ThrowableFilter 按异常文本匹配事件。
type ThrowableFilter = logfilter.ThrowableFilter

// StructuredDataFilter 是面向结构化属性的 MapFilter 别名。
type StructuredDataFilter = logfilter.StructuredDataFilter

// MapFilterOperator 指定 map 过滤器多条件关系。
type MapFilterOperator = logfilter.MapFilterOperator

const (
	MapFilterAnd = logfilter.MapFilterAnd
	MapFilterOr  = logfilter.MapFilterOr
)

// MapFilter 按事件属性键值匹配。
type MapFilter = logfilter.MapFilter

// MapFilterOption 调整 MapFilter。
type MapFilterOption = logfilter.MapFilterOption

// ThreadContextMapFilter 是 MDC 场景下的 MapFilter 别名实现。
type ThreadContextMapFilter = logfilter.ThreadContextMapFilter

// RegexFilterField 指定 RegexFilter 的匹配目标。
type RegexFilterField = logfilter.RegexFilterField

const (
	RegexFieldMessage = logfilter.RegexFieldMessage
	RegexFieldLogger  = logfilter.RegexFieldLogger
	RegexFieldAttr    = logfilter.RegexFieldAttr
)

// RegexFilterOption 调整正则过滤器。
type RegexFilterOption = logfilter.RegexFilterOption

// RegexFilter 按消息、logger 名称或属性值执行正则匹配。
type RegexFilter = logfilter.RegexFilter

// ParseFilterDecision 解析过滤器裁决名称。
func ParseFilterDecision(value string) (FilterDecision, error) {
	return logfilter.ParseFilterDecision(value)
}

// WithFilterOnMatch 设置匹配时的裁决。
func WithFilterOnMatch(decision FilterDecision) FilterOption {
	return logfilter.WithFilterOnMatch(decision)
}

// WithFilterOnMismatch 设置不匹配时的裁决。
func WithFilterOnMismatch(decision FilterDecision) FilterOption {
	return logfilter.WithFilterOnMismatch(decision)
}

// WithScriptFilterOnMatch 设置脚本匹配时的裁决。
func WithScriptFilterOnMatch(decision FilterDecision) ScriptFilterOption {
	return logfilter.WithScriptFilterOnMatch(decision)
}

// WithScriptFilterOnMismatch 设置脚本不匹配时的裁决。
func WithScriptFilterOnMismatch(decision FilterDecision) ScriptFilterOption {
	return logfilter.WithScriptFilterOnMismatch(decision)
}

// WithScriptFilterOnError 设置脚本执行失败时的裁决。
func WithScriptFilterOnError(decision FilterDecision) ScriptFilterOption {
	return logfilter.WithScriptFilterOnError(decision)
}

// NewScriptFilter 创建脚本过滤器。
func NewScriptFilter(evaluator ScriptEvaluator, options ...ScriptFilterOption) (*ScriptFilter, error) {
	return logfilter.NewScriptFilter(evaluator, options...)
}

// NewTimeFilter 创建时间区间过滤器。
func NewTimeFilter(start string, end string, options ...FilterOption) (*TimeFilter, error) {
	return logfilter.NewTimeFilter(start, end, options...)
}

// NewTimeFilterInLocation 创建固定时区的时间区间过滤器。
func NewTimeFilterInLocation(start string, end string, location *time.Location, options ...FilterOption) (*TimeFilter, error) {
	return logfilter.NewTimeFilterInLocation(start, end, location, options...)
}

// NewBurstFilter 创建突发限流过滤器。
func NewBurstFilter(level slog.Level, ratePerSecond float64, maxBurst int, options ...FilterOption) (*BurstFilter, error) {
	return logfilter.NewBurstFilter(level, ratePerSecond, maxBurst, options...)
}

// NewDynamicThresholdFilter 创建动态级别阈值过滤器。
func NewDynamicThresholdFilter(key string, defaultThreshold slog.Level, thresholds map[string]slog.Level, options ...FilterOption) (*DynamicThresholdFilter, error) {
	return logfilter.NewDynamicThresholdFilter(key, defaultThreshold, thresholds, options...)
}

// NewDenyFilter 创建无条件拒绝过滤器。
func NewDenyFilter() *DenyFilter {
	return logfilter.NewDenyFilter()
}

// NewCompositeFilter 创建组合过滤器。
func NewCompositeFilter(filters ...Filter) (*CompositeFilter, error) {
	return logfilter.NewCompositeFilter(filters...)
}

// NewAttrFilter 创建属性过滤器。
func NewAttrFilter(key string, value string, options ...FilterOption) (*AttrFilter, error) {
	return logfilter.NewAttrFilter(key, value, options...)
}

// NewStringMatchFilter 创建消息子串过滤器。
func NewStringMatchFilter(text string, options ...FilterOption) (*StringMatchFilter, error) {
	return logfilter.NewStringMatchFilter(text, options...)
}

// NewThresholdFilter 创建按级别下限过滤的过滤器。
func NewThresholdFilter(level slog.Level, options ...FilterOption) *ThresholdFilter {
	return logfilter.NewThresholdFilter(level, options...)
}

// NewLevelFilter 创建按单个级别匹配的过滤器。
func NewLevelFilter(level slog.Level, options ...FilterOption) *LevelFilter {
	return logfilter.NewLevelFilter(level, options...)
}

// NewLevelRangeFilter 创建级别区间过滤器。
func NewLevelRangeFilter(min slog.Level, max slog.Level, options ...FilterOption) (*LevelRangeFilter, error) {
	return logfilter.NewLevelRangeFilter(min, max, options...)
}

// NewMarkerFilter 创建 marker 过滤器。
func NewMarkerFilter(name string, options ...FilterOption) (*MarkerFilter, error) {
	return logfilter.NewMarkerFilter(name, options...)
}

// NewNoMarkerFilter 创建无 marker 过滤器。
func NewNoMarkerFilter(options ...FilterOption) *NoMarkerFilter {
	return logfilter.NewNoMarkerFilter(options...)
}

// NewThreadContextStackFilter 创建 context stack 过滤器。
func NewThreadContextStackFilter(value string, options ...FilterOption) (*ThreadContextStackFilter, error) {
	return logfilter.NewThreadContextStackFilter(value, options...)
}

// NewThrowableFilter 创建异常文本过滤器。
func NewThrowableFilter(pattern string, options ...FilterOption) (*ThrowableFilter, error) {
	return logfilter.NewThrowableFilter(pattern, options...)
}

// NewStructuredDataFilter 创建结构化属性过滤器。
func NewStructuredDataFilter(values map[string]string, options ...MapFilterOption) (*StructuredDataFilter, error) {
	return logfilter.NewStructuredDataFilter(values, options...)
}

// ParseMapFilterOperator 解析 map 过滤器条件关系。
func ParseMapFilterOperator(value string) (MapFilterOperator, error) {
	return logfilter.ParseMapFilterOperator(value)
}

// WithMapFilterOperator 设置多个键值条件的组合关系。
func WithMapFilterOperator(operator MapFilterOperator) MapFilterOption {
	return logfilter.WithMapFilterOperator(operator)
}

// WithMapFilterOnMatch 设置匹配时裁决。
func WithMapFilterOnMatch(decision FilterDecision) MapFilterOption {
	return logfilter.WithMapFilterOnMatch(decision)
}

// WithMapFilterOnMismatch 设置不匹配时裁决。
func WithMapFilterOnMismatch(decision FilterDecision) MapFilterOption {
	return logfilter.WithMapFilterOnMismatch(decision)
}

// NewMapFilter 创建属性 map 过滤器。
func NewMapFilter(values map[string]string, options ...MapFilterOption) (*MapFilter, error) {
	return logfilter.NewMapFilter(values, options...)
}

// NewThreadContextMapFilter 创建 MDC 键值过滤器。
func NewThreadContextMapFilter(values map[string]string, options ...MapFilterOption) (*ThreadContextMapFilter, error) {
	return logfilter.NewThreadContextMapFilter(values, options...)
}

// WithRegexField 设置正则匹配字段。
func WithRegexField(field RegexFilterField) RegexFilterOption {
	return logfilter.WithRegexField(field)
}

// WithRegexAttrKey 设置正则匹配的属性键。
func WithRegexAttrKey(key string) RegexFilterOption {
	return logfilter.WithRegexAttrKey(key)
}

// WithRegexOnMatch 设置正则匹配时的裁决。
func WithRegexOnMatch(decision FilterDecision) RegexFilterOption {
	return logfilter.WithRegexOnMatch(decision)
}

// WithRegexOnMismatch 设置正则不匹配时的裁决。
func WithRegexOnMismatch(decision FilterDecision) RegexFilterOption {
	return logfilter.WithRegexOnMismatch(decision)
}

// NewRegexFilter 创建正则过滤器。
func NewRegexFilter(pattern string, options ...RegexFilterOption) (*RegexFilter, error) {
	return logfilter.NewRegexFilter(pattern, options...)
}

func parseFloat(value string, field string) (float64, error) {
	return logfilter.ParseFloat(value, field)
}

func normalizeFilters(scope string, filters []Filter) ([]Filter, error) {
	return logfilter.Normalize(scope, filters)
}

func appendFilters(dst []Filter, src []Filter) []Filter {
	return logfilter.Append(dst, src)
}

func applyFilters(ctx context.Context, filters []Filter, event Event) FilterDecision {
	return logfilter.Apply(ctx, filters, event)
}
