package goarklog

import "strings"

func (f xmlFilter) config(kind string) (filterConfig, error) {
	if strings.TrimSpace(f.Type) != "" {
		kind = f.Type
	}
	maxBurst, err := parseXMLInt(f.MaxBurst, "maxBurst")
	if err != nil {
		return filterConfig{}, err
	}
	return filterConfig{
		Type:             kind,
		Level:            f.Level,
		MinLevel:         f.MinLevel,
		MaxLevel:         f.MaxLevel,
		Marker:           f.Marker,
		Text:             f.Text,
		Operator:         f.Operator,
		Start:            f.Start,
		End:              f.End,
		Timezone:         f.Timezone,
		Rate:             f.Rate,
		MaxBurst:         maxBurst,
		Field:            f.Field,
		Key:              f.Key,
		Value:            f.Value,
		DefaultThreshold: f.DefaultThreshold,
		Pattern:          f.Pattern,
		OnMatch:          f.OnMatch,
		OnMismatch:       f.OnMismatch,
		FilterRefs:       xmlFilterRefs(f.FilterRefs),
		KeyValuePair:     xmlKeyValuePairs(f.KeyValuePair),
	}, nil
}

func (c xmlAsyncLogger) config() (asyncLoggerConfig, error) {
	queueSize, err := parseXMLInt(c.QueueSize, "queueSize")
	if err != nil {
		return asyncLoggerConfig{}, err
	}
	batchSize, err := parseXMLInt(c.BatchSize, "batchSize")
	if err != nil {
		return asyncLoggerConfig{}, err
	}
	waitRetries, err := parseXMLInt(c.WaitRetries, "waitRetries")
	if err != nil {
		return asyncLoggerConfig{}, err
	}
	return asyncLoggerConfig{
		Enabled:          parseXMLBoolPointer(c.Enabled),
		QueueSize:        queueSize,
		BatchSize:        batchSize,
		OverflowStrategy: c.OverflowStrategy,
		WaitStrategy:     c.WaitStrategy,
		WaitRetries:      waitRetries,
		SleepTime:        c.SleepTime,
		Timeout:          c.Timeout,
		IncludeLocation:  parseXMLBoolPointer(c.IncludeLocation),
	}, nil
}

func (a xmlRollingDeleteAction) config() rollingDeleteActionConfig {
	return rollingDeleteActionConfig{
		BasePath: a.BasePath,
		MaxDepth: parseXMLIntPointer(a.MaxDepth),
		MaxCount: parseXMLIntPointer(a.MaxCount),
		MaxSize:  a.MaxSize,
		Glob:     firstNonBlank(a.Glob, a.IfFileName.Glob),
		Age:      firstNonBlank(a.Age, a.IfLastModified.Age),
		IfFileName: rollingDeleteFileNameConfig{
			Glob: a.IfFileName.Glob,
		},
		IfLastModified: rollingDeleteLastModifiedConfig{
			Age: a.IfLastModified.Age,
		},
		IfAccumulatedFileCount: rollingDeleteAccumulatedCountConfig{
			Exceeds: parseXMLIntValue(a.IfAccumulatedFileCount.Exceeds),
		},
		IfAccumulatedFileSize: rollingDeleteAccumulatedSizeConfig{
			Exceeds: a.IfAccumulatedFileSize.Exceeds,
		},
	}
}
