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
	Status          string          `xml:"status,attr"`
	MonitorInterval string          `xml:"monitorInterval,attr"`
	Properties      []xmlProperty   `xml:"Properties>Property"`
	CustomLevels    xmlCustomLevels `xml:"CustomLevels"`
	Appenders       xmlAppenders    `xml:"Appenders"`
	Filters         xmlFilters      `xml:"Filters"`
	AsyncLogger     xmlAsyncLogger  `xml:"AsyncLogger"`
	Loggers         xmlLoggers      `xml:"Loggers"`
}

type xmlProperty struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
	Text  string `xml:",chardata"`
}

type xmlCustomLevels struct {
	Levels []xmlCustomLevel `xml:"CustomLevel"`
}

type xmlCustomLevel struct {
	Name     string `xml:"name,attr"`
	Value    string `xml:"value,attr"`
	IntLevel string `xml:"intLevel,attr"`
	Text     string `xml:",chardata"`
}

type xmlAppenders struct {
	Console     []xmlAppender `xml:"Console"`
	File        []xmlAppender `xml:"File"`
	RollingFile []xmlAppender `xml:"RollingFile"`
	Async       []xmlAppender `xml:"Async"`
	Failover    []xmlAppender `xml:"Failover"`
	Routing     []xmlAppender `xml:"Routing"`
	Rewrite     []xmlAppender `xml:"Rewrite"`
	HTTP        []xmlAppender `xml:"Http"`
	Socket      []xmlAppender `xml:"Socket"`
	Syslog      []xmlAppender `xml:"Syslog"`
}

type xmlAppender struct {
	XMLName          xml.Name
	Name             string             `xml:"name,attr"`
	Type             string             `xml:"type,attr"`
	Target           string             `xml:"target,attr"`
	URL              string             `xml:"url,attr"`
	Method           string             `xml:"method,attr"`
	Address          string             `xml:"address,attr"`
	Network          string             `xml:"network,attr"`
	Facility         string             `xml:"facility,attr"`
	AppName          string             `xml:"appName,attr"`
	ConnectTimeout   string             `xml:"connectTimeout,attr"`
	WriteTimeout     string             `xml:"writeTimeout,attr"`
	FileName         string             `xml:"fileName,attr"`
	FilePattern      string             `xml:"filePattern,attr"`
	Primary          string             `xml:"primary,attr"`
	RouteKey         string             `xml:"routeKey,attr"`
	DefaultRoute     string             `xml:"defaultRoute,attr"`
	QueueSize        string             `xml:"queueSize,attr"`
	OverflowStrategy string             `xml:"overflowStrategy,attr"`
	WaitStrategy     string             `xml:"waitStrategy,attr"`
	WaitRetries      string             `xml:"waitRetries,attr"`
	SleepTime        string             `xml:"sleepTime,attr"`
	Timeout          string             `xml:"timeout,attr"`
	BufferSize       string             `xml:"bufferSize,attr"`
	FlushOnWrite     string             `xml:"flushOnWrite,attr"`
	Append           string             `xml:"append,attr"`
	CreateOnDemand   string             `xml:"createOnDemand,attr"`
	FilePermissions  string             `xml:"filePermissions,attr"`
	PatternLayout    xmlLayout          `xml:"PatternLayout"`
	TextLayout       xmlLayout          `xml:"TextLayout"`
	JsonLayout       xmlLayout          `xml:"JsonLayout"`
	JSONLayout       xmlLayout          `xml:"JSONLayout"`
	JsonTemplate     xmlLayout          `xml:"JsonTemplateLayout"`
	XmlLayout        xmlLayout          `xml:"XmlLayout"`
	XMLLayout        xmlLayout          `xml:"XMLLayout"`
	CsvLayout        xmlLayout          `xml:"CsvLayout"`
	CSVLayout        xmlLayout          `xml:"CSVLayout"`
	GelfLayout       xmlLayout          `xml:"GelfLayout"`
	GELFLayout       xmlLayout          `xml:"GELFLayout"`
	Rfc5424Layout    xmlLayout          `xml:"Rfc5424Layout"`
	RFC5424Layout    xmlLayout          `xml:"RFC5424Layout"`
	SyslogLayout     xmlLayout          `xml:"SyslogLayout"`
	YamlLayout       xmlLayout          `xml:"YamlLayout"`
	YAMLLayout       xmlLayout          `xml:"YAMLLayout"`
	HtmlLayout       xmlLayout          `xml:"HtmlLayout"`
	HTMLLayout       xmlLayout          `xml:"HTMLLayout"`
	Layout           xmlLayout          `xml:"Layout"`
	AppenderRefs     []xmlAppenderRef   `xml:"AppenderRef"`
	Failovers        []xmlAppenderRef   `xml:"Failovers>AppenderRef"`
	Routes           []xmlRoute         `xml:"Route"`
	KeyValuePair     []xmlKeyValuePair  `xml:"KeyValuePair"`
	Remove           []xmlRemoveAttr    `xml:"Remove"`
	FilterRefs       []xmlFilterRef     `xml:"FilterRef"`
	Policies         xmlRollingPolicies `xml:"Policies"`
	Strategy         xmlRollingStrategy `xml:"DefaultRolloverStrategy"`
	DirectStrategy   xmlRollingStrategy `xml:"DirectWriteRolloverStrategy"`
}

