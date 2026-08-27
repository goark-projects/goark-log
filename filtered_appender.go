package goarklog

import (
	"context"
	"fmt"
)

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
	ctx = normalizeContext(ctx)
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
