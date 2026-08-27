package goarklog

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"goark.dev/log/internal/textutil"
	"gopkg.in/yaml.v3"
)

type fileConfig struct {
	Configuration     *fileConfig               `yaml:"configuration"`
	Status            string                    `yaml:"status"`
	MonitorInterval   string                    `yaml:"monitorInterval"`
	MonitorKebab      string                    `yaml:"monitor-interval"`
	Properties        map[string]string         `yaml:"properties"`
	CustomLevels      map[string]string         `yaml:"customLevels"`
	CustomLevelsKebab map[string]string         `yaml:"custom-levels"`
	Appenders         map[string]appenderConfig `yaml:"appenders"`
	Filters           map[string]filterConfig   `yaml:"filters"`
	FilterRefs        []string                  `yaml:"filterRefs"`
	FilterRefsKebab   []string                  `yaml:"filter-refs"`
	AsyncLogger       asyncLoggerConfig         `yaml:"asyncLogger"`
	AsyncLoggerKebab  asyncLoggerConfig         `yaml:"async-logger"`
	Async             asyncLoggerConfig         `yaml:"async"`
	Root              loggerConfig              `yaml:"root"`
	Loggers           map[string]loggerConfig   `yaml:"loggers"`
	Goark             struct {
		Log *fileConfig `yaml:"log"`
	} `yaml:"goark"`
}

type appenderConfig struct {
	Type                  string             `yaml:"type"`
	Target                string             `yaml:"target"`
	URL                   string             `yaml:"url"`
	Method                string             `yaml:"method"`
	Address               string             `yaml:"address"`
	Network               string             `yaml:"network"`
	Facility              string             `yaml:"facility"`
	AppName               string             `yaml:"appName"`
	AppNameKebab          string             `yaml:"app-name"`
	ConnectTimeout        string             `yaml:"connectTimeout"`
	ConnectTimeoutKebab   string             `yaml:"connect-timeout"`
	WriteTimeout          string             `yaml:"writeTimeout"`
	WriteTimeoutKebab     string             `yaml:"write-timeout"`
	FileName              string             `yaml:"fileName"`
	FileNameKebab         string             `yaml:"file-name"`
	Path                  string             `yaml:"path"`
	Layout                layoutConfig       `yaml:"layout"`
	Rolling               rollingConfig      `yaml:"rolling"`
	AppenderRefs          appenderRefs       `yaml:"appenderRefs"`
	AppenderRefsKebab     appenderRefs       `yaml:"appender-refs"`
	Refs                  appenderRefs       `yaml:"refs"`
	Primary               string             `yaml:"primary"`
	PrimaryKebab          string             `yaml:"primary-ref"`
	Failovers             []string           `yaml:"failovers"`
	FailoversKebab        []string           `yaml:"failover-refs"`
	RouteKey              string             `yaml:"routeKey"`
	RouteKeyKebab         string             `yaml:"route-key"`
	DefaultRoute          string             `yaml:"defaultRoute"`
	DefaultRouteKebab     string             `yaml:"default-route"`
	Routes                map[string]string  `yaml:"routes"`
	Rewrite               rewriteBuildConfig `yaml:"rewrite"`
	QueueSize             int                `yaml:"queueSize"`
	QueueSizeKebab        int                `yaml:"queue-size"`
	BatchSize             int                `yaml:"batchSize"`
	BatchSizeKebab        int                `yaml:"batch-size"`
	OverflowStrategy      string             `yaml:"overflowStrategy"`
	OverflowStrategyKebab string             `yaml:"overflow-strategy"`
	WaitStrategy          string             `yaml:"waitStrategy"`
	WaitStrategyKebab     string             `yaml:"wait-strategy"`
	WaitRetries           int                `yaml:"waitRetries"`
	WaitRetriesKebab      int                `yaml:"wait-retries"`
	SleepTime             string             `yaml:"sleepTime"`
	SleepTimeKebab        string             `yaml:"sleep-time"`
	Timeout               string             `yaml:"timeout"`
	BufferSize            string             `yaml:"bufferSize"`
	BufferSizeKebab       string             `yaml:"buffer-size"`
	FlushOnWrite          bool               `yaml:"flushOnWrite"`
	FlushOnWriteKebab     bool               `yaml:"flush-on-write"`
	Append                *bool              `yaml:"append"`
	CreateOnDemand        bool               `yaml:"createOnDemand"`
	CreateOnDemandKebab   bool               `yaml:"create-on-demand"`
	FilePermissions       string             `yaml:"filePermissions"`
	FilePermissionsKebab  string             `yaml:"file-permissions"`
	Filters               []string           `yaml:"filters"`
	FilterRefs            []string           `yaml:"filterRefs"`
	FilterRefsKebab       []string           `yaml:"filter-refs"`
}

