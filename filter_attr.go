package goarklog

import (
	"context"
	"fmt"
	"strings"

	"goark.dev/log/internal/logvalue"
)

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
