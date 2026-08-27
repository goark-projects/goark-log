package goarklog

import (
	"context"
	"fmt"
	"strings"
)

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
