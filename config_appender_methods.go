package goarklog

import (
	"strings"
)

func (c appenderConfig) fileName() string {
	for _, value := range []string{c.FileName, c.FileNameKebab, c.Path} {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func (c appenderConfig) refs() []string {
	return c.appenderRefs().strings()
}

func (c appenderConfig) failoverRefs() []string {
	primary := firstNonBlank(c.Primary, c.PrimaryKebab)
	failovers := firstStringRefs(c.Failovers, c.FailoversKebab)
	if primary == "" && len(failovers) == 0 {
		return c.refs()
	}
	refs := make([]string, 0, 1+len(failovers))
	refs = append(refs, primary)
	refs = append(refs, failovers...)
	return refs
}

func (c appenderConfig) appenderRefs() appenderRefs {
	return firstAppenderRefs(c.AppenderRefs, c.AppenderRefsKebab, c.Refs)
}

func (c appenderConfig) appenderRefControls(filters map[string]Filter) ([]AppenderRef, error) {
	return c.appenderRefs().controls(filters)
}

func (c appenderConfig) filterRefs() []string {
	return firstStringRefs(c.Filters, c.FilterRefs, c.FilterRefsKebab)
}

func (c appenderConfig) queueSize() int {
	if c.QueueSize != 0 {
		return c.QueueSize
	}
	return c.QueueSizeKebab
}

func (c appenderConfig) overflowStrategy() string {
	if strings.TrimSpace(c.OverflowStrategy) != "" {
		return c.OverflowStrategy
	}
	return c.OverflowStrategyKebab
}

func (c appenderConfig) waitStrategy() string {
	return firstNonBlank(c.WaitStrategy, c.WaitStrategyKebab)
}

func (c appenderConfig) waitOptions() AsyncWaitOptions {
	return AsyncWaitOptions{
		Retries:   firstNonZero(c.WaitRetries, c.WaitRetriesKebab),
		SleepTime: parseOptionalDuration(firstNonBlank(c.SleepTime, c.SleepTimeKebab)),
		Timeout:   parseOptionalDuration(c.Timeout),
	}
}

func (c appenderConfig) bufferSize() string {
	return firstNonBlank(c.BufferSize, c.BufferSizeKebab)
}

func (c appenderConfig) flushOnWrite() bool {
	return c.FlushOnWrite || c.FlushOnWriteKebab
}

func (c appenderConfig) createOnDemand() bool {
	return c.CreateOnDemand || c.CreateOnDemandKebab
}

func (c appenderConfig) filePermissions() string {
	return firstNonBlank(c.FilePermissions, c.FilePermissionsKebab)
}

func (c appenderConfig) routeKey() string {
	return firstNonBlank(c.RouteKey, c.RouteKeyKebab)
}

func (c appenderConfig) defaultRoute() string {
	return firstNonBlank(c.DefaultRoute, c.DefaultRouteKebab)
}

func (c appenderConfig) routes() map[string]string {
	return copyStringMap(c.Routes)
}

func (c appenderConfig) rewriteConfig() RewriteBuildConfig {
	return RewriteBuildConfig{
		Attrs:       c.Rewrite.attrs(),
		RemoveAttrs: c.Rewrite.removeKeys(),
	}
}

func (c appenderConfig) appenderBuildConfig(name string, layout Layout, delegates []Appender, routes map[string]Appender, defaultRoute Appender) AppenderBuildConfig {
	return AppenderBuildConfig{
		Name:             name,
		Type:             c.Type,
		Target:           c.Target,
		URL:              c.URL,
		Method:           c.Method,
		Address:          c.Address,
		Network:          c.Network,
		Facility:         c.Facility,
		AppName:          firstNonBlank(c.AppName, c.AppNameKebab),
		ConnectTimeout:   firstNonBlank(c.ConnectTimeout, c.ConnectTimeoutKebab),
		WriteTimeout:     firstNonBlank(c.WriteTimeout, c.WriteTimeoutKebab),
		FileName:         c.fileName(),
		Layout:           layout,
		AppenderRefs:     c.refs(),
		Delegates:        append([]Appender(nil), delegates...),
		Routes:           copyAppenderMap(routes),
		DefaultRoute:     defaultRoute,
		RouteKey:         c.routeKey(),
		QueueSize:        c.queueSize(),
		OverflowStrategy: c.overflowStrategy(),
		WaitStrategy:     c.waitStrategy(),
		WaitOptions:      c.waitOptions(),
		BufferSize:       c.bufferSize(),
		FlushOnWrite:     c.flushOnWrite(),
		Append:           c.Append,
		CreateOnDemand:   c.createOnDemand(),
		FilePermissions:  c.filePermissions(),
		Rolling: RollingBuildConfig{
			FilePattern:     c.Rolling.filePattern(),
			MaxSize:         c.Rolling.maxSize(),
			Interval:        c.Rolling.interval(),
			CronSchedule:    c.Rolling.cronSchedule(),
			TimeModulate:    c.Rolling.timeModulate(),
			OnStartup:       c.Rolling.onStartup(),
			MaxBackups:      c.Rolling.maxBackupsPointer(),
			MaxAge:          c.Rolling.maxAge(),
			FileIndex:       c.Rolling.fileIndex(),
			DirectWrite:     c.Rolling.directWrite(),
			Gzip:            c.Rolling.gzipEnabled(),
			AsyncActions:    c.Rolling.asyncActions(),
			DeleteActions:   c.Rolling.deleteActions(c.fileName()),
			ActionQueueSize: c.Rolling.actionQueueSize(),
		},
		Rewrite: c.rewriteConfig(),
	}
}

func copyAppenderMap(values map[string]Appender) map[string]Appender {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]Appender, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}
