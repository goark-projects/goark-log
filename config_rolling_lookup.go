package goarklog

import (
	"fmt"
)

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
