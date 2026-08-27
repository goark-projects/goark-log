package goarklog

import (
	"fmt"
)

func (c *appenderConfig) resolveLookups(lookups *LookupResolver) error {
	var err error
	if c.Type, err = resolveStringLookup(lookups, c.Type); err != nil {
		return fmt.Errorf("type: %w", err)
	}
	if c.Target, err = resolveStringLookup(lookups, c.Target); err != nil {
		return fmt.Errorf("target: %w", err)
	}
	if c.URL, err = resolveStringLookup(lookups, c.URL); err != nil {
		return fmt.Errorf("url: %w", err)
	}
	if c.Method, err = resolveStringLookup(lookups, c.Method); err != nil {
		return fmt.Errorf("method: %w", err)
	}
	if c.Address, err = resolveStringLookup(lookups, c.Address); err != nil {
		return fmt.Errorf("address: %w", err)
	}
	if c.Network, err = resolveStringLookup(lookups, c.Network); err != nil {
		return fmt.Errorf("network: %w", err)
	}
	if c.Facility, err = resolveStringLookup(lookups, c.Facility); err != nil {
		return fmt.Errorf("facility: %w", err)
	}
	if c.AppName, err = resolveStringLookup(lookups, c.AppName); err != nil {
		return fmt.Errorf("appName: %w", err)
	}
	if c.AppNameKebab, err = resolveStringLookup(lookups, c.AppNameKebab); err != nil {
		return fmt.Errorf("app-name: %w", err)
	}
	if c.ConnectTimeout, err = resolveStringLookup(lookups, c.ConnectTimeout); err != nil {
		return fmt.Errorf("connectTimeout: %w", err)
	}
	if c.ConnectTimeoutKebab, err = resolveStringLookup(lookups, c.ConnectTimeoutKebab); err != nil {
		return fmt.Errorf("connect-timeout: %w", err)
	}
	if c.WriteTimeout, err = resolveStringLookup(lookups, c.WriteTimeout); err != nil {
		return fmt.Errorf("writeTimeout: %w", err)
	}
	if c.WriteTimeoutKebab, err = resolveStringLookup(lookups, c.WriteTimeoutKebab); err != nil {
		return fmt.Errorf("write-timeout: %w", err)
	}
	if c.FileName, err = resolveStringLookup(lookups, c.FileName); err != nil {
		return fmt.Errorf("fileName: %w", err)
	}
	if c.FileNameKebab, err = resolveStringLookup(lookups, c.FileNameKebab); err != nil {
		return fmt.Errorf("file-name: %w", err)
	}
	if c.Path, err = resolveStringLookup(lookups, c.Path); err != nil {
		return fmt.Errorf("path: %w", err)
	}
	if err := c.Layout.resolveLookups(lookups); err != nil {
		return fmt.Errorf("layout: %w", err)
	}
	if err := c.Rolling.resolveLookups(lookups); err != nil {
		return fmt.Errorf("rolling: %w", err)
	}
	if c.BufferSize, err = resolveStringLookup(lookups, c.BufferSize); err != nil {
		return fmt.Errorf("bufferSize: %w", err)
	}
	if c.BufferSizeKebab, err = resolveStringLookup(lookups, c.BufferSizeKebab); err != nil {
		return fmt.Errorf("buffer-size: %w", err)
	}
	if c.FilePermissions, err = resolveStringLookup(lookups, c.FilePermissions); err != nil {
		return fmt.Errorf("filePermissions: %w", err)
	}
	if c.FilePermissionsKebab, err = resolveStringLookup(lookups, c.FilePermissionsKebab); err != nil {
		return fmt.Errorf("file-permissions: %w", err)
	}
	if c.WaitStrategy, err = resolveStringLookup(lookups, c.WaitStrategy); err != nil {
		return fmt.Errorf("waitStrategy: %w", err)
	}
	if c.WaitStrategyKebab, err = resolveStringLookup(lookups, c.WaitStrategyKebab); err != nil {
		return fmt.Errorf("wait-strategy: %w", err)
	}
	if c.SleepTime, err = resolveStringLookup(lookups, c.SleepTime); err != nil {
		return fmt.Errorf("sleepTime: %w", err)
	}
	if c.SleepTimeKebab, err = resolveStringLookup(lookups, c.SleepTimeKebab); err != nil {
		return fmt.Errorf("sleep-time: %w", err)
	}
	if c.Timeout, err = resolveStringLookup(lookups, c.Timeout); err != nil {
		return fmt.Errorf("timeout: %w", err)
	}
	if c.AppenderRefs, err = c.AppenderRefs.resolveLookups(lookups); err != nil {
		return fmt.Errorf("appenderRefs: %w", err)
	}
	if c.AppenderRefsKebab, err = c.AppenderRefsKebab.resolveLookups(lookups); err != nil {
		return fmt.Errorf("appender-refs: %w", err)
	}
	if c.Refs, err = c.Refs.resolveLookups(lookups); err != nil {
		return fmt.Errorf("refs: %w", err)
	}
	if c.Primary, err = resolveStringLookup(lookups, c.Primary); err != nil {
		return fmt.Errorf("primary: %w", err)
	}
	if c.PrimaryKebab, err = resolveStringLookup(lookups, c.PrimaryKebab); err != nil {
		return fmt.Errorf("primary-ref: %w", err)
	}
	if c.Failovers, err = resolveStringListLookups(lookups, c.Failovers); err != nil {
		return fmt.Errorf("failovers: %w", err)
	}
	if c.FailoversKebab, err = resolveStringListLookups(lookups, c.FailoversKebab); err != nil {
		return fmt.Errorf("failover-refs: %w", err)
	}
	if c.RouteKey, err = resolveStringLookup(lookups, c.RouteKey); err != nil {
		return fmt.Errorf("routeKey: %w", err)
	}
	if c.RouteKeyKebab, err = resolveStringLookup(lookups, c.RouteKeyKebab); err != nil {
		return fmt.Errorf("route-key: %w", err)
	}
	if c.DefaultRoute, err = resolveStringLookup(lookups, c.DefaultRoute); err != nil {
		return fmt.Errorf("defaultRoute: %w", err)
	}
	if c.DefaultRouteKebab, err = resolveStringLookup(lookups, c.DefaultRouteKebab); err != nil {
		return fmt.Errorf("default-route: %w", err)
	}
	if c.Routes, err = resolveStringMapLookups(lookups, c.Routes); err != nil {
		return fmt.Errorf("routes: %w", err)
	}
	if err := c.Rewrite.resolveLookups(lookups); err != nil {
		return fmt.Errorf("rewrite: %w", err)
	}
	if c.Filters, err = resolveStringListLookups(lookups, c.Filters); err != nil {
		return fmt.Errorf("filters: %w", err)
	}
	if c.FilterRefs, err = resolveStringListLookups(lookups, c.FilterRefs); err != nil {
		return fmt.Errorf("filterRefs: %w", err)
	}
	if c.FilterRefsKebab, err = resolveStringListLookups(lookups, c.FilterRefsKebab); err != nil {
		return fmt.Errorf("filter-refs: %w", err)
	}
	return nil
}

