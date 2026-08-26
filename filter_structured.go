package goarklog

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

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
