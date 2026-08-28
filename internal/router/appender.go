package router

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	logfilter "goark.dev/log/internal/filter"
	"goark.dev/log/internal/logevent"
)

// Event 是路由层处理的日志事件快照。
type Event = logevent.Event

// Filter 是日志事件过滤器。实现必须并发安全。
type Filter = logfilter.Filter

// Appender 是日志事件的最终写出端。
type Appender interface {
	Name() string
	Append(ctx context.Context, event Event) error
	Close() error
}

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

// AppenderControl 是一次 appender 引用解析后的运行期控制器。
type AppenderControl struct {
	ref             string
	level           *slog.Level
	includeLocation *bool
	filters         []Filter
	appender        Appender
}

// ControlledAppender 把结构化引用控制适配成普通 appender。
type ControlledAppender struct {
	control AppenderControl
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
	chain, err := logfilter.Normalize("appender "+delegate.Name(), filters)
	if err != nil {
		return nil, err
	}
	return &FilteredAppender{delegate: delegate, filters: chain}, nil
}

// NewAppenderControl 解析结构化 appender 引用。
func NewAppenderControl(appenderByName map[string]Appender, ref AppenderRef) (AppenderControl, error) {
	name := strings.TrimSpace(ref.Ref)
	if name == "" {
		return AppenderControl{}, fmt.Errorf("appender ref is empty")
	}
	appender, ok := appenderByName[name]
	if !ok {
		return AppenderControl{}, fmt.Errorf("appender %q is not configured", name)
	}
	filters, err := logfilter.Normalize("appender ref "+name, ref.Filters)
	if err != nil {
		return AppenderControl{}, err
	}
	control := AppenderControl{
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

// NewControlledAppender 把引用控制器封装成 appender。
func NewControlledAppender(control AppenderControl) ControlledAppender {
	return ControlledAppender{control: control}
}

func (c AppenderControl) Append(ctx context.Context, event Event) error {
	_, err := c.AppendResult(ctx, event)
	return err
}

// AppendResult 返回底层 appender 是否被实际调用，供核心指标区分跳过和写入。
func (c AppenderControl) AppendResult(ctx context.Context, event Event) (bool, error) {
	if c.appender == nil {
		return false, nil
	}
	if c.level != nil && event.Level < *c.level {
		return false, nil
	}
	if logfilter.Apply(ctx, c.filters, event) == logfilter.FilterDeny {
		return false, nil
	}
	if c.includeLocation != nil && !*c.includeLocation {
		event.PC = 0
	}
	return true, c.appender.Append(ctx, event)
}

func (c AppenderControl) requiresLocation() bool {
	return c.includeLocation != nil && *c.includeLocation
}

func (c AppenderControl) name() string {
	if c.appender == nil {
		return c.ref
	}
	return c.appender.Name()
}

func (a ControlledAppender) Name() string {
	return a.control.name()
}

func (a ControlledAppender) Append(ctx context.Context, event Event) error {
	_, err := a.control.AppendResult(ctx, event)
	return err
}

func (a ControlledAppender) Close() error {
	if a.control.appender == nil {
		return nil
	}
	return a.control.appender.Close()
}

// Delegate 返回被过滤器包装的下游 appender。
func (a *FilteredAppender) Delegate() Appender {
	if a == nil {
		return nil
	}
	return a.delegate
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
	ctx = logevent.NormalizeContext(ctx)
	if logfilter.Apply(ctx, a.filters, event) == logfilter.FilterDeny {
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

func resolveAppenderControls(appenderByName map[string]Appender, refs []AppenderRef) ([]AppenderControl, error) {
	controls := make([]AppenderControl, 0, len(refs))
	for _, ref := range refs {
		control, err := NewAppenderControl(appenderByName, ref)
		if err != nil {
			return nil, err
		}
		controls = append(controls, control)
	}
	return controls, nil
}

func appendUniqueAppenderControls(dst []AppenderControl, src []AppenderControl) []AppenderControl {
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
