package configfile

import (
	"strings"

	"goark.dev/log/internal/textutil"
)

func (c rollingConfig) filePattern() string {
	return textutil.FirstNonBlank(c.FilePattern, c.FilePatternKebab)
}

func (c rollingConfig) maxSize() string {
	if value := c.Policies.sizePolicy().size(); value != "" {
		return value
	}
	if strings.TrimSpace(c.MaxSize) != "" {
		return strings.TrimSpace(c.MaxSize)
	}
	return strings.TrimSpace(c.MaxSizeKebab)
}

func (c rollingConfig) interval() string {
	if value := c.Policies.timePolicy().interval(); value != "" {
		return value
	}
	return strings.TrimSpace(c.Interval)
}

func (c rollingConfig) cronSchedule() string {
	if value := c.Policies.cronPolicy().schedule(); value != "" {
		return value
	}
	return textutil.FirstNonBlank(c.CronSchedule, c.CronScheduleKebab, c.Cron)
}

func (c rollingConfig) timeModulate() *bool {
	return c.Policies.timePolicy().Modulate
}

func (c rollingConfig) maxAge() string {
	return textutil.FirstNonBlank(c.Strategy.MaxAge, c.Strategy.MaxAgeKebab, c.MaxAge, c.MaxAgeKebab)
}

func (c rollingConfig) fileIndex() string {
	return textutil.FirstNonBlank(c.Strategy.FileIndex, c.Strategy.FileIndexKebab)
}

func (c rollingConfig) directWrite() bool {
	strategyType := strings.ToLower(strings.TrimSpace(c.Strategy.Type))
	return c.DirectWrite ||
		c.DirectWriteKebab ||
		c.Strategy.DirectWrite ||
		c.Strategy.DirectWriteKebab ||
		strategyType == "directwrite" ||
		strategyType == "direct-write" ||
		strategyType == "directwriterolloverstrategy"
}

func (c rollingConfig) onStartup() bool {
	if enabled := c.Policies.startupPolicy().Enabled; enabled != nil {
		return *enabled
	}
	return c.OnStartup || c.OnStartupKebab
}

func (c rollingConfig) maxBackups() (int, bool) {
	if c.MaxBackups != nil {
		return *c.MaxBackups, true
	}
	if c.MaxBackupsKebab != nil {
		return *c.MaxBackupsKebab, true
	}
	return 0, false
}

func (c rollingConfig) maxBackupsPointer() *int {
	if c.Strategy.Max != nil {
		value := *c.Strategy.Max
		return &value
	}
	if c.Strategy.MaxBackups != nil {
		value := *c.Strategy.MaxBackups
		return &value
	}
	if c.Strategy.MaxBackupsKebab != nil {
		value := *c.Strategy.MaxBackupsKebab
		return &value
	}
	if c.MaxBackups != nil {
		value := *c.MaxBackups
		return &value
	}
	if c.MaxBackupsKebab != nil {
		value := *c.MaxBackupsKebab
		return &value
	}
	return nil
}

func (c rollingConfig) gzipEnabled() bool {
	return c.Gzip || c.Compress || c.Strategy.Compression.Gzip || c.Strategy.Compression.Compress
}

func (c rollingConfig) asyncActions() bool {
	return c.AsyncActions || c.AsyncActionsKebab ||
		c.Strategy.AsyncActions || c.Strategy.AsyncActionsKebab ||
		c.Strategy.Compression.Async || c.Strategy.Delete.Async ||
		containsAsyncDeleteAction(c.Strategy.DeleteActions) ||
		containsAsyncDeleteAction(c.Strategy.DeleteActionsKebab)
}

func (c rollingConfig) actionQueueSize() int {
	if c.ActionQueueSize > 0 {
		return c.ActionQueueSize
	}
	if c.ActionQueueSizeKebab > 0 {
		return c.ActionQueueSizeKebab
	}
	if c.Strategy.ActionQueueSize > 0 {
		return c.Strategy.ActionQueueSize
	}
	return c.Strategy.ActionQueueSizeKebab
}

func (c rollingConfig) deleteActions(fileName string) []RollingDeleteBuildConfig {
	defaultBase := c.defaultDeleteBasePath(fileName)
	configs := make([]rollingDeleteActionConfig, 0, 1+len(c.Strategy.DeleteActions)+len(c.Strategy.DeleteActionsKebab))
	if !c.Strategy.Delete.empty() {
		configs = append(configs, c.Strategy.Delete)
	}
	for _, action := range c.Strategy.DeleteActions {
		if !action.empty() {
			configs = append(configs, action)
		}
	}
	for _, action := range c.Strategy.DeleteActionsKebab {
		if !action.empty() {
			configs = append(configs, action)
		}
	}
	if len(configs) == 0 {
		return nil
	}
	actions := make([]RollingDeleteBuildConfig, 0, len(configs))
	for _, config := range configs {
		action := config.build(defaultBase)
		actions = append(actions, action)
	}
	return actions
}

func (c rollingConfig) defaultDeleteBasePath(fileName string) string {
	if pattern := c.filePattern(); pattern != "" {
		return filepathDir(pattern)
	}
	if strings.TrimSpace(fileName) != "" {
		return filepathDir(fileName)
	}
	return "."
}

