package goarklog

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// AppenderRef 描述一次到 appender 的结构化引用。
type AppenderRef struct {
	Ref             string
	Level           *slog.Level
	IncludeLocation *bool
	Filters         []Filter
}

// AppenderRefOption 调整结构化 appender 引用。
type AppenderRefOption func(*AppenderRef)

// NewAppenderRef 创建结构化 appender 引用。
func NewAppenderRef(ref string, options ...AppenderRefOption) AppenderRef {
	config := AppenderRef{Ref: ref}
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}
	return config
}

// WithAppenderRefLevel 设置当前引用独有的级别下限。
func WithAppenderRefLevel(level slog.Level) AppenderRefOption {
	return func(config *AppenderRef) {
		copied := level
		config.Level = &copied
	}
}

// WithAppenderRefLocation 设置当前引用是否采集调用位置。
func WithAppenderRefLocation(enabled bool) AppenderRefOption {
	return func(config *AppenderRef) {
		copied := enabled
		config.IncludeLocation = &copied
	}
}

// WithAppenderRefFilters 设置当前引用独有的过滤器链。
func WithAppenderRefFilters(filters ...Filter) AppenderRefOption {
	return func(config *AppenderRef) {
		config.Filters = append(config.Filters, filters...)
	}
}

type appenderControl struct {
	ref             string
	level           *slog.Level
	includeLocation *bool
	filters         []Filter
	appender        Appender
}

type controlledAppender struct {
	control appenderControl
}

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

func newAppenderControl(appenderByName map[string]Appender, ref AppenderRef) (appenderControl, error) {
	name := strings.TrimSpace(ref.Ref)
	if name == "" {
		return appenderControl{}, fmt.Errorf("appender ref is empty")
	}
	appender, ok := appenderByName[name]
	if !ok {
		return appenderControl{}, fmt.Errorf("appender %q is not configured", name)
	}
	filters, err := normalizeFilters("appender ref "+name, ref.Filters)
	if err != nil {
		return appenderControl{}, err
	}
	control := appenderControl{
		ref:      name,
		filters:  filters,
		appender: appender,
	}
	if ref.Level != nil {
		level := *ref.Level
		control.level = &level
	}
	if ref.IncludeLocation != nil {
		includeLocation := *ref.IncludeLocation
		control.includeLocation = &includeLocation
	}
	return control, nil
}

func (c appenderControl) Append(ctx context.Context, event Event) error {
	_, err := c.append(ctx, event)
	return err
}

// append 返回底层 appender 是否被实际调用，供核心指标区分跳过和写入。
func (c appenderControl) append(ctx context.Context, event Event) (bool, error) {
	if c.appender == nil {
		return false, nil
	}
	if c.level != nil && event.Level < *c.level {
		return false, nil
	}
	if applyFilters(ctx, c.filters, event) == FilterDeny {
		return false, nil
	}
	if c.includeLocation != nil && !*c.includeLocation {
		event.PC = 0
	}
	return true, c.appender.Append(ctx, event)
}

func (c appenderControl) requiresLocation() bool {
	return c.includeLocation != nil && *c.includeLocation
}

func (c appenderControl) name() string {
	if c.appender == nil {
		return c.ref
	}
	return c.appender.Name()
}

func (a controlledAppender) Name() string {
	return a.control.name()
}

func (a controlledAppender) Append(ctx context.Context, event Event) error {
	_, err := a.control.append(ctx, event)
	return err
}

func (a controlledAppender) Close() error {
	if a.control.appender == nil {
		return nil
	}
	return a.control.appender.Close()
}

func resolveAppenderControls(appenderByName map[string]Appender, refs []AppenderRef) ([]appenderControl, error) {
	controls := make([]appenderControl, 0, len(refs))
	for _, ref := range refs {
		control, err := newAppenderControl(appenderByName, ref)
		if err != nil {
			return nil, err
		}
		controls = append(controls, control)
	}
	return controls, nil
}

func appendUniqueAppenderControls(dst []appenderControl, src []appenderControl) []appenderControl {
	seen := make(map[string]struct{}, len(dst)+len(src))
	out := dst[:0]
	for _, control := range dst {
		name := control.name()
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, control)
	}
	for _, control := range src {
		name := control.name()
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, control)
	}
	return out
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

func mergeAppenderRefs(simple []string, controls []AppenderRef) []AppenderRef {
	if len(simple) == 0 && len(controls) == 0 {
		return nil
	}
	refs := make([]AppenderRef, 0, len(simple)+len(controls))
	for _, ref := range simple {
		refs = append(refs, AppenderRef{Ref: ref})
	}
	for _, ref := range controls {
		refs = append(refs, copyAppenderRef(ref))
	}
	return refs
}

func copyAppenderRef(ref AppenderRef) AppenderRef {
	copied := AppenderRef{
		Ref:     ref.Ref,
		Filters: append([]Filter(nil), ref.Filters...),
	}
	if ref.Level != nil {
		level := *ref.Level
		copied.Level = &level
	}
	if ref.IncludeLocation != nil {
		includeLocation := *ref.IncludeLocation
		copied.IncludeLocation = &includeLocation
	}
	return copied
}
