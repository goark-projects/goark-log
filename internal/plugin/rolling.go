package plugin

import (
	"fmt"
	"strings"

	"goark.dev/log/internal/configvalue"
	"goark.dev/log/internal/logfile"
	"goark.dev/log/internal/rollingfile"
	internalrouter "goark.dev/log/internal/router"
)

func buildRollingPlugin(config AppenderBuildConfig) (internalrouter.Appender, error) {
	if config.FileName == "" {
		return nil, fmt.Errorf("goark-log: appender %q fileName is empty", config.Name)
	}
	options := []rollingfile.RollingFileOption{
		rollingfile.WithRollingFileName(config.Name),
		rollingfile.WithRollingFileLayout(config.Layout),
	}
	if strings.TrimSpace(config.BufferSize) != "" {
		size, err := configvalue.ByteSize(config.BufferSize)
		if err != nil {
			return nil, fmt.Errorf("goark-log: appender %q: %w", config.Name, err)
		}
		options = append(options, rollingfile.WithRollingFileBufferSize(int(size)))
	}
	if config.FlushOnWrite {
		options = append(options, rollingfile.WithRollingFileFlushOnWrite(true))
	}
	if config.Append != nil {
		options = append(options, rollingfile.WithRollingFileAppend(*config.Append))
	}
	if config.CreateOnDemand {
		options = append(options, rollingfile.WithRollingFileCreateOnDemand(true))
	}
	if strings.TrimSpace(config.FilePermissions) != "" {
		permissions, err := logfile.ParsePermissions(config.FilePermissions)
		if err != nil {
			return nil, fmt.Errorf("goark-log: appender %q: %w", config.Name, err)
		}
		options = append(options, rollingfile.WithRollingFilePermissions(permissions))
	}
	if strings.TrimSpace(config.Rolling.FilePattern) != "" {
		options = append(options, rollingfile.WithRollingFilePattern(config.Rolling.FilePattern))
	}
	if value := strings.ToLower(strings.TrimSpace(config.Rolling.FileIndex)); value != "" {
		indexMode, ok := rollingFileIndexMode(value)
		if !ok {
			return nil, fmt.Errorf("goark-log: appender %q rolling fileIndex %q is unsupported", config.Name, config.Rolling.FileIndex)
		}
		options = append(options, rollingfile.WithRollingFileIndexMode(indexMode))
	}
	if config.Rolling.DirectWrite {
		options = append(options, rollingfile.WithRollingDirectWrite(true))
	}
	if value := config.Rolling.MaxSize; value != "" {
		size, err := configvalue.ByteSize(value)
		if err != nil {
			return nil, fmt.Errorf("goark-log: appender %q: %w", config.Name, err)
		}
		options = append(options, rollingfile.WithRollingMaxSize(size))
	}
	if value := config.Rolling.Interval; strings.TrimSpace(value) != "" {
		interval, err := configvalue.RollingInterval(value)
		if err != nil {
			return nil, fmt.Errorf("goark-log: appender %q: %w", config.Name, err)
		}
		options = append(options, rollingfile.WithRollingInterval(interval))
	}
	if strings.TrimSpace(config.Rolling.CronSchedule) != "" {
		options = append(options, rollingfile.WithRollingCronSchedule(config.Rolling.CronSchedule))
	}
	if config.Rolling.TimeModulate != nil {
		options = append(options, rollingfile.WithRollingTimeModulate(*config.Rolling.TimeModulate))
	}
	if config.Rolling.OnStartup {
		options = append(options, rollingfile.WithRolloverOnStartup(true))
	}
	if config.Rolling.MaxBackups != nil {
		options = append(options, rollingfile.WithRollingMaxBackups(*config.Rolling.MaxBackups))
	}
	if strings.TrimSpace(config.Rolling.MaxAge) != "" {
		age, err := configvalue.RollingMaxAge(config.Rolling.MaxAge)
		if err != nil {
			return nil, fmt.Errorf("goark-log: appender %q: %w", config.Name, err)
		}
		options = append(options, rollingfile.WithRollingMaxAge(age))
	}
	if config.Rolling.Gzip {
		options = append(options, rollingfile.WithRollingGzip(true))
	}
	if config.Rolling.AsyncActions {
		options = append(options, rollingfile.WithRollingAsyncActions(true))
	}
	if config.Rolling.ActionQueueSize > 0 {
		options = append(options, rollingfile.WithRollingActionQueueSize(config.Rolling.ActionQueueSize))
	}
	if len(config.Rolling.DeleteActions) > 0 {
		actions, err := buildRollingDeleteActions(config.Name, config.Rolling.DeleteActions)
		if err != nil {
			return nil, err
		}
		options = append(options, rollingfile.WithRollingDeleteActions(actions...))
	}
	return rollingfile.NewRollingFileAppender(config.FileName, options...)
}

func rollingFileIndexMode(value string) (rollingfile.RollingFileIndexMode, bool) {
	switch value {
	case "min":
		return rollingfile.RollingFileIndexMin, true
	case "max":
		return rollingfile.RollingFileIndexMax, true
	case "nomax", "no-max", "none":
		return rollingfile.RollingFileIndexNoMax, true
	default:
		return "", false
	}
}

func buildRollingDeleteActions(appenderName string, configs []RollingDeleteBuildConfig) ([]rollingfile.RollingDeleteAction, error) {
	actions := make([]rollingfile.RollingDeleteAction, 0, len(configs))
	for index, actionConfig := range configs {
		action, err := buildRollingDeleteAction(actionConfig)
		if err != nil {
			return nil, fmt.Errorf("goark-log: appender %q rolling delete action %d: %w", appenderName, index, err)
		}
		actions = append(actions, action)
	}
	return actions, nil
}

func buildRollingDeleteAction(config RollingDeleteBuildConfig) (rollingfile.RollingDeleteAction, error) {
	action := rollingfile.RollingDeleteAction{
		BasePath: config.BasePath,
		MaxDepth: config.MaxDepth,
		Glob:     config.Glob,
		MaxCount: config.MaxCount,
	}
	if strings.TrimSpace(config.MaxAge) != "" {
		age, err := configvalue.RollingMaxAge(config.MaxAge)
		if err != nil {
			return rollingfile.RollingDeleteAction{}, err
		}
		action.MaxAge = age
	}
	if strings.TrimSpace(config.MaxSize) != "" {
		size, err := configvalue.ByteSize(config.MaxSize)
		if err != nil {
			return rollingfile.RollingDeleteAction{}, err
		}
		action.MaxSize = size
	}
	return action, nil
}
