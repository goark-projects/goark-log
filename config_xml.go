package goarklog

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type xmlConfig struct {
	XMLName         xml.Name
	Status          string         `xml:"status,attr"`
	MonitorInterval string         `xml:"monitorInterval,attr"`
	Properties      []xmlProperty  `xml:"Properties>Property"`
	Appenders       xmlAppenders   `xml:"Appenders"`
	Filters         xmlFilters     `xml:"Filters"`
	AsyncLogger     xmlAsyncLogger `xml:"AsyncLogger"`
	Loggers         xmlLoggers     `xml:"Loggers"`
}

type xmlProperty struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
	Text  string `xml:",chardata"`
}

type xmlAppenders struct {
	Console     []xmlAppender `xml:"Console"`
	File        []xmlAppender `xml:"File"`
	RollingFile []xmlAppender `xml:"RollingFile"`
	Async       []xmlAppender `xml:"Async"`
}

type xmlAppender struct {
	XMLName          xml.Name
	Name             string             `xml:"name,attr"`
	Type             string             `xml:"type,attr"`
	Target           string             `xml:"target,attr"`
	FileName         string             `xml:"fileName,attr"`
	FilePattern      string             `xml:"filePattern,attr"`
	QueueSize        string             `xml:"queueSize,attr"`
	OverflowStrategy string             `xml:"overflowStrategy,attr"`
	WaitStrategy     string             `xml:"waitStrategy,attr"`
	BufferSize       string             `xml:"bufferSize,attr"`
	FlushOnWrite     string             `xml:"flushOnWrite,attr"`
	PatternLayout    xmlLayout          `xml:"PatternLayout"`
	TextLayout       xmlLayout          `xml:"TextLayout"`
	JsonLayout       xmlLayout          `xml:"JsonLayout"`
	JSONLayout       xmlLayout          `xml:"JSONLayout"`
	Layout           xmlLayout          `xml:"Layout"`
	AppenderRefs     []xmlAppenderRef   `xml:"AppenderRef"`
	FilterRefs       []xmlFilterRef     `xml:"FilterRef"`
	Policies         xmlRollingPolicies `xml:"Policies"`
	Strategy         xmlRollingStrategy `xml:"DefaultRolloverStrategy"`
}

type xmlLayout struct {
	XMLName xml.Name
	Type    string `xml:"type,attr"`
	Pattern string `xml:"pattern,attr"`
}

type xmlAppenderRef struct {
	Ref        string         `xml:"ref,attr"`
	Level      string         `xml:"level,attr"`
	FilterRefs []xmlFilterRef `xml:"FilterRef"`
}

type xmlFilterRef struct {
	Ref string `xml:"ref,attr"`
}

type xmlRollingPolicies struct {
	Size    xmlRollingSizePolicy `xml:"SizeBasedTriggeringPolicy"`
	Time    xmlRollingTimePolicy `xml:"TimeBasedTriggeringPolicy"`
	Startup xmlRollingStartup    `xml:"OnStartupTriggeringPolicy"`
}

type xmlRollingSizePolicy struct {
	Size string `xml:"size,attr"`
}

type xmlRollingTimePolicy struct {
	Interval string `xml:"interval,attr"`
	Modulate string `xml:"modulate,attr"`
}

type xmlRollingStartup struct {
	Enabled string `xml:"enabled,attr"`
}

type xmlRollingStrategy struct {
	Max string `xml:"max,attr"`
}

type xmlFilters struct {
	Threshold []xmlFilter `xml:"ThresholdFilter"`
	Level     []xmlFilter `xml:"LevelFilter"`
	Range     []xmlFilter `xml:"LevelRangeFilter"`
	Regex     []xmlFilter `xml:"RegexFilter"`
	Attr      []xmlFilter `xml:"AttrFilter"`
	Attribute []xmlFilter `xml:"AttributeFilter"`
}

type xmlFilter struct {
	Name       string `xml:"name,attr"`
	Type       string `xml:"type,attr"`
	Level      string `xml:"level,attr"`
	MinLevel   string `xml:"minLevel,attr"`
	MaxLevel   string `xml:"maxLevel,attr"`
	Field      string `xml:"field,attr"`
	Key        string `xml:"key,attr"`
	Value      string `xml:"value,attr"`
	Pattern    string `xml:"pattern,attr"`
	OnMatch    string `xml:"onMatch,attr"`
	OnMismatch string `xml:"onMismatch,attr"`
}

type xmlAsyncLogger struct {
	Enabled          string `xml:"enabled,attr"`
	QueueSize        string `xml:"queueSize,attr"`
	BatchSize        string `xml:"batchSize,attr"`
	OverflowStrategy string `xml:"overflowStrategy,attr"`
	WaitStrategy     string `xml:"waitStrategy,attr"`
}

type xmlLoggers struct {
	Root   xmlLogger   `xml:"Root"`
	Logger []xmlLogger `xml:"Logger"`
}

type xmlLogger struct {
	Name         string           `xml:"name,attr"`
	Level        string           `xml:"level,attr"`
	Additivity   string           `xml:"additivity,attr"`
	AppenderRefs []xmlAppenderRef `xml:"AppenderRef"`
	FilterRefs   []xmlFilterRef   `xml:"FilterRef"`
}

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
		properties[name] = firstNonBlank(property.Value, property.Text)
	}
	return properties
}