func (c *layoutConfig) resolveLookups(lookups *LookupResolver) error {
	var err error
	if c.Type, err = resolveStringLookup(lookups, c.Type); err != nil {
		return fmt.Errorf("type: %w", err)
	}
	if c.Pattern, err = resolveStringLookup(lookups, c.Pattern); err != nil {
		return fmt.Errorf("pattern: %w", err)
	}
	if c.EventTemplate, err = resolveStringLookup(lookups, c.EventTemplate); err != nil {
		return fmt.Errorf("eventTemplate: %w", err)
	}
	if c.EventTemplateKebab, err = resolveStringLookup(lookups, c.EventTemplateKebab); err != nil {
		return fmt.Errorf("event-template: %w", err)
	}
	if c.EventTemplateURI, err = resolveStringLookup(lookups, c.EventTemplateURI); err != nil {
		return fmt.Errorf("eventTemplateUri: %w", err)
	}
	if c.EventTemplateURIKebab, err = resolveStringLookup(lookups, c.EventTemplateURIKebab); err != nil {
		return fmt.Errorf("event-template-uri: %w", err)
	}
	if c.EventTemplatePath, err = resolveStringLookup(lookups, c.EventTemplatePath); err != nil {
		return fmt.Errorf("eventTemplatePath: %w", err)
	}
	if c.EventTemplatePathKebab, err = resolveStringLookup(lookups, c.EventTemplatePathKebab); err != nil {
		return fmt.Errorf("event-template-path: %w", err)
	}
	if c.Header, err = resolveStringLookup(lookups, c.Header); err != nil {
		return fmt.Errorf("header: %w", err)
	}
	if c.Footer, err = resolveStringLookup(lookups, c.Footer); err != nil {
		return fmt.Errorf("footer: %w", err)
	}
	return nil
}
