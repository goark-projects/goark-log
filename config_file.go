package goarklog

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

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
	Type                  string        `yaml:"type"`
	Target                string        `yaml:"target"`
	URL                   string        `yaml:"url"`
	Method                string        `yaml:"method"`
	Address               string        `yaml:"address"`
	Network               string        `yaml:"network"`
	Facility              string        `yaml:"facility"`
	AppName               string        `yaml:"appName"`
	AppNameKebab          string        `yaml:"app-name"`
	ConnectTimeout        string        `yaml:"connectTimeout"`
	ConnectTimeoutKebab   string        `yaml:"connect-timeout"`
	WriteTimeout          string        `yaml:"writeTimeout"`
	WriteTimeoutKebab     string        `yaml:"write-timeout"`
	FileName              string        `yaml:"fileName"`
	FileNameKebab         string        `yaml:"file-name"`
	Path                  string        `yaml:"path"`
	Layout                layoutConfig  `yaml:"layout"`
	Rolling               rollingConfig `yaml:"rolling"`
	AppenderRefs          appenderRefs  `yaml:"appenderRefs"`
	AppenderRefsKebab     appenderRefs  `yaml:"appender-refs"`
	Refs                  appenderRefs  `yaml:"refs"`
	QueueSize             int           `yaml:"queueSize"`
	QueueSizeKebab        int           `yaml:"queue-size"`
	OverflowStrategy      string        `yaml:"overflowStrategy"`
	OverflowStrategyKebab string        `yaml:"overflow-strategy"`
	WaitStrategy          string        `yaml:"waitStrategy"`
	WaitStrategyKebab     string        `yaml:"wait-strategy"`
	WaitRetries           int           `yaml:"waitRetries"`
	WaitRetriesKebab      int           `yaml:"wait-retries"`
	SleepTime             string        `yaml:"sleepTime"`
	SleepTimeKebab        string        `yaml:"sleep-time"`
	Timeout               string        `yaml:"timeout"`
	BufferSize            string        `yaml:"bufferSize"`
	BufferSizeKebab       string        `yaml:"buffer-size"`
	FlushOnWrite          bool          `yaml:"flushOnWrite"`
	FlushOnWriteKebab     bool          `yaml:"flush-on-write"`
	Filters               []string      `yaml:"filters"`
	FilterRefs            []string      `yaml:"filterRefs"`
	FilterRefsKebab       []string      `yaml:"filter-refs"`
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
	Header                    string `yaml:"header"`
	Footer                    string `yaml:"footer"`
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
	Level             string       `yaml:"level"`
	AppenderRefs      appenderRefs `yaml:"appenderRefs"`
	AppenderRefsKebab appenderRefs `yaml:"appender-refs"`
	Refs              appenderRefs `yaml:"refs"`
	Filters           []string     `yaml:"filters"`
	FilterRefs        []string     `yaml:"filterRefs"`
	FilterRefsKebab   []string     `yaml:"filter-refs"`
	Additivity        *bool        `yaml:"additivity"`
}

type appenderRefs []appenderRefConfig

type appenderRefConfig struct {
	Ref             string   `yaml:"ref"`
	Level           string   `yaml:"level"`
	Filters         []string `yaml:"filters"`
	FilterRefs      []string `yaml:"filterRefs"`
	FilterRefsKebab []string `yaml:"filter-refs"`
}

func (r *appenderRefs) UnmarshalYAML(node *yaml.Node) error {
	if node == nil || node.Kind == 0 {
		return nil
	}
	if node.Kind != yaml.SequenceNode {
		return fmt.Errorf("goark-log: appenderRefs must be a sequence")
	}
	refs := make([]appenderRefConfig, 0, len(node.Content))
	for index, item := range node.Content {
		switch item.Kind {
		case yaml.ScalarNode:
			var ref string
			if err := item.Decode(&ref); err != nil {
				return fmt.Errorf("goark-log: appenderRefs[%d]: %w", index, err)
			}
			refs = append(refs, appenderRefConfig{Ref: ref})
		case yaml.MappingNode:
			var ref appenderRefConfig
			if err := item.Decode(&ref); err != nil {
				return fmt.Errorf("goark-log: appenderRefs[%d]: %w", index, err)
			}
			refs = append(refs, ref)
		default:
			return fmt.Errorf("goark-log: appenderRefs[%d] must be a string or object", index)
		}
	}
	*r = refs
	return nil
}

func (r appenderRefs) strings() []string {
	if len(r) == 0 {
		return nil
	}
	refs := make([]string, 0, len(r))
	for _, ref := range r {
		if ref.hasControls() {
			continue
		}
		refs = append(refs, strings.TrimSpace(ref.Ref))
	}
	return refs
}

func (r appenderRefs) controls(filters map[string]Filter) ([]AppenderRef, error) {
	if len(r) == 0 {
		return nil, nil
	}
	controls := make([]AppenderRef, 0, len(r))
	for _, ref := range r {
		if !ref.hasControls() {
			continue
		}
		control, err := ref.build(filters)
		if err != nil {
			return nil, err
		}
		controls = append(controls, control)
	}
	return controls, nil
}

func (r appenderRefs) resolveLookups(lookups *LookupResolver) (appenderRefs, error) {
	if len(r) == 0 {
		return r, nil
	}
	out := make(appenderRefs, 0, len(r))
	for index, ref := range r {
		resolved, err := ref.resolveLookups(lookups)
		if err != nil {
			return nil, fmt.Errorf("appenderRefs[%d]: %w", index, err)
		}
		out = append(out, resolved)
	}
	return out, nil
}

func (c appenderRefConfig) hasControls() bool {
	return strings.TrimSpace(c.Level) != "" ||
		len(c.Filters) > 0 ||
		len(c.FilterRefs) > 0 ||
		len(c.FilterRefsKebab) > 0
}

func (c appenderRefConfig) filterRefs() []string {
	return firstStringRefs(c.Filters, c.FilterRefs, c.FilterRefsKebab)
}

func (c appenderRefConfig) build(filters map[string]Filter) (AppenderRef, error) {
	ref := AppenderRef{Ref: strings.TrimSpace(c.Ref)}
	if strings.TrimSpace(c.Level) != "" {
		level, err := ParseLevel(c.Level)
		if err != nil {
			return AppenderRef{}, err
		}
		ref.Level = &level
	}
	resolved, err := resolveFilters(filters, c.filterRefs())
	if err != nil {
		return AppenderRef{}, err
	}
	ref.Filters = resolved
	return ref, nil
}

