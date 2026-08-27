package goarklog

import (
	"fmt"
	"os"
	"strings"
)

func buildConsolePlugin(config AppenderBuildConfig) (Appender, error) {
	target := strings.ToLower(strings.TrimSpace(config.Target))
	switch target {
	case "", "stderr":
		return NewConsoleAppender(WithConsoleName(config.Name), WithConsoleLayout(config.Layout), WithConsoleWriter(os.Stderr)), nil
	case "stdout":
		return NewConsoleAppender(WithConsoleName(config.Name), WithConsoleLayout(config.Layout), WithConsoleWriter(os.Stdout)), nil
	default:
		return nil, fmt.Errorf("goark-log: appender %q console target %q is invalid", config.Name, config.Target)
	}
}

func buildFilePlugin(config AppenderBuildConfig) (Appender, error) {
	if config.FileName == "" {
		return nil, fmt.Errorf("goark-log: appender %q fileName is empty", config.Name)
	}
	options := []FileOption{
		WithFileName(config.Name),
		WithFileLayout(config.Layout),
	}
	if strings.TrimSpace(config.BufferSize) != "" {
		size, err := ParseByteSize(config.BufferSize)
		if err != nil {
			return nil, fmt.Errorf("goark-log: appender %q: %w", config.Name, err)
		}
		options = append(options, WithFileBufferSize(int(size)))
	}
	if config.FlushOnWrite {
		options = append(options, WithFileFlushOnWrite(true))
	}
	if config.Append != nil {
		options = append(options, WithFileAppend(*config.Append))
	}
	if config.CreateOnDemand {
		options = append(options, WithFileCreateOnDemand(true))
	}
	if strings.TrimSpace(config.FilePermissions) != "" {
		permissions, err := parseLogFilePermissions(config.FilePermissions)
		if err != nil {
			return nil, fmt.Errorf("goark-log: appender %q: %w", config.Name, err)
		}
		options = append(options, WithFilePermissions(permissions))
	}
	return NewFileAppender(config.FileName, options...)
}

func buildJSONPlugin(config AppenderBuildConfig) (Appender, error) {
	if strings.TrimSpace(config.FileName) != "" {
		options := []JSONAppenderOption{
			WithJSONAppenderName(config.Name),
		}
		if strings.TrimSpace(config.BufferSize) != "" {
			size, err := ParseByteSize(config.BufferSize)
			if err != nil {
				return nil, fmt.Errorf("goark-log: appender %q: %w", config.Name, err)
			}
			options = append(options, WithJSONAppenderBufferSize(int(size)))
		}
		if config.FlushOnWrite {
			options = append(options, WithJSONAppenderFlushOnWrite(true))
		}
		return NewJSONFileAppender(config.FileName, options...)
	}
	switch strings.ToLower(strings.TrimSpace(config.Target)) {
	case "", "stderr":
		return NewJSONAppender(WithJSONAppenderName(config.Name), WithJSONAppenderWriter(os.Stderr)), nil
	case "stdout":
		return NewJSONAppender(WithJSONAppenderName(config.Name), WithJSONAppenderWriter(os.Stdout)), nil
	case "file":
		return nil, fmt.Errorf("goark-log: appender %q JSON target file requires fileName", config.Name)
	default:
		return nil, fmt.Errorf("goark-log: appender %q JSON target %q is invalid", config.Name, config.Target)
	}
}

func buildRollingPlugin(config AppenderBuildConfig) (Appender, error) {
	if config.FileName == "" {
		return nil, fmt.Errorf("goark-log: appender %q fileName is empty", config.Name)
	}
	options := []RollingFileOption{
		WithRollingFileName(config.Name),
		WithRollingFileLayout(config.Layout),
	}
	if strings.TrimSpace(config.BufferSize) != "" {
		size, err := ParseByteSize(config.BufferSize)
		if err != nil {
			return nil, fmt.Errorf("goark-log: appender %q: %w", config.Name, err)
		}
		options = append(options, WithRollingFileBufferSize(int(size)))
	}
	if config.FlushOnWrite {
		options = append(options, WithRollingFileFlushOnWrite(true))
	}
	if config.Append != nil {
		options = append(options, WithRollingFileAppend(*config.Append))
	}
	if config.CreateOnDemand {
		options = append(options, WithRollingFileCreateOnDemand(true))
	}
	if strings.TrimSpace(config.FilePermissions) != "" {
		permissions, err := parseLogFilePermissions(config.FilePermissions)
		if err != nil {
			return nil, fmt.Errorf("goark-log: appender %q: %w", config.Name, err)
		}
		options = append(options, WithRollingFilePermissions(permissions))
	}
	if strings.TrimSpace(config.Rolling.FilePattern) != "" {
		options = append(options, WithRollingFilePattern(config.Rolling.FilePattern))
	}
	if value := strings.ToLower(strings.TrimSpace(config.Rolling.FileIndex)); value != "" {
		switch value {
		case "min":
			options = append(options, WithRollingFileIndexMode(RollingFileIndexMin))
		case "max":
			options = append(options, WithRollingFileIndexMode(RollingFileIndexMax))
		case "nomax", "no-max", "none":
			options = append(options, WithRollingFileIndexMode(RollingFileIndexNoMax))
		default:
			return nil, fmt.Errorf("goark-log: appender %q rolling fileIndex %q is unsupported", config.Name, config.Rolling.FileIndex)
		}
	}
	if config.Rolling.DirectWrite {
		options = append(options, WithRollingDirectWrite(true))
	}
	if value := config.Rolling.MaxSize; value != "" {
		size, err := ParseByteSize(value)
		if err != nil {
			return nil, fmt.Errorf("goark-log: appender %q: %w", config.Name, err)
		}
		options = append(options, WithRollingMaxSize(size))
	}
	if value := config.Rolling.Interval; strings.TrimSpace(value) != "" {
		interval, err := ParseRollingInterval(value)
		if err != nil {
			return nil, fmt.Errorf("goark-log: appender %q: %w", config.Name, err)
		}
		options = append(options, WithRollingInterval(interval))
	}
	if strings.TrimSpace(config.Rolling.CronSchedule) != "" {
		options = append(options, WithRollingCronSchedule(config.Rolling.CronSchedule))
	}
	if config.Rolling.TimeModulate != nil {
		options = append(options, WithRollingTimeModulate(*config.Rolling.TimeModulate))
	}
	if config.Rolling.OnStartup {
		options = append(options, WithRolloverOnStartup(true))
	}
	if config.Rolling.MaxBackups != nil {
		options = append(options, WithRollingMaxBackups(*config.Rolling.MaxBackups))
	}
	if strings.TrimSpace(config.Rolling.MaxAge) != "" {
		age, err := ParseRollingMaxAge(config.Rolling.MaxAge)
		if err != nil {
			return nil, fmt.Errorf("goark-log: appender %q: %w", config.Name, err)
		}
		options = append(options, WithRollingMaxAge(age))
	}
	if config.Rolling.Gzip {
		options = append(options, WithRollingGzip(true))
	}
	if config.Rolling.AsyncActions {
		options = append(options, WithRollingAsyncActions(true))
	}
	if config.Rolling.ActionQueueSize > 0 {
		options = append(options, WithRollingActionQueueSize(config.Rolling.ActionQueueSize))
	}
	if len(config.Rolling.DeleteActions) > 0 {
		actions := make([]RollingDeleteAction, 0, len(config.Rolling.DeleteActions))
		for index, actionConfig := range config.Rolling.DeleteActions {
			action, err := buildRollingDeleteAction(actionConfig)
			if err != nil {
				return nil, fmt.Errorf("goark-log: appender %q rolling delete action %d: %w", config.Name, index, err)
			}
			actions = append(actions, action)
		}
		options = append(options, WithRollingDeleteActions(actions...))
	}
	return NewRollingFileAppender(config.FileName, options...)
}