type layoutConfig struct {
	Type                      string `yaml:"type"`
	Pattern                   string `yaml:"pattern"`
	EventTemplate             string `yaml:"eventTemplate"`
	EventTemplateKebab        string `yaml:"event-template"`
	EventTemplateURI          string `yaml:"eventTemplateUri"`
	EventTemplateURIKebab     string `yaml:"event-template-uri"`
	EventTemplatePath         string `yaml:"eventTemplatePath"`
	EventTemplatePathKebab    string `yaml:"event-template-path"`
	Compact                   bool   `yaml:"compact"`
	EventEOL                  bool   `yaml:"eventEol"`
	EventEOLKebab             bool   `yaml:"event-eol"`
	Complete                  bool   `yaml:"complete"`
	IncludeStacktrace         bool   `yaml:"includeStacktrace"`
	IncludeStacktraceKebab    bool   `yaml:"include-stacktrace"`
	StacktraceAsString        bool   `yaml:"stacktraceAsString"`
	StacktraceAsStringKebab   bool   `yaml:"stacktrace-as-string"`
	PropertiesAsList          bool   `yaml:"propertiesAsList"`
	PropertiesAsListKebab     bool   `yaml:"properties-as-list"`
	IncludeNullDelimiter      bool   `yaml:"includeNullDelimiter"`
	IncludeNullDelimiterKebab bool   `yaml:"include-null-delimiter"`
	DisableANSI               bool   `yaml:"disableAnsi"`
	DisableANSIKebab          bool   `yaml:"disable-ansi"`
	Header                    string `yaml:"header"`
	Footer                    string `yaml:"footer"`
}

type filterConfig struct {
	Type               string               `yaml:"type"`
	Level              string               `yaml:"level"`
	MinLevel           string               `yaml:"minLevel"`
	MinLevelKebab      string               `yaml:"min-level"`
	MaxLevel           string               `yaml:"maxLevel"`
	MaxLevelKebab      string               `yaml:"max-level"`
	Marker             string               `yaml:"marker"`
	Text               string               `yaml:"text"`
	Operator           string               `yaml:"operator"`
	Start              string               `yaml:"start"`
	End                string               `yaml:"end"`
	Timezone           string               `yaml:"timezone"`
	Rate               string               `yaml:"rate"`
	MaxBurst           int                  `yaml:"maxBurst"`
	MaxBurstKebab      int                  `yaml:"max-burst"`
	Field              string               `yaml:"field"`
	Key                string               `yaml:"key"`
	Value              string               `yaml:"value"`
	Values             map[string]string    `yaml:"values"`
	Thresholds         map[string]string    `yaml:"thresholds"`
	Filters            []string             `yaml:"filters"`
	FilterRefs         []string             `yaml:"filterRefs"`
	FilterRefsKebab    []string             `yaml:"filter-refs"`
	KeyValuePair       []keyValuePairConfig `yaml:"KeyValuePair"`
	KeyValuePairs      []keyValuePairConfig `yaml:"keyValuePairs"`
	KeyValuePairsKebab []keyValuePairConfig `yaml:"key-value-pairs"`
	DefaultThreshold   string               `yaml:"defaultThreshold"`
	DefaultKebab       string               `yaml:"default-threshold"`
	Pattern            string               `yaml:"pattern"`
	OnMatch            string               `yaml:"onMatch"`
	OnMatchKebab       string               `yaml:"on-match"`
	OnMismatch         string               `yaml:"onMismatch"`
	OnMismatchKebab    string               `yaml:"on-mismatch"`
}