func (c appenderRefConfig) resolveLookups(lookups *LookupResolver) (appenderRefConfig, error) {
	var err error
	if c.Ref, err = resolveStringLookup(lookups, c.Ref); err != nil {
		return appenderRefConfig{}, fmt.Errorf("ref: %w", err)
	}
	if c.Level, err = resolveStringLookup(lookups, c.Level); err != nil {
		return appenderRefConfig{}, fmt.Errorf("level: %w", err)
	}
	if c.Filters, err = resolveStringListLookups(lookups, c.Filters); err != nil {
		return appenderRefConfig{}, fmt.Errorf("filters: %w", err)
	}
	if c.FilterRefs, err = resolveStringListLookups(lookups, c.FilterRefs); err != nil {
		return appenderRefConfig{}, fmt.Errorf("filterRefs: %w", err)
	}
	if c.FilterRefsKebab, err = resolveStringListLookups(lookups, c.FilterRefsKebab); err != nil {
		return appenderRefConfig{}, fmt.Errorf("filter-refs: %w", err)
	}
	return c, nil
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

func finalizeDecodedConfig(config fileConfig, lookups *LookupResolver) (*fileConfig, error) {
	effective, err := config.effective()
	if err != nil {
		return nil, err
	}
	if lookups == nil {
		lookups = NewLookupResolver()
	}
	if err := effective.resolveLookups(lookups.clone()); err != nil {
		return nil, err
	}
	return effective, nil
}

func (c *fileConfig) options(registry *PluginRegistry) (Options, error) {
	if c == nil || c.empty() {
		return DefaultOptions(), nil
	}
	if err := c.registerCustomLevels(); err != nil {
		return Options{}, err
	}
	if err := c.validateAsyncLoggerConfig(); err != nil {
		return Options{}, err
	}
	if registry == nil {
		registry = DefaultPluginRegistry()
	}
	filters, err := c.buildFilters(registry)
	if err != nil {
		return Options{}, err
	}
	appenders, err := c.buildAppenders(filters, registry)
	if err != nil {
		return Options{}, err
	}
	globalFilters, err := resolveFilters(filters, c.filterRefs())
	if err != nil {
		_ = closeAppenderList(appenders)
		return Options{}, err
	}
	rootFilters, err := resolveFilters(filters, c.Root.filterRefs())
	if err != nil {
		_ = closeAppenderList(appenders)
		return Options{}, fmt.Errorf("goark-log: root: %w", err)
	}
	rootLevel, err := ParseLevel(c.Root.Level)
	if err != nil {
		_ = closeAppenderList(appenders)
		return Options{}, err
	}
	options := Options{
		Appenders: appenders,
		Filters:   globalFilters,
		Async:     c.asyncLoggerOptions(),
		Root: RootLogger{
			Level:        rootLevel,
			AppenderRefs: c.Root.refs(),
			Filters:      rootFilters,
		},
	}
	options.Root.AppenderRefControls, err = c.Root.appenderRefControls(filters)
	if err != nil {
		_ = closeAppenderList(appenders)
		return Options{}, fmt.Errorf("goark-log: root: %w", err)
	}
	loggerNames := sortedLoggerNames(c.Loggers)
	for _, name := range loggerNames {
		loggerConfig := c.Loggers[name]
		var level *slog.Level
		if strings.TrimSpace(loggerConfig.Level) != "" {
			parsed, err := ParseLevel(loggerConfig.Level)
			if err != nil {
				_ = closeAppenderList(appenders)
				return Options{}, fmt.Errorf("goark-log: logger %q: %w", name, err)
			}
			level = &parsed
		}
		rule := LoggerRule{
			Name:         name,
			Level:        level,
			AppenderRefs: loggerConfig.refs(),
		}
		rule.AppenderRefControls, err = loggerConfig.appenderRefControls(filters)
		if err != nil {
			_ = closeAppenderList(appenders)
			return Options{}, fmt.Errorf("goark-log: logger %q: %w", name, err)
		}
		rule.Filters, err = resolveFilters(filters, loggerConfig.filterRefs())
		if err != nil {
			_ = closeAppenderList(appenders)
			return Options{}, fmt.Errorf("goark-log: logger %q: %w", name, err)
		}
		if loggerConfig.Additivity != nil {
			rule.Additivity = *loggerConfig.Additivity
			rule.AdditivitySet = true
		}
		options.Loggers = append(options.Loggers, rule)
	}
	return options, nil
}

func (c *fileConfig) validateAsyncLoggerConfig() error {
	used := 0
	for _, candidate := range []asyncLoggerConfig{c.AsyncLogger, c.AsyncLoggerKebab, c.Async} {
		if !candidate.empty() {
			used++
		}
	}
	if used > 1 {
		return fmt.Errorf("goark-log: config must use only one of asyncLogger, async-logger, or async")
	}
	config := c.asyncLoggerConfig()
	if config.empty() {
		return nil
	}
	if config.queueSize() < 0 {
		return fmt.Errorf("goark-log: asyncLogger queueSize must be >= 0")
	}
	if config.batchSize() < 0 {
		return fmt.Errorf("goark-log: asyncLogger batchSize must be >= 0")
	}
	if _, err := ParseAsyncOverflowStrategy(config.overflowStrategy()); err != nil {
		return fmt.Errorf("goark-log: asyncLogger: %w", err)
	}
	if _, err := ParseAsyncWaitStrategy(config.waitStrategy()); err != nil {
		return fmt.Errorf("goark-log: asyncLogger: %w", err)
	}
	if err := validateAsyncWaitOptions(config.waitOptions()); err != nil {
		return fmt.Errorf("goark-log: asyncLogger: %w", err)
	}
	return nil
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

func (c *fileConfig) resolveLookups(lookups *LookupResolver) error {
	if c == nil {
		return nil
	}
	if lookups == nil {
		lookups = NewLookupResolver()
	}
	var err error
	if c.Status, err = resolveStringLookup(lookups, c.Status); err != nil {
		return fmt.Errorf("goark-log: status: %w", err)
	}
	if c.MonitorInterval, err = resolveStringLookup(lookups, c.MonitorInterval); err != nil {
		return fmt.Errorf("goark-log: monitorInterval: %w", err)
	}
	if c.MonitorKebab, err = resolveStringLookup(lookups, c.MonitorKebab); err != nil {
		return fmt.Errorf("goark-log: monitor-interval: %w", err)
	}
	if err := c.resolveProperties(lookups); err != nil {
		return err
	}
	if err := c.resolveCustomLevelLookups(lookups); err != nil {
		return err
	}
	c.FilterRefs, err = resolveStringListLookups(lookups, c.FilterRefs)
	if err != nil {
		return fmt.Errorf("goark-log: filterRefs: %w", err)
	}
	c.FilterRefsKebab, err = resolveStringListLookups(lookups, c.FilterRefsKebab)
	if err != nil {
		return fmt.Errorf("goark-log: filter-refs: %w", err)
	}
	if err := c.AsyncLogger.resolveLookups(lookups); err != nil {
		return fmt.Errorf("goark-log: asyncLogger: %w", err)
	}
	if err := c.AsyncLoggerKebab.resolveLookups(lookups); err != nil {
		return fmt.Errorf("goark-log: async-logger: %w", err)
	}
	if err := c.Async.resolveLookups(lookups); err != nil {
		return fmt.Errorf("goark-log: async: %w", err)
	}
	for name, spec := range c.Appenders {
		if err := spec.resolveLookups(lookups); err != nil {
			return fmt.Errorf("goark-log: appender %q: %w", name, err)
		}
		c.Appenders[name] = spec
	}
	for name, spec := range c.Filters {
		if err := spec.resolveLookups(lookups); err != nil {
			return fmt.Errorf("goark-log: filter %q: %w", name, err)
		}
		c.Filters[name] = spec
	}
	if err := c.Root.resolveLookups(lookups); err != nil {
		return fmt.Errorf("goark-log: root: %w", err)
	}
	for name, spec := range c.Loggers {
		if err := spec.resolveLookups(lookups); err != nil {
			return fmt.Errorf("goark-log: logger %q: %w", name, err)
		}
		c.Loggers[name] = spec
	}
	return nil
}

func (c *fileConfig) resolveProperties(lookups *LookupResolver) error {
	if len(c.Properties) == 0 {
		return nil
	}
	resolved := make(map[string]string, len(c.Properties))
	for key, value := range c.Properties {
		key = strings.TrimSpace(key)
		if key == "" {
			return fmt.Errorf("goark-log: property name is empty")
		}
		out, err := resolveStringLookup(lookups, value)
		if err != nil {
			return fmt.Errorf("goark-log: property %q: %w", key, err)
		}
		resolved[key] = out
	}
	c.Properties = resolved
	lookups.Register("prop", func(key string) (string, bool) {
		value, ok := resolved[strings.TrimSpace(key)]
		return value, ok
	})
	lookups.Register("property", func(key string) (string, bool) {
		value, ok := resolved[strings.TrimSpace(key)]
		return value, ok
	})
	return nil
}

func (c *appenderConfig) resolveLookups(lookups *LookupResolver) error {
	var err error
	if c.Type, err = resolveStringLookup(lookups, c.Type); err != nil {
		return fmt.Errorf("type: %w", err)
	}
	if c.Target, err = resolveStringLookup(lookups, c.Target); err != nil {
		return fmt.Errorf("target: %w", err)
	}
	if c.URL, err = resolveStringLookup(lookups, c.URL); err != nil {
		return fmt.Errorf("url: %w", err)
	}
	if c.Method, err = resolveStringLookup(lookups, c.Method); err != nil {
		return fmt.Errorf("method: %w", err)
	}
	if c.Address, err = resolveStringLookup(lookups, c.Address); err != nil {
		return fmt.Errorf("address: %w", err)
	}
	if c.Network, err = resolveStringLookup(lookups, c.Network); err != nil {
		return fmt.Errorf("network: %w", err)
	}
	if c.Facility, err = resolveStringLookup(lookups, c.Facility); err != nil {
		return fmt.Errorf("facility: %w", err)
	}
	if c.AppName, err = resolveStringLookup(lookups, c.AppName); err != nil {
		return fmt.Errorf("appName: %w", err)
	}
	if c.AppNameKebab, err = resolveStringLookup(lookups, c.AppNameKebab); err != nil {
		return fmt.Errorf("app-name: %w", err)
	}
	if c.ConnectTimeout, err = resolveStringLookup(lookups, c.ConnectTimeout); err != nil {
		return fmt.Errorf("connectTimeout: %w", err)
	}
	if c.ConnectTimeoutKebab, err = resolveStringLookup(lookups, c.ConnectTimeoutKebab); err != nil {
		return fmt.Errorf("connect-timeout: %w", err)
	}
	if c.WriteTimeout, err = resolveStringLookup(lookups, c.WriteTimeout); err != nil {
		return fmt.Errorf("writeTimeout: %w", err)
	}
	if c.WriteTimeoutKebab, err = resolveStringLookup(lookups, c.WriteTimeoutKebab); err != nil {
		return fmt.Errorf("write-timeout: %w", err)
	}
	if c.FileName, err = resolveStringLookup(lookups, c.FileName); err != nil {
		return fmt.Errorf("fileName: %w", err)
	}
	if c.FileNameKebab, err = resolveStringLookup(lookups, c.FileNameKebab); err != nil {
		return fmt.Errorf("file-name: %w", err)
	}
	if c.Path, err = resolveStringLookup(lookups, c.Path); err != nil {
		return fmt.Errorf("path: %w", err)
	}
	if err := c.Layout.resolveLookups(lookups); err != nil {
		return fmt.Errorf("layout: %w", err)
	}
	if err := c.Rolling.resolveLookups(lookups); err != nil {
		return fmt.Errorf("rolling: %w", err)
	}
	if c.BufferSize, err = resolveStringLookup(lookups, c.BufferSize); err != nil {
		return fmt.Errorf("bufferSize: %w", err)
	}
	if c.BufferSizeKebab, err = resolveStringLookup(lookups, c.BufferSizeKebab); err != nil {
		return fmt.Errorf("buffer-size: %w", err)
	}
	if c.WaitStrategy, err = resolveStringLookup(lookups, c.WaitStrategy); err != nil {
		return fmt.Errorf("waitStrategy: %w", err)
	}
	if c.WaitStrategyKebab, err = resolveStringLookup(lookups, c.WaitStrategyKebab); err != nil {
		return fmt.Errorf("wait-strategy: %w", err)
	}
	if c.SleepTime, err = resolveStringLookup(lookups, c.SleepTime); err != nil {
		return fmt.Errorf("sleepTime: %w", err)
	}
	if c.SleepTimeKebab, err = resolveStringLookup(lookups, c.SleepTimeKebab); err != nil {
		return fmt.Errorf("sleep-time: %w", err)
	}
	if c.Timeout, err = resolveStringLookup(lookups, c.Timeout); err != nil {
		return fmt.Errorf("timeout: %w", err)
	}
	if c.AppenderRefs, err = c.AppenderRefs.resolveLookups(lookups); err != nil {
		return fmt.Errorf("appenderRefs: %w", err)
	}
	if c.AppenderRefsKebab, err = c.AppenderRefsKebab.resolveLookups(lookups); err != nil {
		return fmt.Errorf("appender-refs: %w", err)
	}
	if c.Refs, err = c.Refs.resolveLookups(lookups); err != nil {
		return fmt.Errorf("refs: %w", err)
	}
	if c.Filters, err = resolveStringListLookups(lookups, c.Filters); err != nil {
		return fmt.Errorf("filters: %w", err)
	}
	if c.FilterRefs, err = resolveStringListLookups(lookups, c.FilterRefs); err != nil {
		return fmt.Errorf("filterRefs: %w", err)
	}
	if c.FilterRefsKebab, err = resolveStringListLookups(lookups, c.FilterRefsKebab); err != nil {
		return fmt.Errorf("filter-refs: %w", err)
	}
	return nil
}

func (c *layoutConfig) resolveLookups(lookups *LookupResolver) error {
	var err error
	if c.Type, err = resolveStringLookup(lookups, c.Type); err != nil {
		return fmt.Errorf("type: %w", err)
	}
	if c.Pattern, err = resolveStringLookup(lookups, c.Pattern); err != nil {
		return fmt.Errorf("pattern: %w", err)
	}
	if c.EventTemplate, err = resolveStringLookup(lookups, c.EventTemplate); err != nil {
		return fmt.Errorf("eventTemplate: %w", err)
	}
	if c.EventTemplateKebab, err = resolveStringLookup(lookups, c.EventTemplateKebab); err != nil {
		return fmt.Errorf("event-template: %w", err)
	}
	if c.EventTemplateURI, err = resolveStringLookup(lookups, c.EventTemplateURI); err != nil {
		return fmt.Errorf("eventTemplateUri: %w", err)
	}
	if c.EventTemplateURIKebab, err = resolveStringLookup(lookups, c.EventTemplateURIKebab); err != nil {
		return fmt.Errorf("event-template-uri: %w", err)
	}
	if c.EventTemplatePath, err = resolveStringLookup(lookups, c.EventTemplatePath); err != nil {
		return fmt.Errorf("eventTemplatePath: %w", err)
	}
	if c.EventTemplatePathKebab, err = resolveStringLookup(lookups, c.EventTemplatePathKebab); err != nil {
		return fmt.Errorf("event-template-path: %w", err)
	}
	if c.Header, err = resolveStringLookup(lookups, c.Header); err != nil {
		return fmt.Errorf("header: %w", err)
	}
	if c.Footer, err = resolveStringLookup(lookups, c.Footer); err != nil {
		return fmt.Errorf("footer: %w", err)
	}
	return nil
}

func (c *rollingConfig) resolveLookups(lookups *LookupResolver) error {
	var err error
	if c.FilePattern, err = resolveStringLookup(lookups, c.FilePattern); err != nil {
		return fmt.Errorf("filePattern: %w", err)
	}
	if c.FilePatternKebab, err = resolveStringLookup(lookups, c.FilePatternKebab); err != nil {
		return fmt.Errorf("file-pattern: %w", err)
	}
	if c.MaxSize, err = resolveStringLookup(lookups, c.MaxSize); err != nil {
		return fmt.Errorf("maxSize: %w", err)
	}
	if c.MaxSizeKebab, err = resolveStringLookup(lookups, c.MaxSizeKebab); err != nil {
		return fmt.Errorf("max-size: %w", err)
	}
	if c.Interval, err = resolveStringLookup(lookups, c.Interval); err != nil {
		return fmt.Errorf("interval: %w", err)
	}
	if c.Cron, err = resolveStringLookup(lookups, c.Cron); err != nil {
		return fmt.Errorf("cron: %w", err)
	}
	if c.CronSchedule, err = resolveStringLookup(lookups, c.CronSchedule); err != nil {
		return fmt.Errorf("cronSchedule: %w", err)
	}
	if c.CronScheduleKebab, err = resolveStringLookup(lookups, c.CronScheduleKebab); err != nil {
		return fmt.Errorf("cron-schedule: %w", err)
	}
	if c.MaxAge, err = resolveStringLookup(lookups, c.MaxAge); err != nil {
		return fmt.Errorf("maxAge: %w", err)
	}
	if c.MaxAgeKebab, err = resolveStringLookup(lookups, c.MaxAgeKebab); err != nil {
		return fmt.Errorf("max-age: %w", err)
	}
	if err := c.Policies.resolveLookups(lookups); err != nil {
		return fmt.Errorf("policies: %w", err)
	}
	if err := c.Strategy.resolveLookups(lookups); err != nil {
		return fmt.Errorf("strategy: %w", err)
	}
	return nil
}

func (c *rollingPoliciesConfig) resolveLookups(lookups *LookupResolver) error {
	policies := []*rollingSizePolicyConfig{
		&c.Size,
		&c.SizeKebab,
		&c.SizeBasedTriggeringPolicy,
		&c.SizeBasedTriggeringPolicyXML,
	}
	for _, policy := range policies {
		if err := policy.resolveLookups(lookups); err != nil {
			return err
		}
	}
	timePolicies := []*rollingTimePolicyConfig{
		&c.Time,
		&c.TimeKebab,
		&c.TimeBasedTriggeringPolicy,
		&c.TimeBasedTriggeringPolicyXML,
	}
	for _, policy := range timePolicies {
		if err := policy.resolveLookups(lookups); err != nil {
			return err
		}
	}
	cronPolicies := []*rollingCronPolicyConfig{
		&c.Cron,
		&c.CronKebab,
		&c.CronTriggeringPolicy,
		&c.CronTriggeringPolicyXML,
	}
	for _, policy := range cronPolicies {
		if err := policy.resolveLookups(lookups); err != nil {
			return err
		}
	}
	return nil
}

func (c *rollingSizePolicyConfig) resolveLookups(lookups *LookupResolver) error {
	var err error
	if c.Size, err = resolveStringLookup(lookups, c.Size); err != nil {
		return fmt.Errorf("size: %w", err)
	}
	if c.MaxSize, err = resolveStringLookup(lookups, c.MaxSize); err != nil {
		return fmt.Errorf("maxSize: %w", err)
	}
	if c.MaxSizeKebab, err = resolveStringLookup(lookups, c.MaxSizeKebab); err != nil {
		return fmt.Errorf("max-size: %w", err)
	}
	return nil
}

func (c *rollingTimePolicyConfig) resolveLookups(lookups *LookupResolver) error {
	var err error
	if c.Interval, err = resolveStringLookup(lookups, c.Interval); err != nil {
		return fmt.Errorf("interval: %w", err)
	}
	if c.Every, err = resolveStringLookup(lookups, c.Every); err != nil {
		return fmt.Errorf("every: %w", err)
	}
	if c.Unit, err = resolveStringLookup(lookups, c.Unit); err != nil {
		return fmt.Errorf("unit: %w", err)
	}
	return nil
}

func (c *rollingCronPolicyConfig) resolveLookups(lookups *LookupResolver) error {
	var err error
	if c.Schedule, err = resolveStringLookup(lookups, c.Schedule); err != nil {
		return fmt.Errorf("schedule: %w", err)
	}
	if c.Cron, err = resolveStringLookup(lookups, c.Cron); err != nil {
		return fmt.Errorf("cron: %w", err)
	}
	if c.CronSchedule, err = resolveStringLookup(lookups, c.CronSchedule); err != nil {
		return fmt.Errorf("cronSchedule: %w", err)
	}
	if c.CronKebab, err = resolveStringLookup(lookups, c.CronKebab); err != nil {
		return fmt.Errorf("cron-schedule: %w", err)
	}
	return nil
}

func (c *rollingStrategyConfig) resolveLookups(lookups *LookupResolver) error {
	var err error
	if c.MaxAge, err = resolveStringLookup(lookups, c.MaxAge); err != nil {
		return fmt.Errorf("maxAge: %w", err)
	}
	if c.MaxAgeKebab, err = resolveStringLookup(lookups, c.MaxAgeKebab); err != nil {
		return fmt.Errorf("max-age: %w", err)
	}
	if c.FileIndex, err = resolveStringLookup(lookups, c.FileIndex); err != nil {
		return fmt.Errorf("fileIndex: %w", err)
	}
	if c.FileIndexKebab, err = resolveStringLookup(lookups, c.FileIndexKebab); err != nil {
		return fmt.Errorf("file-index: %w", err)
	}
	if err := c.Delete.resolveLookups(lookups); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	for index := range c.DeleteActions {
		if err := c.DeleteActions[index].resolveLookups(lookups); err != nil {
			return fmt.Errorf("deleteActions[%d]: %w", index, err)
		}
	}
	for index := range c.DeleteActionsKebab {
		if err := c.DeleteActionsKebab[index].resolveLookups(lookups); err != nil {
			return fmt.Errorf("delete-actions[%d]: %w", index, err)
		}
	}
	return nil
}

func (c *rollingDeleteActionConfig) resolveLookups(lookups *LookupResolver) error {
	var err error
	if c.BasePath, err = resolveStringLookup(lookups, c.BasePath); err != nil {
		return fmt.Errorf("basePath: %w", err)
	}
	if c.BasePathKebab, err = resolveStringLookup(lookups, c.BasePathKebab); err != nil {
		return fmt.Errorf("base-path: %w", err)
	}
	if c.Glob, err = resolveStringLookup(lookups, c.Glob); err != nil {
		return fmt.Errorf("glob: %w", err)
	}
	if c.Age, err = resolveStringLookup(lookups, c.Age); err != nil {
		return fmt.Errorf("age: %w", err)
	}
	if c.MaxSize, err = resolveStringLookup(lookups, c.MaxSize); err != nil {
		return fmt.Errorf("maxSize: %w", err)
	}
	if c.MaxSizeKebab, err = resolveStringLookup(lookups, c.MaxSizeKebab); err != nil {
		return fmt.Errorf("max-size: %w", err)
	}
	if err := c.IfFileName.resolveLookups(lookups); err != nil {
		return fmt.Errorf("ifFileName: %w", err)
	}
	if err := c.IfFileNameKebab.resolveLookups(lookups); err != nil {
		return fmt.Errorf("if-file-name: %w", err)
	}
	if err := c.IfLastModified.resolveLookups(lookups); err != nil {
		return fmt.Errorf("ifLastModified: %w", err)
	}
	if err := c.IfLastModifiedKebab.resolveLookups(lookups); err != nil {
		return fmt.Errorf("if-last-modified: %w", err)
	}
	if err := c.IfAccumulatedFileSize.resolveLookups(lookups); err != nil {
		return fmt.Errorf("ifAccumulatedFileSize: %w", err)
	}
	if err := c.IfAccumulatedFileSizeKebab.resolveLookups(lookups); err != nil {
		return fmt.Errorf("if-accumulated-file-size: %w", err)
	}
	return nil
}

func (c *rollingDeleteFileNameConfig) resolveLookups(lookups *LookupResolver) error {
	var err error
	if c.Glob, err = resolveStringLookup(lookups, c.Glob); err != nil {
		return fmt.Errorf("glob: %w", err)
	}
	return nil
}

func (c *rollingDeleteLastModifiedConfig) resolveLookups(lookups *LookupResolver) error {
	var err error
	if c.Age, err = resolveStringLookup(lookups, c.Age); err != nil {
		return fmt.Errorf("age: %w", err)
	}
	return nil
}

func (c *rollingDeleteAccumulatedSizeConfig) resolveLookups(lookups *LookupResolver) error {
	var err error
	if c.Exceeds, err = resolveStringLookup(lookups, c.Exceeds); err != nil {
		return fmt.Errorf("exceeds: %w", err)
	}
	return nil
}

func (c *loggerConfig) resolveLookups(lookups *LookupResolver) error {
	var err error
	if c.Level, err = resolveStringLookup(lookups, c.Level); err != nil {
		return fmt.Errorf("level: %w", err)
	}
	if c.AppenderRefs, err = c.AppenderRefs.resolveLookups(lookups); err != nil {
		return fmt.Errorf("appenderRefs: %w", err)
	}
	if c.AppenderRefsKebab, err = c.AppenderRefsKebab.resolveLookups(lookups); err != nil {
		return fmt.Errorf("appender-refs: %w", err)
	}
	if c.Refs, err = c.Refs.resolveLookups(lookups); err != nil {
		return fmt.Errorf("refs: %w", err)
	}
	if c.Filters, err = resolveStringListLookups(lookups, c.Filters); err != nil {
		return fmt.Errorf("filters: %w", err)
	}
	if c.FilterRefs, err = resolveStringListLookups(lookups, c.FilterRefs); err != nil {
		return fmt.Errorf("filterRefs: %w", err)
	}
	if c.FilterRefsKebab, err = resolveStringListLookups(lookups, c.FilterRefsKebab); err != nil {
		return fmt.Errorf("filter-refs: %w", err)
	}
	return nil
}

func (c *filterConfig) resolveLookups(lookups *LookupResolver) error {
	var err error
	if c.Type, err = resolveStringLookup(lookups, c.Type); err != nil {
		return fmt.Errorf("type: %w", err)
	}
	if c.Level, err = resolveStringLookup(lookups, c.Level); err != nil {
		return fmt.Errorf("level: %w", err)
	}
	if c.MinLevel, err = resolveStringLookup(lookups, c.MinLevel); err != nil {
		return fmt.Errorf("minLevel: %w", err)
	}
	if c.MinLevelKebab, err = resolveStringLookup(lookups, c.MinLevelKebab); err != nil {
		return fmt.Errorf("min-level: %w", err)
	}
	if c.MaxLevel, err = resolveStringLookup(lookups, c.MaxLevel); err != nil {
		return fmt.Errorf("maxLevel: %w", err)
	}
	if c.MaxLevelKebab, err = resolveStringLookup(lookups, c.MaxLevelKebab); err != nil {
		return fmt.Errorf("max-level: %w", err)
	}
	if c.Marker, err = resolveStringLookup(lookups, c.Marker); err != nil {
		return fmt.Errorf("marker: %w", err)
	}
	if c.Text, err = resolveStringLookup(lookups, c.Text); err != nil {
		return fmt.Errorf("text: %w", err)
	}
	if c.Operator, err = resolveStringLookup(lookups, c.Operator); err != nil {
		return fmt.Errorf("operator: %w", err)
	}
	if c.Start, err = resolveStringLookup(lookups, c.Start); err != nil {
		return fmt.Errorf("start: %w", err)
	}
	if c.End, err = resolveStringLookup(lookups, c.End); err != nil {
		return fmt.Errorf("end: %w", err)
	}
	if c.Timezone, err = resolveStringLookup(lookups, c.Timezone); err != nil {
		return fmt.Errorf("timezone: %w", err)
	}
	if c.Rate, err = resolveStringLookup(lookups, c.Rate); err != nil {
		return fmt.Errorf("rate: %w", err)
	}
	if c.Field, err = resolveStringLookup(lookups, c.Field); err != nil {
		return fmt.Errorf("field: %w", err)
	}
	if c.Key, err = resolveStringLookup(lookups, c.Key); err != nil {
		return fmt.Errorf("key: %w", err)
	}
	if c.Value, err = resolveStringLookup(lookups, c.Value); err != nil {
		return fmt.Errorf("value: %w", err)
	}
	if c.Values, err = resolveStringMapLookups(lookups, c.Values); err != nil {
		return fmt.Errorf("values: %w", err)
	}
	if c.Thresholds, err = resolveStringMapLookups(lookups, c.Thresholds); err != nil {
		return fmt.Errorf("thresholds: %w", err)
	}
	if c.Filters, err = resolveStringListLookups(lookups, c.Filters); err != nil {
		return fmt.Errorf("filters: %w", err)
	}
	if c.FilterRefs, err = resolveStringListLookups(lookups, c.FilterRefs); err != nil {
		return fmt.Errorf("filterRefs: %w", err)
	}
	if c.FilterRefsKebab, err = resolveStringListLookups(lookups, c.FilterRefsKebab); err != nil {
		return fmt.Errorf("filter-refs: %w", err)
	}
	if c.KeyValuePair, err = resolveKeyValuePairLookups(lookups, c.KeyValuePair); err != nil {
		return fmt.Errorf("KeyValuePair: %w", err)
	}
	if c.KeyValuePairs, err = resolveKeyValuePairLookups(lookups, c.KeyValuePairs); err != nil {
		return fmt.Errorf("keyValuePairs: %w", err)
	}
	if c.KeyValuePairsKebab, err = resolveKeyValuePairLookups(lookups, c.KeyValuePairsKebab); err != nil {
		return fmt.Errorf("key-value-pairs: %w", err)
	}
	if c.DefaultThreshold, err = resolveStringLookup(lookups, c.DefaultThreshold); err != nil {
		return fmt.Errorf("defaultThreshold: %w", err)
	}
	if c.DefaultKebab, err = resolveStringLookup(lookups, c.DefaultKebab); err != nil {
		return fmt.Errorf("default-threshold: %w", err)
	}
	if c.Pattern, err = resolveStringLookup(lookups, c.Pattern); err != nil {
		return fmt.Errorf("pattern: %w", err)
	}
	if c.OnMatch, err = resolveStringLookup(lookups, c.OnMatch); err != nil {
		return fmt.Errorf("onMatch: %w", err)
	}
	if c.OnMatchKebab, err = resolveStringLookup(lookups, c.OnMatchKebab); err != nil {
		return fmt.Errorf("on-match: %w", err)
	}
	if c.OnMismatch, err = resolveStringLookup(lookups, c.OnMismatch); err != nil {
		return fmt.Errorf("onMismatch: %w", err)
	}
	if c.OnMismatchKebab, err = resolveStringLookup(lookups, c.OnMismatchKebab); err != nil {
		return fmt.Errorf("on-mismatch: %w", err)
	}
	return nil
}

func (c *asyncLoggerConfig) resolveLookups(lookups *LookupResolver) error {
	var err error
	if c.OverflowStrategy, err = resolveStringLookup(lookups, c.OverflowStrategy); err != nil {
		return fmt.Errorf("overflowStrategy: %w", err)
	}
	if c.OverflowStrategyKebab, err = resolveStringLookup(lookups, c.OverflowStrategyKebab); err != nil {
		return fmt.Errorf("overflow-strategy: %w", err)
	}
	if c.WaitStrategy, err = resolveStringLookup(lookups, c.WaitStrategy); err != nil {
		return fmt.Errorf("waitStrategy: %w", err)
	}
	if c.WaitStrategyKebab, err = resolveStringLookup(lookups, c.WaitStrategyKebab); err != nil {
		return fmt.Errorf("wait-strategy: %w", err)
	}
	if c.SleepTime, err = resolveStringLookup(lookups, c.SleepTime); err != nil {
		return fmt.Errorf("sleepTime: %w", err)
	}
	if c.SleepTimeKebab, err = resolveStringLookup(lookups, c.SleepTimeKebab); err != nil {
		return fmt.Errorf("sleep-time: %w", err)
	}
	if c.Timeout, err = resolveStringLookup(lookups, c.Timeout); err != nil {
		return fmt.Errorf("timeout: %w", err)
	}
	return nil
}

func resolveStringLookup(lookups *LookupResolver, value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return value, nil
	}
	return lookups.Resolve(value)
}

func resolveStringListLookups(lookups *LookupResolver, values []string) ([]string, error) {
	if len(values) == 0 {
		return values, nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		resolved, err := resolveStringLookup(lookups, value)
		if err != nil {
			return nil, err
		}
		out = append(out, resolved)
	}
	return out, nil
}

func resolveStringMapLookups(lookups *LookupResolver, values map[string]string) (map[string]string, error) {
	if len(values) == 0 {
		return values, nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		resolvedKey, err := resolveStringLookup(lookups, key)
		if err != nil {
			return nil, err
		}
		resolvedValue, err := resolveStringLookup(lookups, value)
		if err != nil {
			return nil, err
		}
		out[resolvedKey] = resolvedValue
	}
	return out, nil
}

func resolveKeyValuePairLookups(lookups *LookupResolver, values []keyValuePairConfig) ([]keyValuePairConfig, error) {
	if len(values) == 0 {
		return values, nil
	}
	out := make([]keyValuePairConfig, 0, len(values))
	for index, value := range values {
		resolved, err := value.resolveLookups(lookups)
		if err != nil {
			return nil, fmt.Errorf("%d: %w", index, err)
		}
		if strings.TrimSpace(resolved.Key) == "" {
			return nil, fmt.Errorf("%d: key is empty", index)
		}
		out = append(out, resolved)
	}
	return out, nil
}

func (c keyValuePairConfig) resolveLookups(lookups *LookupResolver) (keyValuePairConfig, error) {
	var err error
	if c.Key, err = resolveStringLookup(lookups, c.Key); err != nil {
		return keyValuePairConfig{}, fmt.Errorf("key: %w", err)
	}
	if c.Value, err = resolveStringLookup(lookups, c.Value); err != nil {
		return keyValuePairConfig{}, fmt.Errorf("value: %w", err)
	}
	return c, nil
}

func (c *fileConfig) resolveCustomLevelLookups(lookups *LookupResolver) error {
	if c == nil {
		return nil
	}
	resolved, err := resolveStringMapLookups(lookups, c.CustomLevels)
	if err != nil {
		return fmt.Errorf("goark-log: customLevels: %w", err)
	}
	c.CustomLevels = resolved
	resolvedKebab, err := resolveStringMapLookups(lookups, c.CustomLevelsKebab)
	if err != nil {
		return fmt.Errorf("goark-log: custom-levels: %w", err)
	}
	c.CustomLevelsKebab = resolvedKebab
	return nil
}

func (c *fileConfig) registerCustomLevels() error {
	levels := mergeStringMaps(copyStringMap(c.CustomLevels), c.CustomLevelsKebab)
	for name, value := range levels {
		parsed, err := parseCustomLevelValue(value)
		if err != nil {
			return fmt.Errorf("goark-log: custom level %q: %w", name, err)
		}
		if err := RegisterLevel(name, slog.Level(parsed)); err != nil {
			return fmt.Errorf("goark-log: custom level %q: %w", name, err)
		}
	}
	return nil
}

func parseCustomLevelValue(value string) (int, error) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("value %q is invalid", value)
	}
	return parsed, nil
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
		c.Additivity == nil
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

