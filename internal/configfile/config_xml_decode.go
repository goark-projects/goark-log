package configfile

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"

	"goark.dev/log/internal/configxml"
	"goark.dev/log/internal/textutil"
)

func decodeXMLConfig(reader io.Reader, lookups *LookupResolver) (*fileConfig, error) {
	var config xmlConfig
	decoder := xml.NewDecoder(reader)
	if err := decoder.Decode(&config); err != nil {
		if err == io.EOF {
			return &fileConfig{}, nil
		}
		return nil, err
	}
	file, err := config.fileConfig()
	if err != nil {
		return nil, err
	}
	return finalizeDecodedConfig(file, lookups)
}

func (c xmlConfig) fileConfig() (fileConfig, error) {
	file := fileConfig{
		Status:          c.Status,
		MonitorInterval: c.MonitorInterval,
		Properties:      c.properties(),
		CustomLevels:    c.customLevels(),
		Appenders:       make(map[string]appenderConfig),
		Filters:         make(map[string]filterConfig),
		Loggers:         make(map[string]loggerConfig),
	}
	if err := c.appenders(&file); err != nil {
		return fileConfig{}, err
	}
	if err := c.filters(&file); err != nil {
		return fileConfig{}, err
	}
	if err := c.loggers(&file); err != nil {
		return fileConfig{}, err
	}
	async, err := c.AsyncLogger.config()
	if err != nil {
		return fileConfig{}, err
	}
	file.AsyncLogger = async
	return file, nil
}

func (c xmlConfig) customLevels() map[string]string {
	if len(c.CustomLevels.Levels) == 0 {
		return nil
	}
	levels := make(map[string]string, len(c.CustomLevels.Levels))
	for _, level := range c.CustomLevels.Levels {
		name := strings.TrimSpace(level.Name)
		if name == "" {
			continue
		}
		levels[name] = textutil.FirstNonBlank(level.IntLevel, level.Value, level.Text)
	}
	return levels
}

func (c xmlConfig) properties() map[string]string {
	if len(c.Properties) == 0 {
		return nil
	}
	properties := make(map[string]string, len(c.Properties))
	for _, property := range c.Properties {
		name := strings.TrimSpace(property.Name)
		if name == "" {
			continue
		}
		properties[name] = textutil.FirstNonBlank(property.Value, property.Text)
	}
	return properties
}

func (c xmlConfig) appenders(file *fileConfig) error {
	groups := [][]xmlAppender{
		c.Appenders.Console,
		c.Appenders.File,
		c.Appenders.RollingFile,
		c.Appenders.Async,
		c.Appenders.Failover,
		c.Appenders.Routing,
		c.Appenders.Rewrite,
		c.Appenders.HTTP,
		c.Appenders.Socket,
		c.Appenders.Syslog,
	}
	for _, group := range groups {
		for _, item := range group {
			name, appender, err := item.config()
			if err != nil {
				return err
			}
			file.Appenders[name] = appender
		}
	}
	return nil
}

func (c xmlConfig) filters(file *fileConfig) error {
	groups := []struct {
		kind    string
		filters []xmlFilter
	}{
		{kind: "threshold", filters: c.Filters.Threshold},
		{kind: "level", filters: c.Filters.Level},
		{kind: "levelRange", filters: c.Filters.Range},
		{kind: "regex", filters: c.Filters.Regex},
		{kind: "attr", filters: c.Filters.Attr},
		{kind: "attr", filters: c.Filters.Attribute},
		{kind: "deny", filters: c.Filters.Deny},
		{kind: "denyAll", filters: c.Filters.DenyAll},
		{kind: "composite", filters: c.Filters.Composite},
		{kind: "marker", filters: c.Filters.Marker},
		{kind: "noMarker", filters: c.Filters.NoMarker},
		{kind: "map", filters: c.Filters.Map},
		{kind: "threadContextMap", filters: c.Filters.ThreadContextMap},
		{kind: "threadContextStack", filters: c.Filters.ThreadContextStack},
		{kind: "structuredData", filters: c.Filters.StructuredData},
		{kind: "throwable", filters: c.Filters.Throwable},
		{kind: "stringMatch", filters: c.Filters.StringMatch},
		{kind: "time", filters: c.Filters.Time},
		{kind: "burst", filters: c.Filters.Burst},
		{kind: "dynamicThreshold", filters: c.Filters.DynamicThreshold},
	}
	for _, group := range groups {
		for _, item := range group.filters {
			name := strings.TrimSpace(item.Name)
			if name == "" {
				return fmt.Errorf("goark-log: XML filter name is empty")
			}
			config, err := item.config(group.kind)
			if err != nil {
				return fmt.Errorf("goark-log: XML filter %q: %w", name, err)
			}
			file.Filters[name] = config
		}
	}
	return nil
}

