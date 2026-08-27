package goarklog

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"goark.dev/log/internal/logvalue"
)

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
		return logvalue.String(value), true
	default:
		return "", false
	}
}