func (c *fileConfig) asyncLoggerOptions() AsyncLoggerOptions {
	config := c.asyncLoggerConfig()
	if config.empty() {
		return AsyncLoggerOptions{}
	}
	options := AsyncLoggerOptions{
		QueueSize:        config.queueSize(),
		BatchSize:        config.batchSize(),
		OverflowStrategy: AsyncOverflowStrategy(config.overflowStrategy()),
		WaitStrategy:     AsyncWaitStrategy(config.waitStrategy()),
		WaitOptions:      config.waitOptions(),
		IncludeLocation:  config.includeLocation(),
	}
	if config.Enabled != nil {
		options.Enabled = *config.Enabled
	}
	return options
}

func (c *fileConfig) asyncLoggerConfig() asyncLoggerConfig {
	for _, candidate := range []asyncLoggerConfig{c.AsyncLogger, c.AsyncLoggerKebab, c.Async} {
		if !candidate.empty() {
			return candidate
		}
	}
	return asyncLoggerConfig{}
}

func (c asyncLoggerConfig) queueSize() int {
	if c.QueueSize != 0 {
		return c.QueueSize
	}
	return c.QueueSizeKebab
}

func (c asyncLoggerConfig) batchSize() int {
	if c.BatchSize != 0 {
		return c.BatchSize
	}
	return c.BatchSizeKebab
}

