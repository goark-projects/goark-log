package goarklog

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"goark.dev/log/internal/logvalue"
)

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
		if configured, exists := f.thresholds[logvalue.String(value)]; exists {
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
