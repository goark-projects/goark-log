package goarklog

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

type appenderControl struct {
	ref      string
	level    *slog.Level
	filters  []Filter
	appender Appender
}

type controlledAppender struct {
	control appenderControl
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
	return control, nil
}

func (c appenderControl) Append(ctx context.Context, event Event) error {
	if c.appender == nil {
		return nil
	}
	if c.level != nil && event.Level < *c.level {
		return nil
	}
	if applyFilters(ctx, c.filters, event) == FilterDeny {
		return nil
	}
	return c.appender.Append(ctx, event)
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
	return a.control.Append(ctx, event)
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