func (c asyncLoggerConfig) overflowStrategy() string {
	return firstNonBlank(c.OverflowStrategy, c.OverflowStrategyKebab)
}

func (c asyncLoggerConfig) waitStrategy() string {
	return firstNonBlank(c.WaitStrategy, c.WaitStrategyKebab)
}

func (c asyncLoggerConfig) waitOptions() AsyncWaitOptions {
	return AsyncWaitOptions{
		Retries:   firstNonZero(c.WaitRetries, c.WaitRetriesKebab),
		SleepTime: parseOptionalDuration(firstNonBlank(c.SleepTime, c.SleepTimeKebab)),
		Timeout:   parseOptionalDuration(c.Timeout),
	}
}

func (c asyncLoggerConfig) includeLocation() bool {
	if c.IncludeLocation != nil {
		return *c.IncludeLocation
	}
	if c.IncludeLocationKebab != nil {
		return *c.IncludeLocationKebab
	}
	return false
}

func (c *fileConfig) buildAppenders(filters map[string]Filter, registry *PluginRegistry) ([]Appender, error) {
	if len(c.Appenders) == 0 {
		return DefaultOptions().Appenders, nil
	}
	appenderNames := sortedAppenderNames(c.Appenders)
	built := make(map[string]Appender, len(c.Appenders))
	appenders := make([]Appender, 0, len(c.Appenders))
	asyncNames := make([]string, 0)
	for _, name := range appenderNames {
		spec := c.Appenders[name]
		if normalizeKind(spec.Type) == "async" {
			asyncNames = append(asyncNames, name)
			continue
		}
		appender, err := buildConcreteAppender(name, spec, filters, registry)
		if err != nil {
			_ = closeAppenderList(appenders)
			return nil, err
		}
		built[name] = appender
		appenders = append(appenders, appender)
	}
	for _, name := range asyncNames {
		appender, err := buildAsyncAppender(name, c.Appenders[name], built, filters, registry)
		if err != nil {
			_ = closeAppenderList(appenders)
			return nil, err
		}
		built[name] = appender
		appenders = append(appenders, appender)
	}
	return appenders, nil
}