func buildRollingDeleteAction(config RollingDeleteBuildConfig) (RollingDeleteAction, error) {
	action := RollingDeleteAction{
		BasePath: config.BasePath,
		MaxDepth: config.MaxDepth,
		Glob:     config.Glob,
		MaxCount: config.MaxCount,
	}
	if strings.TrimSpace(config.MaxAge) != "" {
		age, err := ParseRollingMaxAge(config.MaxAge)
		if err != nil {
			return RollingDeleteAction{}, err
		}
		action.MaxAge = age
	}
	if strings.TrimSpace(config.MaxSize) != "" {
		size, err := ParseByteSize(config.MaxSize)
		if err != nil {
			return RollingDeleteAction{}, err
		}
		action.MaxSize = size
	}
	return action, nil
}

func buildAsyncPlugin(config AppenderBuildConfig) (Appender, error) {
	if len(config.Delegates) == 0 {
		return nil, fmt.Errorf("goark-log: async appender %q requires appenderRefs", config.Name)
	}
	strategy, err := ParseAsyncOverflowStrategy(config.OverflowStrategy)
	if err != nil {
		return nil, fmt.Errorf("goark-log: async appender %q: %w", config.Name, err)
	}
	options := []AsyncOption{
		WithAsyncName(config.Name),
		WithAsyncOverflowStrategy(strategy),
	}
	if err := validateAsyncWaitOptions(config.WaitOptions); err != nil {
		return nil, fmt.Errorf("goark-log: async appender %q: %w", config.Name, err)
	}
	options = append(options, WithAsyncWaitOptions(config.WaitOptions))
	waitStrategy, err := ParseAsyncWaitStrategy(config.WaitStrategy)
	if err != nil {
		return nil, fmt.Errorf("goark-log: async appender %q: %w", config.Name, err)
	}
	options = append(options, WithAsyncWaitStrategy(waitStrategy))
	if config.QueueSize != 0 {
		options = append(options, WithAsyncQueueSize(config.QueueSize))
	}
	if config.BatchSize != 0 {
		options = append(options, WithAsyncBatchSize(config.BatchSize))
	}
	return NewAsyncAppender(config.Delegates, options...)
}

func buildFailoverPlugin(config AppenderBuildConfig) (Appender, error) {
	if len(config.Delegates) < 2 {
		return nil, fmt.Errorf("goark-log: failover appender %q requires primary and failovers", config.Name)
	}
	return NewFailoverAppender(config.Delegates[0], config.Delegates[1:],
		WithFailoverName(config.Name),
		WithFailoverCloseChildren(false),
	)
}

func buildRoutingPlugin(config AppenderBuildConfig) (Appender, error) {
	options := []RoutingOption{
		WithRoutingName(config.Name),
		WithRoutingCloseChildren(false),
	}
	if strings.TrimSpace(config.RouteKey) != "" {
		options = append(options, WithRoutingAttrKey(config.RouteKey))
	}
	if config.DefaultRoute != nil {
		options = append(options, WithRoutingDefault(config.DefaultRoute))
	}
	return NewRoutingAppender(config.Routes, options...)
}

func buildRewritePlugin(config AppenderBuildConfig) (Appender, error) {
	if len(config.Delegates) != 1 {
		return nil, fmt.Errorf("goark-log: rewrite appender %q requires exactly one appenderRef", config.Name)
	}
	return NewRewriteAppender(config.Delegates[0], newAttributeRewritePolicy(config.Rewrite),
		WithRewriteName(config.Name),
		WithRewriteCloseDelegate(false),
	)
}