func (c xmlConfig) loggers(file *fileConfig) error {
	if !c.Loggers.Root.empty() {
		root, err := c.Loggers.Root.config(false)
		if err != nil {
			return err
		}
		file.Root = root
	}
	for _, logger := range c.Loggers.Logger {
		config, err := logger.config(true)
		if err != nil {
			return err
		}
		file.Loggers[strings.TrimSpace(logger.Name)] = config
	}
	return nil
}

func (l xmlLogger) config(named bool) (loggerConfig, error) {
	if named && strings.TrimSpace(l.Name) == "" {
		return loggerConfig{}, fmt.Errorf("goark-log: XML logger name is empty")
	}
	appenderRefs, err := xmlAppenderRefs(l.AppenderRefs)
	if err != nil {
		return loggerConfig{}, fmt.Errorf("goark-log: XML logger %q: %w", l.Name, err)
	}
	includeLocation, err := configxml.BoolPointerStrict(l.IncludeLocation, "includeLocation")
	if err != nil {
		return loggerConfig{}, fmt.Errorf("goark-log: XML logger %q: %w", l.Name, err)
	}
	config := loggerConfig{
		Level:           l.Level,
		AppenderRefs:    appenderRefs,
		Filters:         xmlFilterRefs(l.FilterRefs),
		IncludeLocation: includeLocation,
	}
	if strings.TrimSpace(l.Additivity) != "" {
		value, err := strconv.ParseBool(strings.ToLower(strings.TrimSpace(l.Additivity)))
		if err != nil {
			return loggerConfig{}, fmt.Errorf("goark-log: XML logger %q additivity is invalid", l.Name)
		}
		config.Additivity = &value
	}
	return config, nil
}

func (l xmlLogger) empty() bool {
	return strings.TrimSpace(l.Name) == "" &&
		strings.TrimSpace(l.Level) == "" &&
		strings.TrimSpace(l.Additivity) == "" &&
		strings.TrimSpace(l.IncludeLocation) == "" &&
		len(l.AppenderRefs) == 0 &&
		len(l.FilterRefs) == 0
}

func (f xmlFilter) config(kind string) (filterConfig, error) {
	if strings.TrimSpace(f.Type) != "" {
		kind = f.Type
	}
	maxBurst, err := configxml.Int(f.MaxBurst, "maxBurst")
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
	queueSize, err := configxml.Int(c.QueueSize, "queueSize")
	if err != nil {
		return asyncLoggerConfig{}, err
	}
	batchSize, err := configxml.Int(c.BatchSize, "batchSize")
	if err != nil {
		return asyncLoggerConfig{}, err
	}
	waitRetries, err := configxml.Int(c.WaitRetries, "waitRetries")
	if err != nil {
		return asyncLoggerConfig{}, err
	}
	return asyncLoggerConfig{
		Enabled:          configxml.BoolPointer(c.Enabled),
		QueueSize:        queueSize,
		BatchSize:        batchSize,
		OverflowStrategy: c.OverflowStrategy,
		WaitStrategy:     c.WaitStrategy,
		WaitRetries:      waitRetries,
		SleepTime:        c.SleepTime,
		Timeout:          c.Timeout,
		IncludeLocation:  configxml.BoolPointer(c.IncludeLocation),
	}, nil
}

