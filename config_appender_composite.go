package goarklog

import (
	"fmt"
	"strings"
)

func buildCompositeAppender(name string, spec appenderConfig, specs map[string]appenderConfig, built map[string]Appender, filters map[string]Filter, registry *PluginRegistry) (Appender, bool, error) {
	switch normalizeKind(spec.Type) {
	case "async":
		appender, waiting, err := buildAsyncAppender(name, spec, specs, built, filters, registry)
		return appender, waiting, err
	case "failover", "failoverappender":
		return buildFailoverAppender(name, spec, specs, built, registry)
	case "routing", "routingappender":
		return buildRoutingAppender(name, spec, specs, built, registry)
	case "rewrite", "rewriteappender":
		return buildRewriteAppender(name, spec, specs, built, registry)
	default:
		return nil, false, fmt.Errorf("goark-log: unsupported composite appender %q type %q", name, spec.Type)
	}
}

func buildAsyncAppender(name string, spec appenderConfig, specs map[string]appenderConfig, built map[string]Appender, filters map[string]Filter, registry *PluginRegistry) (Appender, bool, error) {
	refs := spec.refs()
	controls, err := spec.appenderRefControls(filters)
	if err != nil {
		return nil, false, fmt.Errorf("goark-log: async appender %q: %w", name, err)
	}
	if len(refs) == 0 && len(controls) == 0 {
		return nil, false, fmt.Errorf("goark-log: async appender %q requires appenderRefs", name)
	}
	delegates := make([]Appender, 0, len(refs)+len(controls))
	for _, ref := range refs {
		appender, waiting, err := resolveCompositeAppenderRef("async appender", name, ref, specs, built)
		if err != nil || waiting {
			return nil, waiting, err
		}
		delegates = append(delegates, appender)
	}
	for _, ref := range controls {
		if _, waiting, err := resolveCompositeAppenderRef("async appender", name, ref.Ref, specs, built); err != nil || waiting {
			return nil, waiting, err
		}
		control, err := newAppenderControl(built, ref)
		if err != nil {
			return nil, false, fmt.Errorf("goark-log: async appender %q: %w", name, err)
		}
		delegates = append(delegates, controlledAppender{control: control})
	}
	factory, ok := registry.appenderFactory(spec.Type)
	if !ok {
		return nil, false, fmt.Errorf("goark-log: unsupported appender %q type %q", name, spec.Type)
	}
	appender, err := factory(spec.appenderBuildConfig(name, nil, delegates, nil, nil))
	if err != nil {
		return nil, false, err
	}
	wrapped, err := wrapAppenderFilters(name, appender, spec.filterRefs(), filters)
	if err != nil {
		_ = appender.Close()
		return nil, false, err
	}
	return wrapped, false, nil
}

func buildFailoverAppender(name string, spec appenderConfig, specs map[string]appenderConfig, built map[string]Appender, registry *PluginRegistry) (Appender, bool, error) {
	refs := spec.failoverRefs()
	if len(refs) < 2 {
		return nil, false, fmt.Errorf("goark-log: failover appender %q requires primary and failovers", name)
	}
	delegates, waiting, err := resolveCompositeAppenderRefs("failover appender", name, refs, specs, built)
	if err != nil || waiting {
		return nil, waiting, err
	}
	factory, ok := registry.appenderFactory(spec.Type)
	if !ok {
		return nil, false, fmt.Errorf("goark-log: unsupported appender %q type %q", name, spec.Type)
	}
	appender, err := factory(spec.appenderBuildConfig(name, nil, delegates, nil, nil))
	return appender, false, err
}

func buildRoutingAppender(name string, spec appenderConfig, specs map[string]appenderConfig, built map[string]Appender, registry *PluginRegistry) (Appender, bool, error) {
	routes, waiting, err := resolveRoutingRoutes("routing appender", name, spec.routes(), specs, built)
	if err != nil || waiting {
		return nil, waiting, err
	}
	defaultRoute, waiting, err := resolveOptionalCompositeAppenderRef("routing appender", name, spec.defaultRoute(), specs, built)
	if err != nil || waiting {
		return nil, waiting, err
	}
	factory, ok := registry.appenderFactory(spec.Type)
	if !ok {
		return nil, false, fmt.Errorf("goark-log: unsupported appender %q type %q", name, spec.Type)
	}
	appender, err := factory(spec.appenderBuildConfig(name, nil, nil, routes, defaultRoute))
	return appender, false, err
}

func buildRewriteAppender(name string, spec appenderConfig, specs map[string]appenderConfig, built map[string]Appender, registry *PluginRegistry) (Appender, bool, error) {
	refs := spec.refs()
	if len(refs) != 1 {
		return nil, false, fmt.Errorf("goark-log: rewrite appender %q requires exactly one appenderRef", name)
	}
	delegates, waiting, err := resolveCompositeAppenderRefs("rewrite appender", name, refs, specs, built)
	if err != nil || waiting {
		return nil, waiting, err
	}
	factory, ok := registry.appenderFactory(spec.Type)
	if !ok {
		return nil, false, fmt.Errorf("goark-log: unsupported appender %q type %q", name, spec.Type)
	}
	appender, err := factory(spec.appenderBuildConfig(name, nil, delegates, nil, nil))
	return appender, false, err
}

func resolveCompositeAppenderRefs(ownerKind string, owner string, refs []string, specs map[string]appenderConfig, built map[string]Appender) ([]Appender, bool, error) {
	appenders := make([]Appender, 0, len(refs))
	for _, ref := range refs {
		appender, waiting, err := resolveCompositeAppenderRef(ownerKind, owner, ref, specs, built)
		if err != nil || waiting {
			return nil, waiting, err
		}
		appenders = append(appenders, appender)
	}
	return appenders, false, nil
}

func resolveOptionalCompositeAppenderRef(ownerKind string, owner string, ref string, specs map[string]appenderConfig, built map[string]Appender) (Appender, bool, error) {
	if strings.TrimSpace(ref) == "" {
		return nil, false, nil
	}
	return resolveCompositeAppenderRef(ownerKind, owner, ref, specs, built)
}

func resolveCompositeAppenderRef(ownerKind string, owner string, ref string, specs map[string]appenderConfig, built map[string]Appender) (Appender, bool, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, false, fmt.Errorf("goark-log: %s %q appender ref is empty", ownerKind, owner)
	}
	if appender, ok := built[ref]; ok {
		return appender, false, nil
	}
	if _, ok := specs[ref]; ok {
		return nil, true, nil
	}
	return nil, false, fmt.Errorf("goark-log: %s %q references unknown appender %q", ownerKind, owner, ref)
}

func resolveRoutingRoutes(ownerKind string, owner string, routeRefs map[string]string, specs map[string]appenderConfig, built map[string]Appender) (map[string]Appender, bool, error) {
	if len(routeRefs) == 0 {
		return nil, false, nil
	}
	routes := make(map[string]Appender, len(routeRefs))
	for key, ref := range routeRefs {
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, false, fmt.Errorf("goark-log: %s %q route key is empty", ownerKind, owner)
		}
		appender, waiting, err := resolveCompositeAppenderRef(ownerKind, owner, ref, specs, built)
		if err != nil || waiting {
			return nil, waiting, err
		}
		routes[key] = appender
	}
	return routes, false, nil
}

func isCompositeAppenderKind(value string) bool {
	switch normalizeKind(value) {
	case "async", "failover", "failoverappender", "routing", "routingappender", "rewrite", "rewriteappender":
		return true
	default:
		return false
	}
}