func buildConcreteAppender(name string, spec appenderConfig, filters map[string]Filter, registry *PluginRegistry) (Appender, error) {
	layout, err := buildLayout(spec.Layout, registry)
	if err != nil {
		return nil, fmt.Errorf("goark-log: appender %q: %w", name, err)
	}
	kind := normalizeKind(spec.Type)
	if kind == "" {
		return nil, fmt.Errorf("goark-log: appender %q type is empty", name)
	}
	factory, ok := registry.appenderFactory(kind)
	if !ok {
		return nil, fmt.Errorf("goark-log: unsupported appender %q type %q", name, spec.Type)
	}
	appender, err := factory(spec.appenderBuildConfig(name, layout, nil))
	if err != nil {
		return nil, err
	}
	wrapped, err := wrapAppenderFilters(name, appender, spec.filterRefs(), filters)
	if err != nil {
		_ = appender.Close()
		return nil, err
	}
	return wrapped, nil
}

func buildAsyncAppender(name string, spec appenderConfig, built map[string]Appender, filters map[string]Filter, registry *PluginRegistry) (Appender, error) {
	refs := spec.refs()
	controls, err := spec.appenderRefControls(filters)
	if err != nil {
		return nil, fmt.Errorf("goark-log: async appender %q: %w", name, err)
	}
	if len(refs) == 0 && len(controls) == 0 {
		return nil, fmt.Errorf("goark-log: async appender %q requires appenderRefs", name)
	}
	delegates := make([]Appender, 0, len(refs)+len(controls))
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			return nil, fmt.Errorf("goark-log: async appender %q appender ref is empty", name)
		}
		appender, ok := built[ref]
		if !ok {
			return nil, fmt.Errorf("goark-log: async appender %q references unknown or async appender %q", name, ref)
		}
		delegates = append(delegates, appender)
	}
	for _, ref := range controls {
		control, err := newAppenderControl(built, ref)
		if err != nil {
			return nil, fmt.Errorf("goark-log: async appender %q: %w", name, err)
		}
		delegates = append(delegates, controlledAppender{control: control})
	}
	factory, ok := registry.appenderFactory(spec.Type)
	if !ok {
		return nil, fmt.Errorf("goark-log: unsupported appender %q type %q", name, spec.Type)
	}
	appender, err := factory(spec.appenderBuildConfig(name, nil, delegates))
	if err != nil {
		return nil, err
	}
	wrapped, err := wrapAppenderFilters(name, appender, spec.filterRefs(), filters)
	if err != nil {
		_ = appender.Close()
		return nil, err
	}
	return wrapped, nil
}