type xmlLayout struct {
	XMLName              xml.Name
	Type                 string `xml:"type,attr"`
	Pattern              string `xml:"pattern,attr"`
	Template             string `xml:"eventTemplate,attr"`
	TemplateURI          string `xml:"eventTemplateUri,attr"`
	Compact              string `xml:"compact,attr"`
	EventEOL             string `xml:"eventEol,attr"`
	Complete             string `xml:"complete,attr"`
	IncludeStacktrace    string `xml:"includeStacktrace,attr"`
	StacktraceAsString   string `xml:"stacktraceAsString,attr"`
	PropertiesAsList     string `xml:"propertiesAsList,attr"`
	IncludeNullDelimiter string `xml:"includeNullDelimiter,attr"`
	Header               string `xml:"header,attr"`
	Footer               string `xml:"footer,attr"`
}

type xmlAppenderRef struct {
	Ref             string         `xml:"ref,attr"`
	Level           string         `xml:"level,attr"`
	IncludeLocation string         `xml:"includeLocation,attr"`
	FilterRefs      []xmlFilterRef `xml:"FilterRef"`
}

type xmlFilterRef struct {
	Ref string `xml:"ref,attr"`
}

type xmlRoute struct {
	Key         string         `xml:"key,attr"`
	Ref         string         `xml:"ref,attr"`
	AppenderRef xmlAppenderRef `xml:"AppenderRef"`
}

type xmlRemoveAttr struct {
	Key  string `xml:"key,attr"`
	Name string `xml:"name,attr"`
}

type xmlRollingPolicies struct {
	Size    xmlRollingSizePolicy `xml:"SizeBasedTriggeringPolicy"`
	Time    xmlRollingTimePolicy `xml:"TimeBasedTriggeringPolicy"`
	Cron    xmlRollingCronPolicy `xml:"CronTriggeringPolicy"`
	Startup xmlRollingStartup    `xml:"OnStartupTriggeringPolicy"`
}

type xmlRollingSizePolicy struct {
	Size string `xml:"size,attr"`
}

type xmlRollingTimePolicy struct {
	Interval string `xml:"interval,attr"`
	Modulate string `xml:"modulate,attr"`
}

type xmlRollingCronPolicy struct {
	Schedule string `xml:"schedule,attr"`
}

type xmlRollingStartup struct {
	Enabled string `xml:"enabled,attr"`
}

type xmlRollingStrategy struct {
	Type      string                 `xml:"type,attr"`
	Max       string                 `xml:"max,attr"`
	FileIndex string                 `xml:"fileIndex,attr"`
	Delete    xmlRollingDeleteAction `xml:"Delete"`
}