type keyValuePairConfig struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

type rollingConfig struct {
	FilePattern          string                `yaml:"filePattern"`
	FilePatternKebab     string                `yaml:"file-pattern"`
	MaxSize              string                `yaml:"maxSize"`
	MaxSizeKebab         string                `yaml:"max-size"`
	Interval             string                `yaml:"interval"`
	Cron                 string                `yaml:"cron"`
	CronSchedule         string                `yaml:"cronSchedule"`
	CronScheduleKebab    string                `yaml:"cron-schedule"`
	OnStartup            bool                  `yaml:"onStartup"`
	OnStartupKebab       bool                  `yaml:"on-startup"`
	MaxBackups           *int                  `yaml:"maxBackups"`
	MaxBackupsKebab      *int                  `yaml:"max-backups"`
	MaxAge               string                `yaml:"maxAge"`
	MaxAgeKebab          string                `yaml:"max-age"`
	Gzip                 bool                  `yaml:"gzip"`
	Compress             bool                  `yaml:"compress"`
	DirectWrite          bool                  `yaml:"directWrite"`
	DirectWriteKebab     bool                  `yaml:"direct-write"`
	AsyncActions         bool                  `yaml:"asyncActions"`
	AsyncActionsKebab    bool                  `yaml:"async-actions"`
	ActionQueueSize      int                   `yaml:"actionQueueSize"`
	ActionQueueSizeKebab int                   `yaml:"action-queue-size"`
	Policies             rollingPoliciesConfig `yaml:"policies"`
	Strategy             rollingStrategyConfig `yaml:"strategy"`
}

type rollingPoliciesConfig struct {
	Size                         rollingSizePolicyConfig    `yaml:"size"`
	SizeKebab                    rollingSizePolicyConfig    `yaml:"size-based-triggering-policy"`
	SizeBasedTriggeringPolicy    rollingSizePolicyConfig    `yaml:"sizeBasedTriggeringPolicy"`
	SizeBasedTriggeringPolicyXML rollingSizePolicyConfig    `yaml:"SizeBasedTriggeringPolicy"`
	Time                         rollingTimePolicyConfig    `yaml:"time"`
	TimeKebab                    rollingTimePolicyConfig    `yaml:"time-based-triggering-policy"`
	TimeBasedTriggeringPolicy    rollingTimePolicyConfig    `yaml:"timeBasedTriggeringPolicy"`
	TimeBasedTriggeringPolicyXML rollingTimePolicyConfig    `yaml:"TimeBasedTriggeringPolicy"`
	Cron                         rollingCronPolicyConfig    `yaml:"cron"`
	CronKebab                    rollingCronPolicyConfig    `yaml:"cron-triggering-policy"`
	CronTriggeringPolicy         rollingCronPolicyConfig    `yaml:"cronTriggeringPolicy"`
	CronTriggeringPolicyXML      rollingCronPolicyConfig    `yaml:"CronTriggeringPolicy"`
	Startup                      rollingStartupPolicyConfig `yaml:"startup"`
	StartupKebab                 rollingStartupPolicyConfig `yaml:"on-startup-triggering-policy"`
	OnStartupTriggeringPolicy    rollingStartupPolicyConfig `yaml:"onStartupTriggeringPolicy"`
	OnStartupTriggeringPolicyXML rollingStartupPolicyConfig `yaml:"OnStartupTriggeringPolicy"`
}

type rollingSizePolicyConfig struct {
	Size         string `yaml:"size"`
	MaxSize      string `yaml:"maxSize"`
	MaxSizeKebab string `yaml:"max-size"`
}

type rollingTimePolicyConfig struct {
	Interval string `yaml:"interval"`
	Every    string `yaml:"every"`
	Unit     string `yaml:"unit"`
	Modulate *bool  `yaml:"modulate"`
}