func (c rollingPoliciesConfig) sizePolicy() rollingSizePolicyConfig {
	for _, policy := range []rollingSizePolicyConfig{
		c.Size,
		c.SizeKebab,
		c.SizeBasedTriggeringPolicy,
		c.SizeBasedTriggeringPolicyXML,
	} {
		if !policy.empty() {
			return policy
		}
	}
	return rollingSizePolicyConfig{}
}

func (c rollingPoliciesConfig) timePolicy() rollingTimePolicyConfig {
	for _, policy := range []rollingTimePolicyConfig{
		c.Time,
		c.TimeKebab,
		c.TimeBasedTriggeringPolicy,
		c.TimeBasedTriggeringPolicyXML,
	} {
		if !policy.empty() {
			return policy
		}
	}
	return rollingTimePolicyConfig{}
}

func (c rollingPoliciesConfig) cronPolicy() rollingCronPolicyConfig {
	for _, policy := range []rollingCronPolicyConfig{
		c.Cron,
		c.CronKebab,
		c.CronTriggeringPolicy,
		c.CronTriggeringPolicyXML,
	} {
		if !policy.empty() {
			return policy
		}
	}
	return rollingCronPolicyConfig{}
}

func (c rollingPoliciesConfig) startupPolicy() rollingStartupPolicyConfig {
	for _, policy := range []rollingStartupPolicyConfig{
		c.Startup,
		c.StartupKebab,
		c.OnStartupTriggeringPolicy,
		c.OnStartupTriggeringPolicyXML,
	} {
		if policy.Enabled != nil {
			return policy
		}
	}
	return rollingStartupPolicyConfig{}
}

func (c rollingSizePolicyConfig) empty() bool {
	return textutil.FirstNonBlank(c.Size, c.MaxSize, c.MaxSizeKebab) == ""
}

func (c rollingSizePolicyConfig) size() string {
	return textutil.FirstNonBlank(c.Size, c.MaxSize, c.MaxSizeKebab)
}

func (c rollingTimePolicyConfig) empty() bool {
	return textutil.FirstNonBlank(c.Interval, c.Every, c.Unit) == "" && c.Modulate == nil
}

func (c rollingTimePolicyConfig) interval() string {
	if strings.TrimSpace(c.Unit) == "" {
		return textutil.FirstNonBlank(c.Interval, c.Every)
	}
	return strings.TrimSpace(textutil.FirstNonBlank(c.Interval, c.Every)) + strings.TrimSpace(c.Unit)
}

func (c rollingCronPolicyConfig) empty() bool {
	return c.schedule() == ""
}

func (c rollingCronPolicyConfig) schedule() string {
	return textutil.FirstNonBlank(c.Schedule, c.CronSchedule, c.CronKebab, c.Cron)
}

func (c rollingDeleteActionConfig) empty() bool {
	return textutil.FirstNonBlank(c.BasePath, c.BasePathKebab, c.Glob, c.Age,
		c.IfFileName.Glob, c.IfFileNameKebab.Glob,
		c.IfLastModified.Age, c.IfLastModifiedKebab.Age,
		c.MaxSize, c.MaxSizeKebab,
		c.IfAccumulatedFileSize.Exceeds, c.IfAccumulatedFileSizeKebab.Exceeds) == "" &&
		c.MaxDepth == nil && c.MaxDepthKebab == nil &&
		c.MaxCount == nil && c.MaxCountKebab == nil &&
		c.IfAccumulatedFileCount.Exceeds == 0 && c.IfAccumulatedFileCountKebab.Exceeds == 0
}

func (c rollingDeleteActionConfig) build(defaultBase string) RollingDeleteBuildConfig {
	config := RollingDeleteBuildConfig{
		BasePath: textutil.FirstNonBlank(c.BasePath, c.BasePathKebab, defaultBase),
		Glob:     textutil.FirstNonBlank(c.Glob, c.IfFileName.Glob, c.IfFileNameKebab.Glob),
		MaxAge:   textutil.FirstNonBlank(c.Age, c.IfLastModified.Age, c.IfLastModifiedKebab.Age),
		MaxSize:  textutil.FirstNonBlank(c.MaxSize, c.MaxSizeKebab, c.IfAccumulatedFileSize.Exceeds, c.IfAccumulatedFileSizeKebab.Exceeds),
	}
	if c.MaxDepth != nil {
		config.MaxDepth = *c.MaxDepth
	} else if c.MaxDepthKebab != nil {
		config.MaxDepth = *c.MaxDepthKebab
	}
	if c.MaxCount != nil {
		config.MaxCount = *c.MaxCount
	} else if c.MaxCountKebab != nil {
		config.MaxCount = *c.MaxCountKebab
	} else if c.IfAccumulatedFileCount.Exceeds > 0 {
		config.MaxCount = c.IfAccumulatedFileCount.Exceeds
	} else if c.IfAccumulatedFileCountKebab.Exceeds > 0 {
		config.MaxCount = c.IfAccumulatedFileCountKebab.Exceeds
	}
	return config
}

func containsAsyncDeleteAction(actions []rollingDeleteActionConfig) bool {
	for _, action := range actions {
		if action.Async {
			return true
		}
	}
	return false
}

func filepathDir(path string) string {
	index := strings.LastIndexAny(path, `/\`)
	if index < 0 {
		return "."
	}
	return path[:index]
}
