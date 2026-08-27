package goarklog

import (
	"fmt"
	"strings"

	"goark.dev/log/internal/configprops"
	"goark.dev/log/internal/textutil"
)

func applyAppenderProperty(config *fileConfig, aliases configprops.Aliases, key string, value string) error {
	id, field, ok := configprops.SplitID(key)
	if !ok {
		return nil
	}
	id = aliases.AppenderName(id)
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
		appender.Failovers = configprops.List(value)
	case "routeKey", "route-key", "attrKey", "attr-key":
		appender.RouteKey = value
	case "defaultRoute", "default-route":
		appender.DefaultRoute = value
	case "queueSize", "queue-size":
		parsed, err := configprops.Int(value, key)
		if err != nil {
			return err
		}
		appender.QueueSize = parsed
	case "batchSize", "batch-size":
		parsed, err := configprops.Int(value, key)
		if err != nil {
			return err
		}
		appender.BatchSize = parsed
	case "overflowStrategy", "overflow-strategy":
		appender.OverflowStrategy = value
	case "waitStrategy", "wait-strategy":
		appender.WaitStrategy = value
	case "waitRetries", "wait-retries":
		parsed, err := configprops.Int(value, key)
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
		parsed, err := configprops.Bool(value, key)
		if err != nil {
			return err
		}
		appender.FlushOnWrite = parsed
	case "append":
		parsed, err := configprops.Bool(value, key)
		if err != nil {
			return err
		}
		appender.Append = &parsed
	case "createOnDemand", "create-on-demand":
		parsed, err := configprops.Bool(value, key)
		if err != nil {
			return err
		}
		appender.CreateOnDemand = parsed
	case "filePermissions", "file-permissions":
		appender.FilePermissions = value
	case "filters", "filterRefs", "filter-refs":
		appender.Filters = configprops.List(value)
	case "rolling.filePattern", "rolling.file-pattern":
		appender.Rolling.FilePattern = value
	case "rolling.maxSize", "rolling.max-size":
		appender.Rolling.MaxSize = value
	case "rolling.interval":
		appender.Rolling.Interval = value
	case "rolling.cron", "rolling.cronSchedule", "rolling.cron-schedule", "rolling.policies.cron.schedule", "rolling.policies.cronTriggeringPolicy.schedule", "rolling.policies.cron-triggering-policy.schedule":
		appender.Rolling.CronSchedule = value
	case "rolling.strategy.delete.maxCount", "rolling.strategy.delete.max-count":
		parsed, err := configprops.Int(value, key)
		if err != nil {
			return err
		}
		appender.Rolling.Strategy.Delete.MaxCount = &parsed
	case "rolling.strategy.delete.maxSize", "rolling.strategy.delete.max-size":
		appender.Rolling.Strategy.Delete.MaxSize = value
	case "rolling.strategy.delete.ifAccumulatedFileCount.exceeds", "rolling.strategy.delete.if-accumulated-file-count.exceeds":
		parsed, err := configprops.Int(value, key)
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
		parsed, err := configprops.Bool(value, key)
		if err != nil {
			return err
		}
		appender.Rolling.DirectWrite = parsed
	}
	config.Appenders[id] = appender
	return nil
}

func applyFilterProperty(config *fileConfig, key string, value string) error {
	id, field, ok := configprops.SplitID(key)
	if !ok {
		return nil
	}
	filter := config.Filters[id]
	switch {
	case field == "type":
		filter.Type = value
	case field == "level":
		filter.Level = value
	case field == "minLevel" || field == "min-level":
		filter.MinLevel = value
	case field == "maxLevel" || field == "max-level":
		filter.MaxLevel = value
	case field == "marker":
		filter.Marker = value
	case field == "text":
		filter.Text = value
	case field == "operator":
		filter.Operator = value
	case field == "start":
		filter.Start = value
	case field == "end":
		filter.End = value
	case field == "timezone":
		filter.Timezone = value
	case field == "rate":
		filter.Rate = value
	case field == "maxBurst" || field == "max-burst":
		parsed, err := configprops.Int(value, key)
		if err != nil {
			return err
		}
		filter.MaxBurst = parsed
	case field == "field":
		filter.Field = value
	case field == "key":
		filter.Key = value
	case field == "value":
		filter.Value = value
	case strings.HasPrefix(field, "values."):
		mapKey := strings.TrimSpace(strings.TrimPrefix(field, "values."))
		if mapKey == "" {
			return fmt.Errorf("goark-log: properties filter.%s.%s has empty values key", id, field)
		}
		if filter.Values == nil {
			filter.Values = make(map[string]string)
		}
		filter.Values[mapKey] = value
	case strings.HasPrefix(field, "thresholds."):
		mapKey := strings.TrimSpace(strings.TrimPrefix(field, "thresholds."))
		if mapKey == "" {
			return fmt.Errorf("goark-log: properties filter.%s.%s has empty thresholds key", id, field)
		}
		if filter.Thresholds == nil {
			filter.Thresholds = make(map[string]string)
		}
		filter.Thresholds[mapKey] = value
	case field == "filters" || field == "filterRefs" || field == "filter-refs":
		filter.FilterRefs = configprops.List(value)
	case field == "defaultThreshold" || field == "default-threshold":
		filter.DefaultThreshold = value
	case field == "pattern":
		filter.Pattern = value
	case field == "onMatch" || field == "on-match":
		filter.OnMatch = value
	case field == "onMismatch" || field == "on-mismatch":
		filter.OnMismatch = value
	}
	config.Filters[id] = filter
	return nil
}

type propertyFilterPair struct {
	key      string
	value    string
	hasKey   bool
	hasValue bool
}

func applyFilterKeyValuePairs(config *fileConfig, values map[string]string) error {
	pairsByFilter := make(map[string]map[string]propertyFilterPair)
	for key, value := range values {
		filterID, pairID, field, ok := configprops.SplitFilterPairKey(key)
		if !ok {
			continue
		}
		pairs := pairsByFilter[filterID]
		if pairs == nil {
			pairs = make(map[string]propertyFilterPair)
			pairsByFilter[filterID] = pairs
		}
		pair := pairs[pairID]
		switch field {
		case "key":
			pair.key = value
			pair.hasKey = true
		case "value":
			pair.value = value
			pair.hasValue = true
		}
		pairs[pairID] = pair
	}
	filterIDs := textutil.SortedKeys(pairsByFilter)
	for _, filterID := range filterIDs {
		filter := config.Filters[filterID]
		pairIDs := textutil.SortedKeys(pairsByFilter[filterID])
		for _, pairID := range pairIDs {
			pair := pairsByFilter[filterID][pairID]
			if !pair.hasKey && !pair.hasValue {
				continue
			}
			if !pair.hasKey || strings.TrimSpace(pair.key) == "" || !pair.hasValue {
				return fmt.Errorf("goark-log: properties filter.%s.%s requires key and value", filterID, pairID)
			}
			filter.KeyValuePair = append(filter.KeyValuePair, keyValuePairConfig{
				Key:   pair.key,
				Value: pair.value,
			})
		}
		config.Filters[filterID] = filter
	}
	return nil
}
