package filter

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"goark.dev/log/internal/logvalue"
)

// MapFilterOperator 指定 map 过滤器多条件关系。
type MapFilterOperator string

const (
	MapFilterAnd MapFilterOperator = "and"
	MapFilterOr  MapFilterOperator = "or"
)

// ParseMapFilterOperator 解析 map 过滤器条件关系。
func ParseMapFilterOperator(value string) (MapFilterOperator, error) {
	switch MapFilterOperator(strings.ToLower(strings.TrimSpace(value))) {
	case "", MapFilterAnd:
		return MapFilterAnd, nil
	case MapFilterOr:
		return MapFilterOr, nil
	default:
		return "", fmt.Errorf("goark-log: unsupported map filter operator %q", value)
	}
}

// MapFilter 按事件属性键值匹配。
type MapFilter struct {
	values   map[string]string
	operator MapFilterOperator
	outcome  filterOutcome
}

// MapFilterOption 调整 MapFilter。
type MapFilterOption func(*mapFilterSettings)

type mapFilterSettings struct {
	operator MapFilterOperator
	base     filterSettings
}

// WithMapFilterOperator 设置多个键值条件的组合关系。
func WithMapFilterOperator(operator MapFilterOperator) MapFilterOption {
	return func(settings *mapFilterSettings) {
		settings.operator = operator
	}
}

// WithMapFilterOnMatch 设置匹配时裁决。
func WithMapFilterOnMatch(decision FilterDecision) MapFilterOption {
	return func(settings *mapFilterSettings) {
		settings.base.outcome.onMatch = decision
	}
}

// WithMapFilterOnMismatch 设置不匹配时裁决。
func WithMapFilterOnMismatch(decision FilterDecision) MapFilterOption {
	return func(settings *mapFilterSettings) {
		settings.base.outcome.onMismatch = decision
	}
}

// NewMapFilter 创建属性 map 过滤器。
func NewMapFilter(values map[string]string, options ...MapFilterOption) (*MapFilter, error) {
	settings := mapFilterSettings{
		operator: MapFilterAnd,
		base:     *newSettings(FilterNeutral, FilterDeny),
	}
	for _, option := range options {
		if option != nil {
			option(&settings)
		}
	}
	if _, err := ParseMapFilterOperator(string(settings.operator)); err != nil {
		return nil, err
	}
	normalized := make(map[string]string, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("goark-log: map filter key is empty")
		}
		normalized[key] = value
	}
	if len(normalized) == 0 {
		return nil, fmt.Errorf("goark-log: map filter requires at least one value")
	}
	return &MapFilter{values: normalized, operator: settings.operator, outcome: settings.base.outcome}, nil
}

func (f *MapFilter) Decide(_ context.Context, event Event) FilterDecision {
	if f == nil {
		return FilterNeutral
	}
	matched := f.matches(event)
	return f.outcome.decide(matched)
}

func (f *MapFilter) matches(event Event) bool {
	if f.operator == MapFilterOr {
		for key, want := range f.values {
			value, ok := event.Attr(key)
			if ok && logvalue.String(value) == want {
				return true
			}
		}
		return false
	}
	for key, want := range f.values {
		value, ok := event.Attr(key)
		if !ok || logvalue.String(value) != want {
			return false
		}
	}
	return true
}

// ThreadContextMapFilter 是 MDC 场景下的 MapFilter 别名实现。
type ThreadContextMapFilter struct {
	*MapFilter
}

// NewThreadContextMapFilter 创建 MDC 键值过滤器。
func NewThreadContextMapFilter(values map[string]string, options ...MapFilterOption) (*ThreadContextMapFilter, error) {
	filter, err := NewMapFilter(values, options...)
	if err != nil {
		return nil, err
	}
	return &ThreadContextMapFilter{MapFilter: filter}, nil
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
		base:  *newSettings(FilterNeutral, FilterDeny),
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