type xmlRollingDeleteAction struct {
	BasePath               string                           `xml:"basePath,attr"`
	MaxDepth               string                           `xml:"maxDepth,attr"`
	MaxCount               string                           `xml:"maxCount,attr"`
	MaxSize                string                           `xml:"maxSize,attr"`
	Glob                   string                           `xml:"glob,attr"`
	Age                    string                           `xml:"age,attr"`
	IfFileName             xmlRollingDeleteFileName         `xml:"IfFileName"`
	IfLastModified         xmlRollingDeleteLastModified     `xml:"IfLastModified"`
	IfAccumulatedFileCount xmlRollingDeleteAccumulatedCount `xml:"IfAccumulatedFileCount"`
	IfAccumulatedFileSize  xmlRollingDeleteAccumulatedSize  `xml:"IfAccumulatedFileSize"`
}

type xmlRollingDeleteFileName struct {
	Glob string `xml:"glob,attr"`
}

type xmlRollingDeleteLastModified struct {
	Age string `xml:"age,attr"`
}

type xmlRollingDeleteAccumulatedCount struct {
	Exceeds string `xml:"exceeds,attr"`
}

type xmlRollingDeleteAccumulatedSize struct {
	Exceeds string `xml:"exceeds,attr"`
}

type xmlFilters struct {
	Threshold          []xmlFilter `xml:"ThresholdFilter"`
	Level              []xmlFilter `xml:"LevelFilter"`
	Range              []xmlFilter `xml:"LevelRangeFilter"`
	Regex              []xmlFilter `xml:"RegexFilter"`
	Attr               []xmlFilter `xml:"AttrFilter"`
	Attribute          []xmlFilter `xml:"AttributeFilter"`
	Deny               []xmlFilter `xml:"DenyFilter"`
	DenyAll            []xmlFilter `xml:"DenyAllFilter"`
	Composite          []xmlFilter `xml:"CompositeFilter"`
	Marker             []xmlFilter `xml:"MarkerFilter"`
	NoMarker           []xmlFilter `xml:"NoMarkerFilter"`
	Map                []xmlFilter `xml:"MapFilter"`
	ThreadContextMap   []xmlFilter `xml:"ThreadContextMapFilter"`
	ThreadContextStack []xmlFilter `xml:"ThreadContextStackFilter"`
	StructuredData     []xmlFilter `xml:"StructuredDataFilter"`
	Throwable          []xmlFilter `xml:"ThrowableFilter"`
	StringMatch        []xmlFilter `xml:"StringMatchFilter"`
	Time               []xmlFilter `xml:"TimeFilter"`
	Burst              []xmlFilter `xml:"BurstFilter"`
	DynamicThreshold   []xmlFilter `xml:"DynamicThresholdFilter"`
}

type xmlFilter struct {
	Name             string            `xml:"name,attr"`
	Type             string            `xml:"type,attr"`
	Level            string            `xml:"level,attr"`
	MinLevel         string            `xml:"minLevel,attr"`
	MaxLevel         string            `xml:"maxLevel,attr"`
	Marker           string            `xml:"marker,attr"`
	Text             string            `xml:"text,attr"`
	Operator         string            `xml:"operator,attr"`
	Start            string            `xml:"start,attr"`
	End              string            `xml:"end,attr"`
	Timezone         string            `xml:"timezone,attr"`
	Rate             string            `xml:"rate,attr"`
	MaxBurst         string            `xml:"maxBurst,attr"`
	Field            string            `xml:"field,attr"`
	Key              string            `xml:"key,attr"`
	Value            string            `xml:"value,attr"`
	DefaultThreshold string            `xml:"defaultThreshold,attr"`
	Pattern          string            `xml:"pattern,attr"`
	OnMatch          string            `xml:"onMatch,attr"`
	OnMismatch       string            `xml:"onMismatch,attr"`
	FilterRefs       []xmlFilterRef    `xml:"FilterRef"`
	KeyValuePair     []xmlKeyValuePair `xml:"KeyValuePair"`
}

type xmlKeyValuePair struct {
	Key   string `xml:"key,attr"`
	Value string `xml:"value,attr"`
}