type rollingCronPolicyConfig struct {
	Schedule          string `yaml:"schedule"`
	Cron              string `yaml:"cron"`
	CronSchedule      string `yaml:"cronSchedule"`
	CronKebab         string `yaml:"cron-schedule"`
	EvaluateOnStartup bool   `yaml:"evaluateOnStartup"`
}

type rollingStartupPolicyConfig struct {
	Enabled *bool `yaml:"enabled"`
}

type rollingStrategyConfig struct {
	Type                 string                      `yaml:"type"`
	Max                  *int                        `yaml:"max"`
	MaxBackups           *int                        `yaml:"maxBackups"`
	MaxBackupsKebab      *int                        `yaml:"max-backups"`
	MaxAge               string                      `yaml:"maxAge"`
	MaxAgeKebab          string                      `yaml:"max-age"`
	FileIndex            string                      `yaml:"fileIndex"`
	FileIndexKebab       string                      `yaml:"file-index"`
	DirectWrite          bool                        `yaml:"directWrite"`
	DirectWriteKebab     bool                        `yaml:"direct-write"`
	AsyncActions         bool                        `yaml:"asyncActions"`
	AsyncActionsKebab    bool                        `yaml:"async-actions"`
	ActionQueueSize      int                         `yaml:"actionQueueSize"`
	ActionQueueSizeKebab int                         `yaml:"action-queue-size"`
	Compression          rollingCompressionConfig    `yaml:"compression"`
	Delete               rollingDeleteActionConfig   `yaml:"delete"`
	DeleteActions        []rollingDeleteActionConfig `yaml:"deleteActions"`
	DeleteActionsKebab   []rollingDeleteActionConfig `yaml:"delete-actions"`
}

type rollingCompressionConfig struct {
	Gzip     bool `yaml:"gzip"`
	Compress bool `yaml:"compress"`
	Async    bool `yaml:"async"`
}

type rollingDeleteActionConfig struct {
	BasePath                    string                              `yaml:"basePath"`
	BasePathKebab               string                              `yaml:"base-path"`
	MaxDepth                    *int                                `yaml:"maxDepth"`
	MaxDepthKebab               *int                                `yaml:"max-depth"`
	MaxCount                    *int                                `yaml:"maxCount"`
	MaxCountKebab               *int                                `yaml:"max-count"`
	MaxSize                     string                              `yaml:"maxSize"`
	MaxSizeKebab                string                              `yaml:"max-size"`
	Glob                        string                              `yaml:"glob"`
	Age                         string                              `yaml:"age"`
	Async                       bool                                `yaml:"async"`
	IfFileName                  rollingDeleteFileNameConfig         `yaml:"ifFileName"`
	IfFileNameKebab             rollingDeleteFileNameConfig         `yaml:"if-file-name"`
	IfLastModified              rollingDeleteLastModifiedConfig     `yaml:"ifLastModified"`
	IfLastModifiedKebab         rollingDeleteLastModifiedConfig     `yaml:"if-last-modified"`
	IfAccumulatedFileCount      rollingDeleteAccumulatedCountConfig `yaml:"ifAccumulatedFileCount"`
	IfAccumulatedFileCountKebab rollingDeleteAccumulatedCountConfig `yaml:"if-accumulated-file-count"`
	IfAccumulatedFileSize       rollingDeleteAccumulatedSizeConfig  `yaml:"ifAccumulatedFileSize"`
	IfAccumulatedFileSizeKebab  rollingDeleteAccumulatedSizeConfig  `yaml:"if-accumulated-file-size"`
}

type rollingDeleteFileNameConfig struct {
	Glob string `yaml:"glob"`
}

type rollingDeleteLastModifiedConfig struct {
	Age string `yaml:"age"`
}

type rollingDeleteAccumulatedCountConfig struct {
	Exceeds int `yaml:"exceeds"`
}

type rollingDeleteAccumulatedSizeConfig struct {
	Exceeds string `yaml:"exceeds"`
}

