package goarklog

import (
	"fmt"
	"strings"
)

func applyAppenderProperty(config *fileConfig, aliases propertyAliases, key string, value string) error {
	id, field, ok := splitPropertyID(key)
	if !ok {
		return nil
	}
	id = aliases.appenderName(id)
	appender := config.Appenders[id]
	if strings.HasPrefix(field, "layout.") {
		if err := applyLayoutProperty(&appender.Layout, strings.TrimPrefix(field, "layout."), value); err != nil {
			return err
		}
		config.Appenders[id] = appender
		return nil
	}
	if strings.HasPrefix(field, "routes.") {
		routeKey := strings.TrimSpace(strings.TrimPrefix(field, "routes."))
		if routeKey == "" {
			return fmt.Errorf("goark-log: properties appender.%s.%s has empty route key", id, field)
		}
		if appender.Routes == nil {
			appender.Routes = make(map[string]string)
		}
		appender.Routes[routeKey] = value
		config.Appenders[id] = appender
		return nil
	}
	if strings.HasPrefix(field, "rewrite.") {
		if err := applyRewriteProperty(&appender.Rewrite, strings.TrimPrefix(field, "rewrite."), value); err != nil {
			return err
		}
		config.Appenders[id] = appender
		return nil
	}
	if strings.HasPrefix(field, "appenderRef.") {
		if err := applyAppenderRefProperty(&appender.AppenderRefs, strings.TrimPrefix(field, "appenderRef."), value); err != nil {
			return err
		}
		config.Appenders[id] = appender
		return nil
	}
	switch field {
	case "name":
		return nil
	case "type":
		appender.Type = value
	case "target":
		appender.Target = value
	case "url":
		appender.URL = value
	case "method":
		appender.Method = value
	case "address":
		appender.Address = value
	case "network":
		appender.Network = value
	case "facility":
		appender.Facility = value
	case "appName", "app-name":
		appender.AppName = value
	case "connectTimeout", "connect-timeout":
		appender.ConnectTimeout = value
	case "writeTimeout", "write-timeout":
		appender.WriteTimeout = value
	case "fileName", "file-name", "path":
		appender.FileName = value
	case "appenderRefs", "appender-refs", "refs":
		appender.AppenderRefs = propertyAppenderRefs(value)
	case "primary", "primary-ref":
		appender.Primary = value
	case "failovers", "failover-refs":
		appender.Failovers = propertyList(value)
	case "routeKey", "route-key", "attrKey", "attr-key":
		appender.RouteKey = value
	case "defaultRoute", "default-route":
		appender.DefaultRoute = value
	case "queueSize", "queue-size":
		parsed, err := parsePropertyInt(value, key)
		if err != nil {
			return err
		}
		appender.QueueSize = parsed
	case "overflowStrategy", "overflow-strategy":
		appender.OverflowStrategy = value
	case "waitStrategy", "wait-strategy":
		appender.WaitStrategy = value
	case "waitRetries", "wait-retries":
		parsed, err := parsePropertyInt(value, key)
		if err != nil {
			return err
		}
		appender.WaitRetries = parsed
	case "sleepTime", "sleep-time":
		appender.SleepTime = value
	case "timeout":
		appender.Timeout = value
	case "bufferSize", "buffer-size":
		appender.BufferSize = value
	case "flushOnWrite", "flush-on-write":
		parsed, err := parsePropertyBool(value, key)
		if err != nil {
			return err
		}
		appender.FlushOnWrite = parsed
	case "append":
		parsed, err := parsePropertyBool(value, key)
		if err != nil {
			return err
		}
		appender.Append = &parsed
	case "createOnDemand", "create-on-demand":
		parsed, err := parsePropertyBool(value, key)
		if err != nil {
			return err
		}
		appender.CreateOnDemand = parsed
	case "filePermissions", "file-permissions":
		appender.FilePermissions = value
	case "filters", "filterRefs", "filter-refs":
		appender.Filters = propertyList(value)
	case "rolling.filePattern", "rolling.file-pattern":
		appender.Rolling.FilePattern = value
	case "rolling.maxSize", "rolling.max-size":
		appender.Rolling.MaxSize = value
	case "rolling.interval":
		appender.Rolling.Interval = value
	case "rolling.cron", "rolling.cronSchedule", "rolling.cron-schedule", "rolling.policies.cron.schedule", "rolling.policies.cronTriggeringPolicy.schedule", "rolling.policies.cron-triggering-policy.schedule":
		appender.Rolling.CronSchedule = value
	case "rolling.strategy.delete.maxCount", "rolling.strategy.delete.max-count":
		parsed, err := parsePropertyInt(value, key)
		if err != nil {
			return err
		}
		appender.Rolling.Strategy.Delete.MaxCount = &parsed
	case "rolling.strategy.delete.maxSize", "rolling.strategy.delete.max-size":
		appender.Rolling.Strategy.Delete.MaxSize = value
	case "rolling.strategy.delete.ifAccumulatedFileCount.exceeds", "rolling.strategy.delete.if-accumulated-file-count.exceeds":
		parsed, err := parsePropertyInt(value, key)
		if err != nil {
			return err
		}
		appender.Rolling.Strategy.Delete.IfAccumulatedFileCount.Exceeds = parsed
	case "rolling.strategy.delete.ifAccumulatedFileSize.exceeds", "rolling.strategy.delete.if-accumulated-file-size.exceeds":
		appender.Rolling.Strategy.Delete.IfAccumulatedFileSize.Exceeds = value
	case "rolling.strategy.type":
		appender.Rolling.Strategy.Type = value
	case "rolling.strategy.fileIndex", "rolling.strategy.file-index":
		appender.Rolling.Strategy.FileIndex = value
	case "rolling.directWrite", "rolling.direct-write", "rolling.strategy.directWrite", "rolling.strategy.direct-write":
		parsed, err := parsePropertyBool(value, key)
		if err != nil {
			return err
		}
		appender.Rolling.DirectWrite = parsed
	}
	config.Appenders[id] = appender
	return nil
}
