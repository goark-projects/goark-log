package plugin

import (
	"context"
	"log/slog"
	"sort"
	"strings"

	"goark.dev/log/internal/delegating"
	"goark.dev/log/internal/logevent"
)

func newAttributeRewritePolicy(config RewriteBuildConfig) delegating.RewritePolicy {
	additions := rewriteAdditions(config.Attrs)
	removals := rewriteRemovalSet(config.RemoveAttrs)
	if len(additions) == 0 && len(removals) == 0 {
		return nil
	}
	return func(_ context.Context, event logevent.Event) (logevent.Event, error) {
		rewritten := event
		attrs := make([]slog.Attr, 0, len(event.Attrs)+len(additions))
		for _, attr := range event.Attrs {
			if _, remove := removals[attr.Key]; remove {
				continue
			}
			attrs = append(attrs, attr)
		}
		attrs = append(attrs, additions...)
		rewritten.Attrs = attrs
		return rewritten, nil
	}
}

func rewriteAdditions(values map[string]string) []slog.Attr {
	if len(values) == 0 {
		return nil
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		key = strings.TrimSpace(key)
		if key != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	attrs := make([]slog.Attr, 0, len(keys))
	for _, key := range keys {
		attrs = append(attrs, slog.String(key, values[key]))
	}
	return attrs
}

func rewriteRemovalSet(keys []string) map[string]struct{} {
	if len(keys) == 0 {
		return nil
	}
	removals := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key != "" {
			removals[key] = struct{}{}
		}
	}
	return removals
}