type asyncLoggerConfig struct {
	Enabled               *bool  `yaml:"enabled"`
	QueueSize             int    `yaml:"queueSize"`
	QueueSizeKebab        int    `yaml:"queue-size"`
	BatchSize             int    `yaml:"batchSize"`
	BatchSizeKebab        int    `yaml:"batch-size"`
	OverflowStrategy      string `yaml:"overflowStrategy"`
	OverflowStrategyKebab string `yaml:"overflow-strategy"`
	WaitStrategy          string `yaml:"waitStrategy"`
	WaitStrategyKebab     string `yaml:"wait-strategy"`
	WaitRetries           int    `yaml:"waitRetries"`
	WaitRetriesKebab      int    `yaml:"wait-retries"`
	SleepTime             string `yaml:"sleepTime"`
	SleepTimeKebab        string `yaml:"sleep-time"`
	Timeout               string `yaml:"timeout"`
	IncludeLocation       *bool  `yaml:"includeLocation"`
	IncludeLocationKebab  *bool  `yaml:"include-location"`
}

type loggerConfig struct {
	Level                string       `yaml:"level"`
	AppenderRefs         appenderRefs `yaml:"appenderRefs"`
	AppenderRefsKebab    appenderRefs `yaml:"appender-refs"`
	Refs                 appenderRefs `yaml:"refs"`
	Filters              []string     `yaml:"filters"`
	FilterRefs           []string     `yaml:"filterRefs"`
	FilterRefsKebab      []string     `yaml:"filter-refs"`
	Additivity           *bool        `yaml:"additivity"`
	IncludeLocation      *bool        `yaml:"includeLocation"`
	IncludeLocationKebab *bool        `yaml:"include-location"`
}

type appenderRefs []appenderRefConfig

type appenderRefConfig struct {
	ID                   string   `yaml:"-"`
	Ref                  string   `yaml:"ref"`
	Level                string   `yaml:"level"`
	IncludeLocation      *bool    `yaml:"includeLocation"`
	IncludeLocationKebab *bool    `yaml:"include-location"`
	Filters              []string `yaml:"filters"`
	FilterRefs           []string `yaml:"filterRefs"`
	FilterRefsKebab      []string `yaml:"filter-refs"`
}

func loadConfigFile(ctx context.Context, path string, lookups *LookupResolver) (*fileConfig, error) {
	if ctx == nil {
		return nil, fmt.Errorf("goark-log: context is nil")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	format, err := configFormat(path)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("goark-log: open config file %q: %w", path, err)
	}
	defer file.Close()
	config, err := decodeConfig(file, format, lookups)
	if err != nil {
		return nil, fmt.Errorf("goark-log: parse config file %q: %w", path, err)
	}
	return config, nil
}

func decodeConfig(reader io.Reader, format string, lookups *LookupResolver) (*fileConfig, error) {
	switch format {
	case "yaml", "json":
		return decodeStructuredConfig(reader, lookups)
	case "xml":
		return decodeXMLConfig(reader, lookups)
	case "properties":
		return decodePropertiesConfig(reader, lookups)
	default:
		return nil, fmt.Errorf("goark-log: unsupported config format %q", format)
	}
}

func decodeStructuredConfig(reader io.Reader, lookups *LookupResolver) (*fileConfig, error) {
	var config fileConfig
	decoder := yaml.NewDecoder(reader)
	decoder.KnownFields(true)
	if err := decoder.Decode(&config); err != nil {
		if errors.Is(err, io.EOF) {
			return &fileConfig{}, nil
		}
		return nil, err
	}
	return finalizeDecodedConfig(config, lookups)
}

func (c *fileConfig) effective() (*fileConfig, error) {
	topLevelUsed := !c.withoutWrappers().empty()
	wrappers := 0
	if c.Goark.Log != nil {
		wrappers++
	}
	if c.Configuration != nil {
		wrappers++
	}
	if wrappers == 0 {
		return c, nil
	}
	if topLevelUsed {
		return nil, fmt.Errorf("goark-log: config must use either top-level fields, configuration, or goark.log")
	}
	if wrappers > 1 {
		return nil, fmt.Errorf("goark-log: config must use only one wrapper: configuration or goark.log")
	}
	if c.Configuration != nil {
		return c.Configuration, nil
	}
	return c.Goark.Log, nil
}