func buildLayout(config layoutConfig, registry *PluginRegistry) (Layout, error) {
	factory, ok := registry.layoutFactory(config.Type)
	if !ok {
		return nil, fmt.Errorf("unsupported layout type %q", config.Type)
	}
	return factory(LayoutBuildConfig{
		Type:             config.Type,
		Pattern:          config.Pattern,
		EventTemplate:    config.eventTemplate(),
		EventTemplateURI: config.eventTemplateURI(),
		Options:          config.options(),
		Registry:         registry,
	})
}

func (c layoutConfig) eventTemplate() string {
	return firstNonBlank(c.EventTemplate, c.EventTemplateKebab)
}

func (c layoutConfig) eventTemplateURI() string {
	return firstNonBlank(c.EventTemplateURI, c.EventTemplateURIKebab, c.EventTemplatePath, c.EventTemplatePathKebab)
}

func (c layoutConfig) options() LayoutOptions {
	return LayoutOptions{
		Compact:              c.Compact,
		EventEOL:             c.EventEOL || c.EventEOLKebab,
		Complete:             c.Complete,
		IncludeStacktrace:    c.IncludeStacktrace || c.IncludeStacktraceKebab,
		StacktraceAsString:   c.StacktraceAsString || c.StacktraceAsStringKebab,
		PropertiesAsList:     c.PropertiesAsList || c.PropertiesAsListKebab,
		IncludeNullDelimiter: c.IncludeNullDelimiter || c.IncludeNullDelimiterKebab,
		Header:               c.Header,
		Footer:               c.Footer,
	}
}

func configFormat(path string) (string, error) {
	switch strings.ToLower(strings.TrimPrefix(filepathExt(path), ".")) {
	case "yml", "yaml":
		return "yaml", nil
	case "json":
		return "json", nil
	case "xml":
		return "xml", nil
	case "toml":
		return "toml", nil
	case "properties":
		return "properties", nil
	default:
		return "", fmt.Errorf("goark-log: unsupported config file extension for %q", path)
	}
}

func filepathExt(path string) string {
	index := strings.LastIndexByte(path, '.')
	if index < 0 {
		return ""
	}
	return path[index:]
}

