package configfile

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

func (c *rollingConfig) resolveLookups(lookups *LookupResolver) error {
	var err error
	if c.FilePattern, err = resolveStringLookup(lookups, c.FilePattern); err != nil {
		return fmt.Errorf("filePattern: %w", err)
	}
	if c.FilePatternKebab, err = resolveStringLookup(lookups, c.FilePatternKebab); err != nil {
		return fmt.Errorf("file-pattern: %w", err)
	}
	if c.MaxSize, err = resolveStringLookup(lookups, c.MaxSize); err != nil {
		return fmt.Errorf("maxSize: %w", err)
	}
	if c.MaxSizeKebab, err = resolveStringLookup(lookups, c.MaxSizeKebab); err != nil {
		return fmt.Errorf("max-size: %w", err)
	}
	if c.Interval, err = resolveStringLookup(lookups, c.Interval); err != nil {
		return fmt.Errorf("interval: %w", err)
	}
	if c.Cron, err = resolveStringLookup(lookups, c.Cron); err != nil {
		return fmt.Errorf("cron: %w", err)
	}
	if c.CronSchedule, err = resolveStringLookup(lookups, c.CronSchedule); err != nil {
		return fmt.Errorf("cronSchedule: %w", err)
	}
	if c.CronScheduleKebab, err = resolveStringLookup(lookups, c.CronScheduleKebab); err != nil {
		return fmt.Errorf("cron-schedule: %w", err)
	}
	if c.MaxAge, err = resolveStringLookup(lookups, c.MaxAge); err != nil {
		return fmt.Errorf("maxAge: %w", err)
	}
	if c.MaxAgeKebab, err = resolveStringLookup(lookups, c.MaxAgeKebab); err != nil {
		return fmt.Errorf("max-age: %w", err)
	}
	if err := c.Policies.resolveLookups(lookups); err != nil {
		return fmt.Errorf("policies: %w", err)
	}
	if err := c.Strategy.resolveLookups(lookups); err != nil {
		return fmt.Errorf("strategy: %w", err)
	}
	return nil
}

func (c *rollingPoliciesConfig) resolveLookups(lookups *LookupResolver) error {
	policies := []*rollingSizePolicyConfig{
		&c.Size,
		&c.SizeKebab,
		&c.SizeBasedTriggeringPolicy,
		&c.SizeBasedTriggeringPolicyXML,
	}
	for _, policy := range policies {
		if err := policy.resolveLookups(lookups); err != nil {
			return err
		}
	}
	timePolicies := []*rollingTimePolicyConfig{
		&c.Time,
		&c.TimeKebab,
		&c.TimeBasedTriggeringPolicy,
		&c.TimeBasedTriggeringPolicyXML,
	}
	for _, policy := range timePolicies {
		if err := policy.resolveLookups(lookups); err != nil {
			return err
		}
	}
	cronPolicies := []*rollingCronPolicyConfig{
		&c.Cron,
		&c.CronKebab,
		&c.CronTriggeringPolicy,
		&c.CronTriggeringPolicyXML,
	}
	for _, policy := range cronPolicies {
		if err := policy.resolveLookups(lookups); err != nil {
			return err
		}
	}
	return nil
}

func (c *rollingSizePolicyConfig) resolveLookups(lookups *LookupResolver) error {
	var err error
	if c.Size, err = resolveStringLookup(lookups, c.Size); err != nil {
		return fmt.Errorf("size: %w", err)
	}
	if c.MaxSize, err = resolveStringLookup(lookups, c.MaxSize); err != nil {
		return fmt.Errorf("maxSize: %w", err)
	}
	if c.MaxSizeKebab, err = resolveStringLookup(lookups, c.MaxSizeKebab); err != nil {
		return fmt.Errorf("max-size: %w", err)
	}
	return nil
}

func (c *rollingTimePolicyConfig) resolveLookups(lookups *LookupResolver) error {
	var err error
	if c.Interval, err = resolveStringLookup(lookups, c.Interval); err != nil {
		return fmt.Errorf("interval: %w", err)
	}
	if c.Every, err = resolveStringLookup(lookups, c.Every); err != nil {
		return fmt.Errorf("every: %w", err)
	}
	if c.Unit, err = resolveStringLookup(lookups, c.Unit); err != nil {
		return fmt.Errorf("unit: %w", err)
	}
	return nil
}