func (c *fileConfig) withoutWrappers() *fileConfig {
	if c == nil {
		return nil
	}
	return &fileConfig{
		Status:            c.Status,
		MonitorInterval:   c.MonitorInterval,
		MonitorKebab:      c.MonitorKebab,
		Properties:        c.Properties,
		CustomLevels:      c.CustomLevels,
		CustomLevelsKebab: c.CustomLevelsKebab,
		Appenders:         c.Appenders,
		Filters:           c.Filters,
		FilterRefs:        c.FilterRefs,
		FilterRefsKebab:   c.FilterRefsKebab,
		AsyncLogger:       c.AsyncLogger,
		AsyncLoggerKebab:  c.AsyncLoggerKebab,
		Async:             c.Async,
		Root:              c.Root,
		Loggers:           c.Loggers,
	}
}

func (c *fileConfig) empty() bool {
	if c == nil {
		return true
	}
	return len(c.Appenders) == 0 &&
		strings.TrimSpace(c.Status) == "" &&
		strings.TrimSpace(c.MonitorInterval) == "" &&
		strings.TrimSpace(c.MonitorKebab) == "" &&
		len(c.Properties) == 0 &&
		len(c.CustomLevels) == 0 &&
		len(c.CustomLevelsKebab) == 0 &&
		len(c.Filters) == 0 &&
		len(c.FilterRefs) == 0 &&
		len(c.FilterRefsKebab) == 0 &&
		c.AsyncLogger.empty() &&
		c.AsyncLoggerKebab.empty() &&
		c.Async.empty() &&
		c.Root.empty() &&
		len(c.Loggers) == 0
}

func (c loggerConfig) empty() bool {
	return strings.TrimSpace(c.Level) == "" &&
		len(c.AppenderRefs) == 0 &&
		len(c.AppenderRefsKebab) == 0 &&
		len(c.Refs) == 0 &&
		len(c.Filters) == 0 &&
		len(c.FilterRefs) == 0 &&
		len(c.FilterRefsKebab) == 0 &&
		c.Additivity == nil &&
		c.IncludeLocation == nil &&
		c.IncludeLocationKebab == nil
}

func (c asyncLoggerConfig) empty() bool {
	return c.Enabled == nil &&
		c.QueueSize == 0 &&
		c.QueueSizeKebab == 0 &&
		c.BatchSize == 0 &&
		c.BatchSizeKebab == 0 &&
		strings.TrimSpace(c.OverflowStrategy) == "" &&
		strings.TrimSpace(c.OverflowStrategyKebab) == "" &&
		strings.TrimSpace(c.WaitStrategy) == "" &&
		strings.TrimSpace(c.WaitStrategyKebab) == "" &&
		c.WaitRetries == 0 &&
		c.WaitRetriesKebab == 0 &&
		strings.TrimSpace(c.SleepTime) == "" &&
		strings.TrimSpace(c.SleepTimeKebab) == "" &&
		strings.TrimSpace(c.Timeout) == "" &&
		c.IncludeLocation == nil &&
		c.IncludeLocationKebab == nil
}

func finalizeDecodedConfig(config fileConfig, lookups *LookupResolver) (*fileConfig, error) {
	effective, err := config.effective()
	if err != nil {
		return nil, err
	}
	if lookups == nil {
		lookups = NewLookupResolver()
	}
	if err := effective.resolveLookups(lookups.Clone()); err != nil {
		return nil, err
	}
	return effective, nil
}

func (c *fileConfig) monitorInterval() (time.Duration, error) {
	if c == nil {
		return 0, nil
	}
	return ParseMonitorInterval(textutil.FirstNonBlank(c.MonitorInterval, c.MonitorKebab))
}
