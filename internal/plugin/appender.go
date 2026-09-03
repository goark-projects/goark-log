package plugin

import (
	"fmt"
	"os"
	"strings"

	"goark.dev/log/internal/asyncappender"
	"goark.dev/log/internal/asyncruntime"
	"goark.dev/log/internal/configvalue"
	"goark.dev/log/internal/delegating"
	"goark.dev/log/internal/fileappender"
	"goark.dev/log/internal/jsonappender"
	"goark.dev/log/internal/logfile"
	internalrouter "goark.dev/log/internal/router"
)

func buildConsolePlugin(config AppenderBuildConfig) (internalrouter.Appender, error) {
	target := strings.ToLower(strings.TrimSpace(config.Target))
	switch target {
	case "", "stdout":
		return fileappender.NewConsoleAppender(
			fileappender.WithConsoleName(config.Name),
			fileappender.WithConsoleLayout(config.Layout),
			fileappender.WithConsoleWriter(os.Stdout),
		), nil
	case "stderr":
		return fileappender.NewConsoleAppender(
			fileappender.WithConsoleName(config.Name),
			fileappender.WithConsoleLayout(config.Layout),
			fileappender.WithConsoleWriter(os.Stderr),
		), nil
	default:
		return nil, fmt.Errorf("goark-log: appender %q console target %q is invalid", config.Name, config.Target)
	}
}

func buildFilePlugin(config AppenderBuildConfig) (internalrouter.Appender, error) {
	if config.FileName == "" {
		return nil, fmt.Errorf("goark-log: appender %q fileName is empty", config.Name)
	}
	options := []fileappender.FileOption{
		fileappender.WithFileName(config.Name),
		fileappender.WithFileLayout(config.Layout),
	}
	if strings.TrimSpace(config.BufferSize) != "" {
		size, err := configvalue.ByteSize(config.BufferSize)
		if err != nil {
			return nil, fmt.Errorf("goark-log: appender %q: %w", config.Name, err)
		}
		options = append(options, fileappender.WithFileBufferSize(int(size)))
	}
	if config.FlushOnWrite {
		options = append(options, fileappender.WithFileFlushOnWrite(true))
	}
	if config.Append != nil {
		options = append(options, fileappender.WithFileAppend(*config.Append))
	}
	if config.CreateOnDemand {
		options = append(options, fileappender.WithFileCreateOnDemand(true))
	}
	if strings.TrimSpace(config.FilePermissions) != "" {
		permissions, err := logfile.ParsePermissions(config.FilePermissions)
		if err != nil {
			return nil, fmt.Errorf("goark-log: appender %q: %w", config.Name, err)
		}
		options = append(options, fileappender.WithFilePermissions(permissions))
	}
	return fileappender.NewFileAppender(config.FileName, options...)
}

func buildJSONPlugin(config AppenderBuildConfig) (internalrouter.Appender, error) {
	if strings.TrimSpace(config.FileName) != "" {
		options := []jsonappender.Option{
			jsonappender.WithName(config.Name),
		}
		if strings.TrimSpace(config.BufferSize) != "" {
			size, err := configvalue.ByteSize(config.BufferSize)
			if err != nil {
				return nil, fmt.Errorf("goark-log: appender %q: %w", config.Name, err)
			}
			options = append(options, jsonappender.WithBufferSize(int(size)))
		}
		if config.FlushOnWrite {
			options = append(options, jsonappender.WithFlushOnWrite(true))
		}
		return jsonappender.NewFile(config.FileName, options...)
	}
	switch strings.ToLower(strings.TrimSpace(config.Target)) {
	case "", "stdout":
		return jsonappender.New(jsonappender.WithName(config.Name), jsonappender.WithWriter(os.Stdout)), nil
	case "stderr":
		return jsonappender.New(jsonappender.WithName(config.Name), jsonappender.WithWriter(os.Stderr)), nil
	case "file":
		return nil, fmt.Errorf("goark-log: appender %q JSON target file requires fileName", config.Name)
	default:
		return nil, fmt.Errorf("goark-log: appender %q JSON target %q is invalid", config.Name, config.Target)
	}
}

