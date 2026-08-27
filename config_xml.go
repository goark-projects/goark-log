package goarklog

import (
	"encoding/xml"
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
	DisableANSI          string `xml:"disableAnsi,attr"`
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
