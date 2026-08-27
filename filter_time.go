package goarklog

import (
	"context"
	"fmt"
	"strings"
	"time"
)

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