func buildAsyncPlugin(config AppenderBuildConfig) (internalrouter.Appender, error) {
	if len(config.Delegates) == 0 {
		return nil, fmt.Errorf("goark-log: async appender %q requires appenderRefs", config.Name)
	}
	strategy, err := asyncruntime.ParseOverflowStrategy(config.OverflowStrategy)
	if err != nil {
		return nil, fmt.Errorf("goark-log: async appender %q: %w", config.Name, err)
	}
	options := []asyncappender.Option{
		asyncappender.WithName(config.Name),
		asyncappender.WithOverflowStrategy(strategy),
	}
	if err := asyncruntime.ValidateWaitOptions(config.WaitOptions); err != nil {
		return nil, fmt.Errorf("goark-log: async appender %q: %w", config.Name, err)
	}
	options = append(options, asyncappender.WithWaitOptions(config.WaitOptions))
	waitStrategy, err := asyncruntime.ParseWaitStrategy(config.WaitStrategy)
	if err != nil {
		return nil, fmt.Errorf("goark-log: async appender %q: %w", config.Name, err)
	}
	options = append(options, asyncappender.WithWaitStrategy(waitStrategy))
	if config.QueueSize != 0 {
		options = append(options, asyncappender.WithQueueSize(config.QueueSize))
	}
	if config.BatchSize != 0 {
		options = append(options, asyncappender.WithBatchSize(config.BatchSize))
	}
	return asyncappender.New(asyncAppenderSinks(config.Delegates), options...)
}

func buildFailoverPlugin(config AppenderBuildConfig) (internalrouter.Appender, error) {
	if len(config.Delegates) < 2 {
		return nil, fmt.Errorf("goark-log: failover appender %q requires primary and failovers", config.Name)
	}
	return delegating.NewFailoverAppender(config.Delegates[0], delegatingAppenders(config.Delegates[1:]),
		delegating.WithFailoverName(config.Name),
		delegating.WithFailoverCloseChildren(false),
	)
}

func buildRoutingPlugin(config AppenderBuildConfig) (internalrouter.Appender, error) {
	options := []delegating.RoutingOption{
		delegating.WithRoutingName(config.Name),
		delegating.WithRoutingCloseChildren(false),
	}
	if strings.TrimSpace(config.RouteKey) != "" {
		options = append(options, delegating.WithRoutingAttrKey(config.RouteKey))
	}
	if config.DefaultRoute != nil {
		options = append(options, delegating.WithRoutingDefault(config.DefaultRoute))
	}
	return delegating.NewRoutingAppender(delegatingAppenderMap(config.Routes), options...)
}

func buildRewritePlugin(config AppenderBuildConfig) (internalrouter.Appender, error) {
	if len(config.Delegates) != 1 {
		return nil, fmt.Errorf("goark-log: rewrite appender %q requires exactly one appenderRef", config.Name)
	}
	return delegating.NewRewriteAppender(config.Delegates[0], newAttributeRewritePolicy(config.Rewrite),
		delegating.WithRewriteName(config.Name),
		delegating.WithRewriteCloseDelegate(false),
	)
}

func asyncAppenderSinks(appenders []internalrouter.Appender) []asyncappender.Sink {
	if len(appenders) == 0 {
		return nil
	}
	converted := make([]asyncappender.Sink, 0, len(appenders))
	for _, appender := range appenders {
		converted = append(converted, appender)
	}
	return converted
}

func delegatingAppenders(appenders []internalrouter.Appender) []delegating.Appender {
	if len(appenders) == 0 {
		return nil
	}
	converted := make([]delegating.Appender, 0, len(appenders))
	for _, appender := range appenders {
		converted = append(converted, appender)
	}
	return converted
}

func delegatingAppenderMap(routes map[string]internalrouter.Appender) map[string]delegating.Appender {
	if len(routes) == 0 {
		return nil
	}
	converted := make(map[string]delegating.Appender, len(routes))
	for key, appender := range routes {
		converted[key] = appender
	}
	return converted
}