func (c appenderConfig) fileName() string {
	for _, value := range []string{c.FileName, c.FileNameKebab, c.Path} {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func (c appenderConfig) refs() []string {
	return c.appenderRefs().strings()
}

func (c appenderConfig) appenderRefs() appenderRefs {
	return firstAppenderRefs(c.AppenderRefs, c.AppenderRefsKebab, c.Refs)
}

func (c appenderConfig) appenderRefControls(filters map[string]Filter) ([]AppenderRef, error) {
	return c.appenderRefs().controls(filters)
}

func (c appenderConfig) filterRefs() []string {
	return firstStringRefs(c.Filters, c.FilterRefs, c.FilterRefsKebab)
}

func (c appenderConfig) queueSize() int {
	if c.QueueSize != 0 {
		return c.QueueSize
	}
	return c.QueueSizeKebab
}

func (c appenderConfig) overflowStrategy() string {
	if strings.TrimSpace(c.OverflowStrategy) != "" {
		return c.OverflowStrategy
	}
	return c.OverflowStrategyKebab
}

func (c appenderConfig) waitStrategy() string {
	return firstNonBlank(c.WaitStrategy, c.WaitStrategyKebab)
}

func (c appenderConfig) waitOptions() AsyncWaitOptions {
	return AsyncWaitOptions{
		Retries:   firstNonZero(c.WaitRetries, c.WaitRetriesKebab),
		SleepTime: parseOptionalDuration(firstNonBlank(c.SleepTime, c.SleepTimeKebab)),
		Timeout:   parseOptionalDuration(c.Timeout),
	}
}

func (c appenderConfig) bufferSize() string {
	return firstNonBlank(c.BufferSize, c.BufferSizeKebab)
}

func (c appenderConfig) flushOnWrite() bool {
	return c.FlushOnWrite || c.FlushOnWriteKebab
}

func (c appenderConfig) appenderBuildConfig(name string, layout Layout, delegates []Appender) AppenderBuildConfig {
	return AppenderBuildConfig{
		Name:             name,
		Type:             c.Type,
		Target:           c.Target,
		URL:              c.URL,
		Method:           c.Method,
		Address:          c.Address,
		Network:          c.Network,
		Facility:         c.Facility,
		AppName:          firstNonBlank(c.AppName, c.AppNameKebab),
		ConnectTimeout:   firstNonBlank(c.ConnectTimeout, c.ConnectTimeoutKebab),
		WriteTimeout:     firstNonBlank(c.WriteTimeout, c.WriteTimeoutKebab),
		FileName:         c.fileName(),
		Layout:           layout,
		AppenderRefs:     c.refs(),
		Delegates:        append([]Appender(nil), delegates...),
		QueueSize:        c.queueSize(),
		OverflowStrategy: c.overflowStrategy(),
		WaitStrategy:     c.waitStrategy(),
		WaitOptions:      c.waitOptions(),
		BufferSize:       c.bufferSize(),
		FlushOnWrite:     c.flushOnWrite(),
		Rolling: RollingBuildConfig{
			FilePattern:     c.Rolling.filePattern(),
			MaxSize:         c.Rolling.maxSize(),
			Interval:        c.Rolling.interval(),
			CronSchedule:    c.Rolling.cronSchedule(),
			TimeModulate:    c.Rolling.timeModulate(),
			OnStartup:       c.Rolling.onStartup(),
			MaxBackups:      c.Rolling.maxBackupsPointer(),
			MaxAge:          c.Rolling.maxAge(),
			FileIndex:       c.Rolling.fileIndex(),
			DirectWrite:     c.Rolling.directWrite(),
			Gzip:            c.Rolling.gzipEnabled(),
			AsyncActions:    c.Rolling.asyncActions(),
			DeleteActions:   c.Rolling.deleteActions(c.fileName()),
			ActionQueueSize: c.Rolling.actionQueueSize(),
		},
	}
}

func (c rollingConfig) filePattern() string {
	return firstNonBlank(c.FilePattern, c.FilePatternKebab)
}

func (c rollingConfig) maxSize() string {
	if value := c.Policies.sizePolicy().size(); value != "" {
		return value
	}
	if strings.TrimSpace(c.MaxSize) != "" {
		return strings.TrimSpace(c.MaxSize)
	}
	return strings.TrimSpace(c.MaxSizeKebab)
}

func (c rollingConfig) interval() string {
	if value := c.Policies.timePolicy().interval(); value != "" {
		return value
	}
	return strings.TrimSpace(c.Interval)
}

func (c rollingConfig) cronSchedule() string {
	if value := c.Policies.cronPolicy().schedule(); value != "" {
		return value
	}
	return firstNonBlank(c.CronSchedule, c.CronScheduleKebab, c.Cron)
}

func (c rollingConfig) timeModulate() *bool {
	return c.Policies.timePolicy().Modulate
}

func (c rollingConfig) maxAge() string {
	return firstNonBlank(c.Strategy.MaxAge, c.Strategy.MaxAgeKebab, c.MaxAge, c.MaxAgeKebab)
}

func (c rollingConfig) fileIndex() string {
	return firstNonBlank(c.Strategy.FileIndex, c.Strategy.FileIndexKebab)
}

func (c rollingConfig) directWrite() bool {
	strategyType := strings.ToLower(strings.TrimSpace(c.Strategy.Type))
	return c.DirectWrite ||
		c.DirectWriteKebab ||
		c.Strategy.DirectWrite ||
		c.Strategy.DirectWriteKebab ||
		strategyType == "directwrite" ||
		strategyType == "direct-write" ||
		strategyType == "directwriterolloverstrategy"
}

func (c rollingConfig) onStartup() bool {
	if enabled := c.Policies.startupPolicy().Enabled; enabled != nil {
		return *enabled
	}
	return c.OnStartup || c.OnStartupKebab
}

func (c rollingConfig) maxBackups() (int, bool) {
	if c.MaxBackups != nil {
		return *c.MaxBackups, true
	}
	if c.MaxBackupsKebab != nil {
		return *c.MaxBackupsKebab, true
	}
	return 0, false
}

func (c rollingConfig) maxBackupsPointer() *int {
	if c.Strategy.Max != nil {
		value := *c.Strategy.Max
		return &value
	}
	if c.Strategy.MaxBackups != nil {
		value := *c.Strategy.MaxBackups
		return &value
	}
	if c.Strategy.MaxBackupsKebab != nil {
		value := *c.Strategy.MaxBackupsKebab
		return &value
	}
	if c.MaxBackups != nil {
		value := *c.MaxBackups
		return &value
	}
	if c.MaxBackupsKebab != nil {
		value := *c.MaxBackupsKebab
		return &value
	}
	return nil
}

func (c rollingConfig) gzipEnabled() bool {
	return c.Gzip || c.Compress || c.Strategy.Compression.Gzip || c.Strategy.Compression.Compress
}

func (c rollingConfig) asyncActions() bool {
	return c.AsyncActions || c.AsyncActionsKebab ||
		c.Strategy.AsyncActions || c.Strategy.AsyncActionsKebab ||
		c.Strategy.Compression.Async || c.Strategy.Delete.Async ||
		containsAsyncDeleteAction(c.Strategy.DeleteActions) ||
		containsAsyncDeleteAction(c.Strategy.DeleteActionsKebab)
}

func (c rollingConfig) actionQueueSize() int {
	if c.ActionQueueSize > 0 {
		return c.ActionQueueSize
	}
	if c.ActionQueueSizeKebab > 0 {
		return c.ActionQueueSizeKebab
	}
	if c.Strategy.ActionQueueSize > 0 {
		return c.Strategy.ActionQueueSize
	}
	return c.Strategy.ActionQueueSizeKebab
}

func (c rollingConfig) deleteActions(fileName string) []RollingDeleteBuildConfig {
	defaultBase := c.defaultDeleteBasePath(fileName)
	configs := make([]rollingDeleteActionConfig, 0, 1+len(c.Strategy.DeleteActions)+len(c.Strategy.DeleteActionsKebab))
	if !c.Strategy.Delete.empty() {
		configs = append(configs, c.Strategy.Delete)
	}
	for _, action := range c.Strategy.DeleteActions {
		if !action.empty() {
			configs = append(configs, action)
		}
	}
	for _, action := range c.Strategy.DeleteActionsKebab {
		if !action.empty() {
			configs = append(configs, action)
		}
	}
	if len(configs) == 0 {
		return nil
	}
	actions := make([]RollingDeleteBuildConfig, 0, len(configs))
	for _, config := range configs {
		action := config.build(defaultBase)
		actions = append(actions, action)
	}
	return actions
}

func (c rollingConfig) defaultDeleteBasePath(fileName string) string {
	if pattern := c.filePattern(); pattern != "" {
		return filepathDir(pattern)
	}
	if strings.TrimSpace(fileName) != "" {
		return filepathDir(fileName)
	}
	return "."
}

func (c rollingPoliciesConfig) sizePolicy() rollingSizePolicyConfig {
	for _, policy := range []rollingSizePolicyConfig{
		c.Size,
		c.SizeKebab,
		c.SizeBasedTriggeringPolicy,
		c.SizeBasedTriggeringPolicyXML,
	} {
		if !policy.empty() {
			return policy
		}
	}
	return rollingSizePolicyConfig{}
}

func (c rollingPoliciesConfig) timePolicy() rollingTimePolicyConfig {
	for _, policy := range []rollingTimePolicyConfig{
		c.Time,
		c.TimeKebab,
		c.TimeBasedTriggeringPolicy,
		c.TimeBasedTriggeringPolicyXML,
	} {
		if !policy.empty() {
			return policy
		}
	}
	return rollingTimePolicyConfig{}
}

func (c rollingPoliciesConfig) cronPolicy() rollingCronPolicyConfig {
	for _, policy := range []rollingCronPolicyConfig{
		c.Cron,
		c.CronKebab,
		c.CronTriggeringPolicy,
		c.CronTriggeringPolicyXML,
	} {
		if !policy.empty() {
			return policy
		}
	}
	return rollingCronPolicyConfig{}
}

func (c rollingPoliciesConfig) startupPolicy() rollingStartupPolicyConfig {
	for _, policy := range []rollingStartupPolicyConfig{
		c.Startup,
		c.StartupKebab,
		c.OnStartupTriggeringPolicy,
		c.OnStartupTriggeringPolicyXML,
	} {
		if policy.Enabled != nil {
			return policy
		}
	}
	return rollingStartupPolicyConfig{}
}

func (c rollingSizePolicyConfig) empty() bool {
	return firstNonBlank(c.Size, c.MaxSize, c.MaxSizeKebab) == ""
}

func (c rollingSizePolicyConfig) size() string {
	return firstNonBlank(c.Size, c.MaxSize, c.MaxSizeKebab)
}

func (c rollingTimePolicyConfig) empty() bool {
	return firstNonBlank(c.Interval, c.Every, c.Unit) == "" && c.Modulate == nil
}

func (c rollingTimePolicyConfig) interval() string {
	if strings.TrimSpace(c.Unit) == "" {
		return firstNonBlank(c.Interval, c.Every)
	}
	return strings.TrimSpace(firstNonBlank(c.Interval, c.Every)) + strings.TrimSpace(c.Unit)
}

func (c rollingCronPolicyConfig) empty() bool {
	return c.schedule() == ""
}

func (c rollingCronPolicyConfig) schedule() string {
	return firstNonBlank(c.Schedule, c.CronSchedule, c.CronKebab, c.Cron)
}

func (c rollingDeleteActionConfig) empty() bool {
	return firstNonBlank(c.BasePath, c.BasePathKebab, c.Glob, c.Age,
		c.IfFileName.Glob, c.IfFileNameKebab.Glob,
		c.IfLastModified.Age, c.IfLastModifiedKebab.Age,
		c.MaxSize, c.MaxSizeKebab,
		c.IfAccumulatedFileSize.Exceeds, c.IfAccumulatedFileSizeKebab.Exceeds) == "" &&
		c.MaxDepth == nil && c.MaxDepthKebab == nil &&
		c.MaxCount == nil && c.MaxCountKebab == nil &&
		c.IfAccumulatedFileCount.Exceeds == 0 && c.IfAccumulatedFileCountKebab.Exceeds == 0
}

func (c rollingDeleteActionConfig) build(defaultBase string) RollingDeleteBuildConfig {
	config := RollingDeleteBuildConfig{
		BasePath: firstNonBlank(c.BasePath, c.BasePathKebab, defaultBase),
		Glob:     firstNonBlank(c.Glob, c.IfFileName.Glob, c.IfFileNameKebab.Glob),
		MaxAge:   firstNonBlank(c.Age, c.IfLastModified.Age, c.IfLastModifiedKebab.Age),
		MaxSize:  firstNonBlank(c.MaxSize, c.MaxSizeKebab, c.IfAccumulatedFileSize.Exceeds, c.IfAccumulatedFileSizeKebab.Exceeds),
	}
	if c.MaxDepth != nil {
		config.MaxDepth = *c.MaxDepth
	} else if c.MaxDepthKebab != nil {
		config.MaxDepth = *c.MaxDepthKebab
	}
	if c.MaxCount != nil {
		config.MaxCount = *c.MaxCount
	} else if c.MaxCountKebab != nil {
		config.MaxCount = *c.MaxCountKebab
	} else if c.IfAccumulatedFileCount.Exceeds > 0 {
		config.MaxCount = c.IfAccumulatedFileCount.Exceeds
	} else if c.IfAccumulatedFileCountKebab.Exceeds > 0 {
		config.MaxCount = c.IfAccumulatedFileCountKebab.Exceeds
	}
	return config
}

func containsAsyncDeleteAction(actions []rollingDeleteActionConfig) bool {
	for _, action := range actions {
		if action.Async {
			return true
		}
	}
	return false
}

func filepathDir(path string) string {
	index := strings.LastIndexAny(path, `/\`)
	if index < 0 {
		return "."
	}
	return path[:index]
}

func (c loggerConfig) refs() []string {
	return c.appenderRefs().strings()
}

func (c loggerConfig) appenderRefs() appenderRefs {
	return firstAppenderRefs(c.AppenderRefs, c.AppenderRefsKebab, c.Refs)
}

func (c loggerConfig) appenderRefControls(filters map[string]Filter) ([]AppenderRef, error) {
	return c.appenderRefs().controls(filters)
}

func (c loggerConfig) filterRefs() []string {
	return firstStringRefs(c.Filters, c.FilterRefs, c.FilterRefsKebab)
}

func (c filterConfig) filterRefs() []string {
	return firstStringRefs(c.Filters, c.FilterRefs, c.FilterRefsKebab)
}

func (c *fileConfig) filterRefs() []string {
	if c == nil {
		return nil
	}
	return firstStringRefs(c.FilterRefs, c.FilterRefsKebab)
}

func (c *fileConfig) buildFilters(registry *PluginRegistry) (map[string]Filter, error) {
	if len(c.Filters) == 0 {
		return nil, nil
	}
	names := sortedFilterNames(c.Filters)
	filters := make(map[string]Filter, len(c.Filters))
	visiting := make(map[string]bool, len(c.Filters))
	for _, name := range names {
		filter, err := c.buildFilter(name, registry, filters, visiting)
		if err != nil {
			return nil, err
		}
		filters[name] = filter
	}
	return filters, nil
}

func (c *fileConfig) buildFilter(name string, registry *PluginRegistry, filters map[string]Filter, visiting map[string]bool) (Filter, error) {
	if filter, ok := filters[name]; ok {
		return filter, nil
	}
	if visiting[name] {
		return nil, fmt.Errorf("goark-log: filter %q has cyclic filterRefs", name)
	}
	spec, ok := c.Filters[name]
	if !ok {
		return nil, fmt.Errorf("goark-log: filter %q is not configured", name)
	}
	visiting[name] = true
	defer delete(visiting, name)
	nested, err := c.resolveNestedFilters(spec.filterRefs(), registry, filters, visiting)
	if err != nil {
		return nil, fmt.Errorf("goark-log: filter %q: %w", name, err)
	}
	filter, err := buildFilter(name, spec, nested, registry)
	if err != nil {
		return nil, err
	}
	filters[name] = filter
	return filter, nil
}

func (c *fileConfig) resolveNestedFilters(refs []string, registry *PluginRegistry, filters map[string]Filter, visiting map[string]bool) ([]Filter, error) {
	if len(refs) == 0 {
		return nil, nil
	}
	nested := make([]Filter, 0, len(refs))
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			return nil, fmt.Errorf("filter ref is empty")
		}
		filter, err := c.buildFilter(ref, registry, filters, visiting)
		if err != nil {
			return nil, err
		}
		nested = append(nested, filter)
	}
	return nested, nil
}

func buildFilter(name string, spec filterConfig, nested []Filter, registry *PluginRegistry) (Filter, error) {
	if normalizeKind(spec.Type) == "" {
		return nil, fmt.Errorf("goark-log: filter %q type is empty", name)
	}
	factory, ok := registry.filterFactory(spec.Type)
	if !ok {
		return nil, fmt.Errorf("goark-log: unsupported filter %q type %q", name, spec.Type)
	}
	config := spec.filterBuildConfig(name)
	config.Filters = nested
	filter, err := factory(config)
	if err != nil {
		return nil, fmt.Errorf("goark-log: filter %q: %w", name, err)
	}
	return filter, nil
}

func parseFilterDecisionOrDefault(value string, fallback FilterDecision) (FilterDecision, error) {
	if strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	return ParseFilterDecision(value)
}

func parseRegexFilterField(value string) (RegexFilterField, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "message", "msg":
		return RegexFieldMessage, nil
	case "logger", "name":
		return RegexFieldLogger, nil
	case "attr", "attribute":
		return RegexFieldAttr, nil
	default:
		return "", fmt.Errorf("unsupported regex filter field %q", value)
	}
}

func (c filterConfig) minLevel() string {
	return firstNonBlank(c.MinLevel, c.MinLevelKebab)
}

func (c filterConfig) maxLevel() string {
	return firstNonBlank(c.MaxLevel, c.MaxLevelKebab)
}

func (c filterConfig) maxBurst() int {
	if c.MaxBurst != 0 {
		return c.MaxBurst
	}
	return c.MaxBurstKebab
}

func (c filterConfig) defaultThreshold() string {
	return firstNonBlank(c.DefaultThreshold, c.DefaultKebab)
}

func (c filterConfig) values() map[string]string {
	values := copyStringMap(c.Values)
	if isDynamicThresholdFilterKind(c.Type) {
		return values
	}
	return mergeStringMaps(values, c.keyValuePairs())
}

func (c filterConfig) thresholds() map[string]string {
	thresholds := copyStringMap(c.Thresholds)
	if !isDynamicThresholdFilterKind(c.Type) {
		return thresholds
	}
	return mergeStringMaps(thresholds, c.keyValuePairs())
}

func (c filterConfig) keyValuePairs() map[string]string {
	pairs := make(map[string]string)
	for _, groups := range [][]keyValuePairConfig{c.KeyValuePair, c.KeyValuePairs, c.KeyValuePairsKebab} {
		for _, pair := range groups {
			key := strings.TrimSpace(pair.Key)
			if key == "" {
				continue
			}
			pairs[key] = pair.Value
		}
	}
	if len(pairs) == 0 {
		return nil
	}
	return pairs
}

func (c filterConfig) filterBuildConfig(name string) FilterBuildConfig {
	return FilterBuildConfig{
		Name:             name,
		Type:             c.Type,
		Level:            c.Level,
		MinLevel:         c.minLevel(),
		MaxLevel:         c.maxLevel(),
		Marker:           c.Marker,
		Text:             c.Text,
		Operator:         c.Operator,
		Start:            c.Start,
		End:              c.End,
		Timezone:         c.Timezone,
		Rate:             c.Rate,
		MaxBurst:         c.maxBurst(),
		Field:            c.Field,
		Key:              c.Key,
		Value:            c.Value,
		Values:           c.values(),
		Thresholds:       c.thresholds(),
		DefaultThreshold: c.defaultThreshold(),
		Pattern:          c.Pattern,
		OnMatch:          firstNonBlank(c.OnMatch, c.OnMatchKebab),
		OnMismatch:       firstNonBlank(c.OnMismatch, c.OnMismatchKebab),
	}
}

func copyStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func mergeStringMaps(base map[string]string, overlay map[string]string) map[string]string {
	if len(overlay) == 0 {
		return base
	}
	if base == nil {
		base = make(map[string]string, len(overlay))
	}
	for key, value := range overlay {
		base[key] = value
	}
	return base
}

func isDynamicThresholdFilterKind(value string) bool {
	switch normalizeKind(value) {
	case "dynamicthreshold", "dynamicthresholdfilter":
		return true
	default:
		return false
	}
}

func wrapAppenderFilters(name string, appender Appender, refs []string, filters map[string]Filter) (Appender, error) {
	resolved, err := resolveFilters(filters, refs)
	if err != nil {
		return nil, fmt.Errorf("goark-log: appender %q: %w", name, err)
	}
	if len(resolved) == 0 {
		return appender, nil
	}
	return NewFilteredAppender(appender, resolved...)
}

func resolveFilters(filters map[string]Filter, refs []string) ([]Filter, error) {
	if len(refs) == 0 {
		return nil, nil
	}
	resolved := make([]Filter, 0, len(refs))
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			return nil, fmt.Errorf("filter ref is empty")
		}
		filter, ok := filters[ref]
		if !ok {
			return nil, fmt.Errorf("filter %q is not configured", ref)
		}
		resolved = append(resolved, filter)
	}
	return resolved, nil
}

func firstAppenderRefs(groups ...appenderRefs) appenderRefs {
	for _, refs := range groups {
		if len(refs) > 0 {
			out := make(appenderRefs, len(refs))
			copy(out, refs)
			return out
		}
	}
	return nil
}

func firstStringRefs(groups ...[]string) []string {
	for _, refs := range groups {
		if len(refs) > 0 {
			out := make([]string, len(refs))
			for index, ref := range refs {
				out[index] = strings.TrimSpace(ref)
			}
			return out
		}
	}
	return nil
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstNonZero(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func parseOptionalDuration(value string) time.Duration {
	text := strings.TrimSpace(value)
	if text == "" {
		return 0
	}
	duration, err := time.ParseDuration(text)
	if err != nil {
		return -1
	}
	return duration
}

func sortedAppenderNames(appenders map[string]appenderConfig) []string {
	names := make([]string, 0, len(appenders))
	for name := range appenders {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sortedLoggerNames(loggers map[string]loggerConfig) []string {
	names := make([]string, 0, len(loggers))
	for name := range loggers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sortedFilterNames(filters map[string]filterConfig) []string {
	names := make([]string, 0, len(filters))
	for name := range filters {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func normalizeKind(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "")
	value = strings.ReplaceAll(value, "_", "")
	return value
}