func (c xmlConfig) appenders(file *fileConfig) error {
	groups := [][]xmlAppender{c.Appenders.Console, c.Appenders.File, c.Appenders.RollingFile, c.Appenders.Async}
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
	}
	for _, group := range groups {
		for _, item := range group.filters {
			name := strings.TrimSpace(item.Name)
			if name == "" {
				return fmt.Errorf("goark-log: XML filter name is empty")
			}
			file.Filters[name] = item.config(group.kind)
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

func (a xmlAppender) config() (string, appenderConfig, error) {
	name := strings.TrimSpace(a.Name)
	if name == "" {
		return "", appenderConfig{}, fmt.Errorf("goark-log: XML appender name is empty")
	}
	queueSize, err := parseXMLInt(a.QueueSize, "queueSize")
	if err != nil {
		return "", appenderConfig{}, fmt.Errorf("goark-log: XML appender %q: %w", name, err)
	}
	flushOnWrite, err := parseXMLBool(a.FlushOnWrite, "flushOnWrite")
	if err != nil {
		return "", appenderConfig{}, fmt.Errorf("goark-log: XML appender %q: %w", name, err)
	}
	config := appenderConfig{
		Type:             xmlAppenderType(a.XMLName.Local, a.Type),
		Target:           xmlConsoleTarget(a.Target),
		FileName:         a.FileName,
		Layout:           a.layout(),
		AppenderRefs:     xmlAppenderRefs(a.AppenderRefs),
		QueueSize:        queueSize,
		OverflowStrategy: a.OverflowStrategy,
		WaitStrategy:     a.WaitStrategy,
		BufferSize:       a.BufferSize,
		FlushOnWrite:     flushOnWrite,
		Filters:          xmlFilterRefs(a.FilterRefs),
		Rolling: rollingConfig{
			FilePattern: a.FilePattern,
			Policies: rollingPoliciesConfig{
				SizeBasedTriggeringPolicy: rollingSizePolicyConfig{
					Size: a.Policies.Size.Size,
				},
				TimeBasedTriggeringPolicy: rollingTimePolicyConfig{
					Interval: a.Policies.Time.Interval,
					Modulate: parseXMLBoolPointer(a.Policies.Time.Modulate),
				},
				OnStartupTriggeringPolicy: rollingStartupPolicyConfig{
					Enabled: parseXMLBoolPointer(a.Policies.Startup.Enabled),
				},
			},
			Strategy: rollingStrategyConfig{
				Max: parseXMLIntPointer(a.Strategy.Max),
			},
		},
	}
	return name, config, nil
}

func (a xmlAppender) layout() layoutConfig {
	for _, layout := range []xmlLayout{a.PatternLayout, a.TextLayout, a.JsonLayout, a.JSONLayout, a.Layout} {
		if layout.XMLName.Local == "" && strings.TrimSpace(layout.Type) == "" && strings.TrimSpace(layout.Pattern) == "" {
			continue
		}
		return layout.config()
	}
	return layoutConfig{}
}

func (l xmlLayout) config() layoutConfig {
	kind := firstNonBlank(l.Type, l.XMLName.Local)
	switch normalizeKind(kind) {
	case "", "patternlayout", "pattern":
		kind = "pattern"
	case "textlayout", "text":
		kind = "text"
	case "jsonlayout", "json":
		kind = "json"
	default:
		kind = l.Type
	}
	return layoutConfig{Type: kind, Pattern: l.Pattern}
}

func (f xmlFilter) config(kind string) filterConfig {
	if strings.TrimSpace(f.Type) != "" {
		kind = f.Type
	}
	return filterConfig{
		Type:       kind,
		Level:      f.Level,
		MinLevel:   f.MinLevel,
		MaxLevel:   f.MaxLevel,
		Field:      f.Field,
		Key:        f.Key,
		Value:      f.Value,
		Pattern:    f.Pattern,
		OnMatch:    f.OnMatch,
		OnMismatch: f.OnMismatch,
	}
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
	return asyncLoggerConfig{
		Enabled:          parseXMLBoolPointer(c.Enabled),
		QueueSize:        queueSize,
		BatchSize:        batchSize,
		OverflowStrategy: c.OverflowStrategy,
		WaitStrategy:     c.WaitStrategy,
	}, nil
}

func (l xmlLogger) config(named bool) (loggerConfig, error) {
	if named && strings.TrimSpace(l.Name) == "" {
		return loggerConfig{}, fmt.Errorf("goark-log: XML logger name is empty")
	}
	config := loggerConfig{
		Level:        l.Level,
		AppenderRefs: xmlAppenderRefs(l.AppenderRefs),
		Filters:      xmlFilterRefs(l.FilterRefs),
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
		len(l.AppenderRefs) == 0 &&
		len(l.FilterRefs) == 0
}

func xmlAppenderType(element string, configured string) string {
	if strings.TrimSpace(configured) != "" {
		return configured
	}
	switch normalizeKind(element) {
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

func xmlAppenderRefs(refs []xmlAppenderRef) appenderRefs {
	out := make(appenderRefs, 0, len(refs))
	for _, ref := range refs {
		out = append(out, appenderRefConfig{
			Ref:        ref.Ref,
			Level:      ref.Level,
			FilterRefs: xmlFilterRefs(ref.FilterRefs),
		})
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

func parseXMLInt(value string, field string) (int, error) {
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("%s is invalid", field)
	}
	return parsed, nil
}

func parseXMLIntPointer(value string) *int {
	parsed, err := parseXMLInt(value, "")
	if err != nil || parsed == 0 {
		return nil
	}
	return &parsed
}

func parseXMLBool(value string, field string) (bool, error) {
	if strings.TrimSpace(value) == "" {
		return false, nil
	}
	parsed, err := strconv.ParseBool(strings.ToLower(strings.TrimSpace(value)))
	if err != nil {
		return false, fmt.Errorf("%s is invalid", field)
	}
	return parsed, nil
}

func parseXMLBoolPointer(value string) *bool {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parsed, err := parseXMLBool(value, "")
	if err != nil {
		return nil
	}
	return &parsed
}
