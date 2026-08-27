package goarklog

import (
	"fmt"
	"strings"

	"goark.dev/log/internal/configxml"
)

func (a xmlAppender) config() (string, appenderConfig, error) {
	name := strings.TrimSpace(a.Name)
	if name == "" {
		return "", appenderConfig{}, fmt.Errorf("goark-log: XML appender name is empty")
	}
	queueSize, err := configxml.Int(a.QueueSize, "queueSize")
	if err != nil {
		return "", appenderConfig{}, fmt.Errorf("goark-log: XML appender %q: %w", name, err)
	}
	batchSize, err := configxml.Int(a.BatchSize, "batchSize")
	if err != nil {
		return "", appenderConfig{}, fmt.Errorf("goark-log: XML appender %q: %w", name, err)
	}
	flushOnWrite, err := configxml.Bool(a.FlushOnWrite, "flushOnWrite")
	if err != nil {
		return "", appenderConfig{}, fmt.Errorf("goark-log: XML appender %q: %w", name, err)
	}
	appendEnabled, err := configxml.BoolPointerStrict(a.Append, "append")
	if err != nil {
		return "", appenderConfig{}, fmt.Errorf("goark-log: XML appender %q: %w", name, err)
	}
	createOnDemand, err := configxml.Bool(a.CreateOnDemand, "createOnDemand")
	if err != nil {
		return "", appenderConfig{}, fmt.Errorf("goark-log: XML appender %q: %w", name, err)
	}
	waitRetries, err := configxml.Int(a.WaitRetries, "waitRetries")
	if err != nil {
		return "", appenderConfig{}, fmt.Errorf("goark-log: XML appender %q: %w", name, err)
	}
	layout, err := a.layout()
	if err != nil {
		return "", appenderConfig{}, fmt.Errorf("goark-log: XML appender %q layout: %w", name, err)
	}
	strategy := a.effectiveStrategy()
	appenderRefs, err := xmlAppenderRefs(a.AppenderRefs)
	if err != nil {
		return "", appenderConfig{}, fmt.Errorf("goark-log: XML appender %q: %w", name, err)
	}
	config := appenderConfig{
		Type:           xmlAppenderType(a.XMLName.Local, a.Type),
		Target:         xmlConsoleTarget(a.Target),
		URL:            a.URL,
		Method:         a.Method,
		Address:        a.Address,
		Network:        a.Network,
		Facility:       a.Facility,
		AppName:        a.AppName,
		ConnectTimeout: a.ConnectTimeout,
		WriteTimeout:   a.WriteTimeout,
		FileName:       a.FileName,
		Layout:         layout,
		AppenderRefs:   appenderRefs,
		Primary:        a.Primary,
		Failovers:      xmlFilterRefsFromAppenderRefs(a.Failovers),
		RouteKey:       a.RouteKey,
		DefaultRoute:   a.DefaultRoute,
		Routes:         xmlRoutes(a.Routes),
		Rewrite: rewriteBuildConfig{
			Attrs:  xmlKeyValuePairMap(a.KeyValuePair),
			Remove: xmlRemoveAttrs(a.Remove),
		},
		QueueSize:        queueSize,
		BatchSize:        batchSize,
		OverflowStrategy: a.OverflowStrategy,
		WaitStrategy:     a.WaitStrategy,
		WaitRetries:      waitRetries,
		SleepTime:        a.SleepTime,
		Timeout:          a.Timeout,
		BufferSize:       a.BufferSize,
		FlushOnWrite:     flushOnWrite,
		Append:           appendEnabled,
		CreateOnDemand:   createOnDemand,
		FilePermissions:  a.FilePermissions,
		Filters:          xmlFilterRefs(a.FilterRefs),
		Rolling: rollingConfig{
			FilePattern: a.FilePattern,
			Policies: rollingPoliciesConfig{
				SizeBasedTriggeringPolicy: rollingSizePolicyConfig{
					Size: a.Policies.Size.Size,
				},
				TimeBasedTriggeringPolicy: rollingTimePolicyConfig{
					Interval: a.Policies.Time.Interval,
					Modulate: configxml.BoolPointer(a.Policies.Time.Modulate),
				},
				CronTriggeringPolicy: rollingCronPolicyConfig{
					Schedule: a.Policies.Cron.Schedule,
				},
				OnStartupTriggeringPolicy: rollingStartupPolicyConfig{
					Enabled: configxml.BoolPointer(a.Policies.Startup.Enabled),
				},
			},
			Strategy: rollingStrategyConfig{
				Type:      strategy.Type,
				Max:       configxml.IntPointer(strategy.Max),
				FileIndex: strategy.FileIndex,
				Delete:    strategy.Delete.config(),
			},
		},
	}
	return name, config, nil
}

func (a xmlAppender) effectiveStrategy() xmlRollingStrategy {
	if !a.DirectStrategy.empty() {
		strategy := a.DirectStrategy
		if strings.TrimSpace(strategy.Type) == "" {
			strategy.Type = "directWrite"
		}
		return strategy
	}
	return a.Strategy
}

func (s xmlRollingStrategy) empty() bool {
	return strings.TrimSpace(s.Type) == "" &&
		strings.TrimSpace(s.Max) == "" &&
		strings.TrimSpace(s.FileIndex) == "" &&
		s.Delete.empty()
}

func (a xmlRollingDeleteAction) empty() bool {
	return strings.TrimSpace(a.BasePath) == "" &&
		strings.TrimSpace(a.MaxDepth) == "" &&
		strings.TrimSpace(a.MaxCount) == "" &&
		strings.TrimSpace(a.MaxSize) == "" &&
		strings.TrimSpace(a.Glob) == "" &&
		strings.TrimSpace(a.Age) == "" &&
		strings.TrimSpace(a.IfFileName.Glob) == "" &&
		strings.TrimSpace(a.IfLastModified.Age) == "" &&
		strings.TrimSpace(a.IfAccumulatedFileCount.Exceeds) == "" &&
		strings.TrimSpace(a.IfAccumulatedFileSize.Exceeds) == ""
}
