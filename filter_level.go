package goarklog

import (
	"context"
	"fmt"
	"log/slog"
)

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
