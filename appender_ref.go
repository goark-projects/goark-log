package goarklog

import "log/slog"

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