func (c *rollingCronPolicyConfig) resolveLookups(lookups *LookupResolver) error {
	var err error
	if c.Schedule, err = resolveStringLookup(lookups, c.Schedule); err != nil {
		return fmt.Errorf("schedule: %w", err)
	}
	if c.Cron, err = resolveStringLookup(lookups, c.Cron); err != nil {
		return fmt.Errorf("cron: %w", err)
	}
	if c.CronSchedule, err = resolveStringLookup(lookups, c.CronSchedule); err != nil {
		return fmt.Errorf("cronSchedule: %w", err)
	}
	if c.CronKebab, err = resolveStringLookup(lookups, c.CronKebab); err != nil {
		return fmt.Errorf("cron-schedule: %w", err)
	}
	return nil
}

func (c *rollingStrategyConfig) resolveLookups(lookups *LookupResolver) error {
	var err error
	if c.MaxAge, err = resolveStringLookup(lookups, c.MaxAge); err != nil {
		return fmt.Errorf("maxAge: %w", err)
	}
	if c.MaxAgeKebab, err = resolveStringLookup(lookups, c.MaxAgeKebab); err != nil {
		return fmt.Errorf("max-age: %w", err)
	}
	if c.FileIndex, err = resolveStringLookup(lookups, c.FileIndex); err != nil {
		return fmt.Errorf("fileIndex: %w", err)
	}
	if c.FileIndexKebab, err = resolveStringLookup(lookups, c.FileIndexKebab); err != nil {
		return fmt.Errorf("file-index: %w", err)
	}
	if err := c.Delete.resolveLookups(lookups); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	for index := range c.DeleteActions {
		if err := c.DeleteActions[index].resolveLookups(lookups); err != nil {
			return fmt.Errorf("deleteActions[%d]: %w", index, err)
		}
	}
	for index := range c.DeleteActionsKebab {
		if err := c.DeleteActionsKebab[index].resolveLookups(lookups); err != nil {
			return fmt.Errorf("delete-actions[%d]: %w", index, err)
		}
	}
	return nil
}

func (c *rollingDeleteActionConfig) resolveLookups(lookups *LookupResolver) error {
	var err error
	if c.BasePath, err = resolveStringLookup(lookups, c.BasePath); err != nil {
		return fmt.Errorf("basePath: %w", err)
	}
	if c.BasePathKebab, err = resolveStringLookup(lookups, c.BasePathKebab); err != nil {
		return fmt.Errorf("base-path: %w", err)
	}
	if c.Glob, err = resolveStringLookup(lookups, c.Glob); err != nil {
		return fmt.Errorf("glob: %w", err)
	}
	if c.Age, err = resolveStringLookup(lookups, c.Age); err != nil {
		return fmt.Errorf("age: %w", err)
	}
	if c.MaxSize, err = resolveStringLookup(lookups, c.MaxSize); err != nil {
		return fmt.Errorf("maxSize: %w", err)
	}
	if c.MaxSizeKebab, err = resolveStringLookup(lookups, c.MaxSizeKebab); err != nil {
		return fmt.Errorf("max-size: %w", err)
	}
	if err := c.IfFileName.resolveLookups(lookups); err != nil {
		return fmt.Errorf("ifFileName: %w", err)
	}
	if err := c.IfFileNameKebab.resolveLookups(lookups); err != nil {
		return fmt.Errorf("if-file-name: %w", err)
	}
	if err := c.IfLastModified.resolveLookups(lookups); err != nil {
		return fmt.Errorf("ifLastModified: %w", err)
	}
	if err := c.IfLastModifiedKebab.resolveLookups(lookups); err != nil {
		return fmt.Errorf("if-last-modified: %w", err)
	}
	if err := c.IfAccumulatedFileSize.resolveLookups(lookups); err != nil {
		return fmt.Errorf("ifAccumulatedFileSize: %w", err)
	}
	if err := c.IfAccumulatedFileSizeKebab.resolveLookups(lookups); err != nil {
		return fmt.Errorf("if-accumulated-file-size: %w", err)
	}
	return nil
}

func (c *rollingDeleteFileNameConfig) resolveLookups(lookups *LookupResolver) error {
	var err error
	if c.Glob, err = resolveStringLookup(lookups, c.Glob); err != nil {
		return fmt.Errorf("glob: %w", err)
	}
	return nil
}

func (c *rollingDeleteLastModifiedConfig) resolveLookups(lookups *LookupResolver) error {
	var err error
	if c.Age, err = resolveStringLookup(lookups, c.Age); err != nil {
		return fmt.Errorf("age: %w", err)
	}
	return nil
}

func (c *rollingDeleteAccumulatedSizeConfig) resolveLookups(lookups *LookupResolver) error {
	var err error
	if c.Exceeds, err = resolveStringLookup(lookups, c.Exceeds); err != nil {
		return fmt.Errorf("exceeds: %w", err)
	}
	return nil
}