func (a xmlRollingDeleteAction) config() rollingDeleteActionConfig {
	return rollingDeleteActionConfig{
		BasePath: a.BasePath,
		MaxDepth: configxml.IntPointer(a.MaxDepth),
		MaxCount: configxml.IntPointer(a.MaxCount),
		MaxSize:  a.MaxSize,
		Glob:     textutil.FirstNonBlank(a.Glob, a.IfFileName.Glob),
		Age:      textutil.FirstNonBlank(a.Age, a.IfLastModified.Age),
		IfFileName: rollingDeleteFileNameConfig{
			Glob: a.IfFileName.Glob,
		},
		IfLastModified: rollingDeleteLastModifiedConfig{
			Age: a.IfLastModified.Age,
		},
		IfAccumulatedFileCount: rollingDeleteAccumulatedCountConfig{
			Exceeds: configxml.IntValue(a.IfAccumulatedFileCount.Exceeds),
		},
		IfAccumulatedFileSize: rollingDeleteAccumulatedSizeConfig{
			Exceeds: a.IfAccumulatedFileSize.Exceeds,
		},
	}
}

func xmlAppenderType(element string, configured string) string {
	if strings.TrimSpace(configured) != "" {
		return configured
	}
	switch textutil.NormalizeKind(element) {
	case "rollingfile":
		return "rollingFile"
	default:
		return element
	}
}

func xmlConsoleTarget(target string) string {
	switch strings.ToUpper(strings.TrimSpace(target)) {
	case "SYSTEM_OUT", "STDOUT":
		return "stdout"
	case "SYSTEM_ERR", "STDERR":
		return "stderr"
	default:
		return target
	}
}

func xmlAppenderRefs(refs []xmlAppenderRef) (appenderRefs, error) {
	out := make(appenderRefs, 0, len(refs))
	for _, ref := range refs {
		includeLocation, err := configxml.BoolPointerStrict(ref.IncludeLocation, "includeLocation")
		if err != nil {
			return nil, fmt.Errorf("AppenderRef %q: %w", ref.Ref, err)
		}
		out = append(out, appenderRefConfig{
			Ref:             ref.Ref,
			Level:           ref.Level,
			IncludeLocation: includeLocation,
			FilterRefs:      xmlFilterRefs(ref.FilterRefs),
		})
	}
	return out, nil
}

func xmlFilterRefsFromAppenderRefs(refs []xmlAppenderRef) []string {
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		out = append(out, strings.TrimSpace(ref.Ref))
	}
	return out
}

func xmlRoutes(routes []xmlRoute) map[string]string {
	if len(routes) == 0 {
		return nil
	}
	out := make(map[string]string, len(routes))
	for _, route := range routes {
		key := strings.TrimSpace(route.Key)
		if key == "" {
			continue
		}
		out[key] = textutil.FirstNonBlank(route.Ref, route.AppenderRef.Ref)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func xmlFilterRefs(refs []xmlFilterRef) []string {
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		out = append(out, strings.TrimSpace(ref.Ref))
	}
	return out
}

func xmlKeyValuePairMap(pairs []xmlKeyValuePair) map[string]string {
	if len(pairs) == 0 {
		return nil
	}
	out := make(map[string]string, len(pairs))
	for _, pair := range pairs {
		key := strings.TrimSpace(pair.Key)
		if key != "" {
			out[key] = pair.Value
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func xmlKeyValuePairs(pairs []xmlKeyValuePair) []keyValuePairConfig {
	out := make([]keyValuePairConfig, 0, len(pairs))
	for _, pair := range pairs {
		out = append(out, keyValuePairConfig{
			Key:   pair.Key,
			Value: pair.Value,
		})
	}
	return out
}

func xmlRemoveAttrs(values []xmlRemoveAttr) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if key := textutil.FirstNonBlank(value.Key, value.Name); key != "" {
			out = append(out, key)
		}
	}
	return out
}