type xmlAsyncLogger struct {
	Enabled          string `xml:"enabled,attr"`
	QueueSize        string `xml:"queueSize,attr"`
	BatchSize        string `xml:"batchSize,attr"`
	OverflowStrategy string `xml:"overflowStrategy,attr"`
	WaitStrategy     string `xml:"waitStrategy,attr"`
	WaitRetries      string `xml:"waitRetries,attr"`
	SleepTime        string `xml:"sleepTime,attr"`
	Timeout          string `xml:"timeout,attr"`
	IncludeLocation  string `xml:"includeLocation,attr"`
}

type xmlLoggers struct {
	Root   xmlLogger   `xml:"Root"`
	Logger []xmlLogger `xml:"Logger"`
}

type xmlLogger struct {
	Name            string           `xml:"name,attr"`
	Level           string           `xml:"level,attr"`
	Additivity      string           `xml:"additivity,attr"`
	IncludeLocation string           `xml:"includeLocation,attr"`
	AppenderRefs    []xmlAppenderRef `xml:"AppenderRef"`
	FilterRefs      []xmlFilterRef   `xml:"FilterRef"`
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
		levels[name] = firstNonBlank(level.IntLevel, level.Value, level.Text)
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
		properties[name] = firstNonBlank(property.Value, property.Text)
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
	appendEnabled, err := parseXMLBoolPointerStrict(a.Append, "append")
	if err != nil {
		return "", appenderConfig{}, fmt.Errorf("goark-log: XML appender %q: %w", name, err)
	}
	createOnDemand, err := parseXMLBool(a.CreateOnDemand, "createOnDemand")
	if err != nil {
		return "", appenderConfig{}, fmt.Errorf("goark-log: XML appender %q: %w", name, err)
	}
	waitRetries, err := parseXMLInt(a.WaitRetries, "waitRetries")
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
					Modulate: parseXMLBoolPointer(a.Policies.Time.Modulate),
				},
				CronTriggeringPolicy: rollingCronPolicyConfig{
					Schedule: a.Policies.Cron.Schedule,
				},
				OnStartupTriggeringPolicy: rollingStartupPolicyConfig{
					Enabled: parseXMLBoolPointer(a.Policies.Startup.Enabled),
				},
			},
			Strategy: rollingStrategyConfig{
				Type:      strategy.Type,
				Max:       parseXMLIntPointer(strategy.Max),
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

func (a xmlAppender) layout() (layoutConfig, error) {
	for _, layout := range []xmlLayout{
		a.PatternLayout,
		a.TextLayout,
		a.JsonLayout,
		a.JSONLayout,
		a.JsonTemplate,
		a.XmlLayout,
		a.XMLLayout,
		a.CsvLayout,
		a.CSVLayout,
		a.GelfLayout,
		a.GELFLayout,
		a.Rfc5424Layout,
		a.RFC5424Layout,
		a.SyslogLayout,
		a.YamlLayout,
		a.YAMLLayout,
		a.HtmlLayout,
		a.HTMLLayout,
		a.Layout,
	} {
		if layout.XMLName.Local == "" &&
			strings.TrimSpace(layout.Type) == "" &&
			strings.TrimSpace(layout.Pattern) == "" &&
			strings.TrimSpace(layout.Template) == "" &&
			strings.TrimSpace(layout.TemplateURI) == "" &&
			layout.emptyOptions() {
			continue
		}
		return layout.config()
	}
	return layoutConfig{}, nil
}

func (l xmlLayout) emptyOptions() bool {
	return strings.TrimSpace(l.Compact) == "" &&
		strings.TrimSpace(l.EventEOL) == "" &&
		strings.TrimSpace(l.Complete) == "" &&
		strings.TrimSpace(l.IncludeStacktrace) == "" &&
		strings.TrimSpace(l.StacktraceAsString) == "" &&
		strings.TrimSpace(l.PropertiesAsList) == "" &&
		strings.TrimSpace(l.IncludeNullDelimiter) == "" &&
		strings.TrimSpace(l.Header) == "" &&
		strings.TrimSpace(l.Footer) == ""
}

func (l xmlLayout) config() (layoutConfig, error) {
	kind := firstNonBlank(l.Type, l.XMLName.Local)
	switch normalizeKind(kind) {
	case "", "patternlayout", "pattern":
		kind = "pattern"
	case "textlayout", "text":
		kind = "text"
	case "jsonlayout", "json":
		kind = "json"
	case "jsontemplatelayout", "jsontemplate":
		kind = "jsonTemplate"
	case "xmllayout", "xml":
		kind = "xml"
	case "csvlayout", "csv":
		kind = "csv"
	case "gelflayout", "gelf":
		kind = "gelf"
	case "rfc5424layout", "rfc5424":
		kind = "rfc5424"
	case "sysloglayout", "syslog":
		kind = "syslog"
	case "yamllayout", "yaml":
		kind = "yaml"
	case "htmllayout", "html":
		kind = "html"
	default:
		kind = l.Type
	}
	compact, err := parseXMLBool(l.Compact, "compact")
	if err != nil {
		return layoutConfig{}, err
	}
	eventEOL, err := parseXMLBool(l.EventEOL, "eventEol")
	if err != nil {
		return layoutConfig{}, err
	}
	complete, err := parseXMLBool(l.Complete, "complete")
	if err != nil {
		return layoutConfig{}, err
	}
	includeStacktrace, err := parseXMLBool(l.IncludeStacktrace, "includeStacktrace")
	if err != nil {
		return layoutConfig{}, err
	}
	stacktraceAsString, err := parseXMLBool(l.StacktraceAsString, "stacktraceAsString")
	if err != nil {
		return layoutConfig{}, err
	}
	propertiesAsList, err := parseXMLBool(l.PropertiesAsList, "propertiesAsList")
	if err != nil {
		return layoutConfig{}, err
	}
	includeNullDelimiter, err := parseXMLBool(l.IncludeNullDelimiter, "includeNullDelimiter")
	if err != nil {
		return layoutConfig{}, err
	}
	return layoutConfig{
		Type:                 kind,
		Pattern:              l.Pattern,
		EventTemplate:        l.Template,
		EventTemplateURI:     l.TemplateURI,
		Compact:              compact,
		EventEOL:             eventEOL,
		Complete:             complete,
		IncludeStacktrace:    includeStacktrace,
		StacktraceAsString:   stacktraceAsString,
		PropertiesAsList:     propertiesAsList,
		IncludeNullDelimiter: includeNullDelimiter,
		Header:               l.Header,
		Footer:               l.Footer,
	}, nil
}

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

func (l xmlLogger) config(named bool) (loggerConfig, error) {
	if named && strings.TrimSpace(l.Name) == "" {
		return loggerConfig{}, fmt.Errorf("goark-log: XML logger name is empty")
	}
	appenderRefs, err := xmlAppenderRefs(l.AppenderRefs)
	if err != nil {
		return loggerConfig{}, fmt.Errorf("goark-log: XML logger %q: %w", l.Name, err)
	}
	includeLocation, err := parseXMLBoolPointerStrict(l.IncludeLocation, "includeLocation")
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

func xmlAppenderRefs(refs []xmlAppenderRef) (appenderRefs, error) {
	out := make(appenderRefs, 0, len(refs))
	for _, ref := range refs {
		includeLocation, err := parseXMLBoolPointerStrict(ref.IncludeLocation, "includeLocation")
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
		out[key] = firstNonBlank(route.Ref, route.AppenderRef.Ref)
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
		if key := firstNonBlank(value.Key, value.Name); key != "" {
			out = append(out, key)
		}
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

func parseXMLIntValue(value string) int {
	parsed, err := parseXMLInt(value, "")
	if err != nil {
		return 0
	}
	return parsed
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

func parseXMLBoolPointerStrict(value string, field string) (*bool, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parsed, err := parseXMLBool(value, field)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func parseXMLBoolValue(value string) bool {
	parsed, err := parseXMLBool(value, "")
	if err != nil {
		return false
	}
	return parsed
}
