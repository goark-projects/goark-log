package goarklog

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type fileConfig struct {
	Configuration    *fileConfig               `yaml:"configuration"`
	Status           string                    `yaml:"status"`
	Properties       map[string]string         `yaml:"properties"`
	Appenders        map[string]appenderConfig `yaml:"appenders"`
	Filters          map[string]filterConfig   `yaml:"filters"`
	FilterRefs       []string                  `yaml:"filterRefs"`
	FilterRefsKebab  []string                  `yaml:"filter-refs"`
	AsyncLogger      asyncLoggerConfig         `yaml:"asyncLogger"`
	AsyncLoggerKebab asyncLoggerConfig         `yaml:"async-logger"`
	Async            asyncLoggerConfig         `yaml:"async"`
	Root             loggerConfig              `yaml:"root"`
	Loggers          map[string]loggerConfig   `yaml:"loggers"`
	Goark            struct {
		Log *fileConfig `yaml:"log"`
	} `yaml:"goark"`
}

type appenderConfig struct {
	Type                  string        `yaml:"type"`
	Target                string        `yaml:"target"`
	FileName              string        `yaml:"fileName"`
	FileNameKebab         string        `yaml:"file-name"`
	Path                  string        `yaml:"path"`
	Layout                layoutConfig  `yaml:"layout"`
	Rolling               rollingConfig `yaml:"rolling"`
	AppenderRefs          []string      `yaml:"appenderRefs"`
	AppenderRefsKebab     []string      `yaml:"appender-refs"`
	Refs                  []string      `yaml:"refs"`
	QueueSize             int           `yaml:"queueSize"`
	QueueSizeKebab        int           `yaml:"queue-size"`
	OverflowStrategy      string        `yaml:"overflowStrategy"`
	OverflowStrategyKebab string        `yaml:"overflow-strategy"`
	BufferSize            string        `yaml:"bufferSize"`
	BufferSizeKebab       string        `yaml:"buffer-size"`
	FlushOnWrite          bool          `yaml:"flushOnWrite"`
	FlushOnWriteKebab     bool          `yaml:"flush-on-write"`
	Filters               []string      `yaml:"filters"`
	FilterRefs            []string      `yaml:"filterRefs"`
	FilterRefsKebab       []string      `yaml:"filter-refs"`
}

type layoutConfig struct {
	Type    string `yaml:"type"`
	Pattern string `yaml:"pattern"`
}

type rollingConfig struct {
	FilePattern          string                `yaml:"filePattern"`
	FilePatternKebab     string                `yaml:"file-pattern"`
	MaxSize              string                `yaml:"maxSize"`
	MaxSizeKebab         string                `yaml:"max-size"`
	Interval             string                `yaml:"interval"`
	OnStartup            bool                  `yaml:"onStartup"`
	OnStartupKebab       bool                  `yaml:"on-startup"`
	MaxBackups           *int                  `yaml:"maxBackups"`
	MaxBackupsKebab      *int                  `yaml:"max-backups"`
	MaxAge               string                `yaml:"maxAge"`
	MaxAgeKebab          string                `yaml:"max-age"`
	Gzip                 bool                  `yaml:"gzip"`
	Compress             bool                  `yaml:"compress"`
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

type rollingStartupPolicyConfig struct {
	Enabled *bool `yaml:"enabled"`
}

type rollingStrategyConfig struct {
	Max                  *int                        `yaml:"max"`
	MaxBackups           *int                        `yaml:"maxBackups"`
	MaxBackupsKebab      *int                        `yaml:"max-backups"`
	MaxAge               string                      `yaml:"maxAge"`
	MaxAgeKebab          string                      `yaml:"max-age"`
	FileIndex            string                      `yaml:"fileIndex"`
	FileIndexKebab       string                      `yaml:"file-index"`
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
	BasePath            string                          `yaml:"basePath"`
	BasePathKebab       string                          `yaml:"base-path"`
	MaxDepth            *int                            `yaml:"maxDepth"`
	MaxDepthKebab       *int                            `yaml:"max-depth"`
	Glob                string                          `yaml:"glob"`
	Age                 string                          `yaml:"age"`
	Async               bool                            `yaml:"async"`
	IfFileName          rollingDeleteFileNameConfig     `yaml:"ifFileName"`
	IfFileNameKebab     rollingDeleteFileNameConfig     `yaml:"if-file-name"`
	IfLastModified      rollingDeleteLastModifiedConfig `yaml:"ifLastModified"`
	IfLastModifiedKebab rollingDeleteLastModifiedConfig `yaml:"if-last-modified"`
}

type rollingDeleteFileNameConfig struct {
	Glob string `yaml:"glob"`
}

type rollingDeleteLastModifiedConfig struct {
	Age string `yaml:"age"`
}

type asyncLoggerConfig struct {
	Enabled               *bool  `yaml:"enabled"`
	QueueSize             int    `yaml:"queueSize"`
	QueueSizeKebab        int    `yaml:"queue-size"`
	BatchSize             int    `yaml:"batchSize"`
	BatchSizeKebab        int    `yaml:"batch-size"`
	OverflowStrategy      string `yaml:"overflowStrategy"`
	OverflowStrategyKebab string `yaml:"overflow-strategy"`
}

type loggerConfig struct {
	Level             string   `yaml:"level"`
	AppenderRefs      []string `yaml:"appenderRefs"`
	AppenderRefsKebab []string `yaml:"appender-refs"`
	Refs              []string `yaml:"refs"`
	Filters           []string `yaml:"filters"`
	FilterRefs        []string `yaml:"filterRefs"`
	FilterRefsKebab   []string `yaml:"filter-refs"`
	Additivity        *bool    `yaml:"additivity"`
}

type filterConfig struct {
	Type            string `yaml:"type"`
	Level           string `yaml:"level"`
	MinLevel        string `yaml:"minLevel"`
	MinLevelKebab   string `yaml:"min-level"`
	MaxLevel        string `yaml:"maxLevel"`
	MaxLevelKebab   string `yaml:"max-level"`
	Field           string `yaml:"field"`
	Key             string `yaml:"key"`
	Value           string `yaml:"value"`
	Pattern         string `yaml:"pattern"`
	OnMatch         string `yaml:"onMatch"`
	OnMatchKebab    string `yaml:"on-match"`
	OnMismatch      string `yaml:"onMismatch"`
	OnMismatchKebab string `yaml:"on-mismatch"`
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
	if format != "yaml" {
		return nil, fmt.Errorf("goark-log: unsupported config format %q for %q", format, path)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("goark-log: open config file %q: %w", path, err)
	}
	defer file.Close()
	config, err := decodeConfig(file, lookups)
	if err != nil {
		return nil, fmt.Errorf("goark-log: parse config file %q: %w", path, err)
	}
	return config, nil
}

func decodeConfig(reader io.Reader, lookups *LookupResolver) (*fileConfig, error) {
	var config fileConfig
	decoder := yaml.NewDecoder(reader)
	decoder.KnownFields(true)
	if err := decoder.Decode(&config); err != nil {
		if errors.Is(err, io.EOF) {
			return &fileConfig{}, nil
		}
		return nil, err
	}
	effective, err := config.effective()
	if err != nil {
		return nil, err
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
	if err := c.resolveProperties(lookups); err != nil {
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
	if c.AppenderRefs, err = resolveStringListLookups(lookups, c.AppenderRefs); err != nil {
		return fmt.Errorf("appenderRefs: %w", err)
	}
	if c.AppenderRefsKebab, err = resolveStringListLookups(lookups, c.AppenderRefsKebab); err != nil {
		return fmt.Errorf("appender-refs: %w", err)
	}
	if c.Refs, err = resolveStringListLookups(lookups, c.Refs); err != nil {
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

func (c *loggerConfig) resolveLookups(lookups *LookupResolver) error {
	var err error
	if c.Level, err = resolveStringLookup(lookups, c.Level); err != nil {
		return fmt.Errorf("level: %w", err)
	}
	if c.AppenderRefs, err = resolveStringListLookups(lookups, c.AppenderRefs); err != nil {
		return fmt.Errorf("appenderRefs: %w", err)
	}
	if c.AppenderRefsKebab, err = resolveStringListLookups(lookups, c.AppenderRefsKebab); err != nil {
		return fmt.Errorf("appender-refs: %w", err)
	}
	if c.Refs, err = resolveStringListLookups(lookups, c.Refs); err != nil {
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
	if c.Field, err = resolveStringLookup(lookups, c.Field); err != nil {
		return fmt.Errorf("field: %w", err)
	}
	if c.Key, err = resolveStringLookup(lookups, c.Key); err != nil {
		return fmt.Errorf("key: %w", err)
	}
	if c.Value, err = resolveStringLookup(lookups, c.Value); err != nil {
		return fmt.Errorf("value: %w", err)
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

func (c *fileConfig) withoutWrappers() *fileConfig {
	if c == nil {
		return nil
	}
	return &fileConfig{
		Status:           c.Status,
		Properties:       c.Properties,
		Appenders:        c.Appenders,
		Filters:          c.Filters,
		FilterRefs:       c.FilterRefs,
		FilterRefsKebab:  c.FilterRefsKebab,
		AsyncLogger:      c.AsyncLogger,
		AsyncLoggerKebab: c.AsyncLoggerKebab,
		Async:            c.Async,
		Root:             c.Root,
		Loggers:          c.Loggers,
	}
}

func (c *fileConfig) empty() bool {
	if c == nil {
		return true
	}
	return len(c.Appenders) == 0 &&
		strings.TrimSpace(c.Status) == "" &&
		len(c.Properties) == 0 &&
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
		strings.TrimSpace(c.OverflowStrategyKebab) == ""
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
	if len(refs) == 0 {
		return nil, fmt.Errorf("goark-log: async appender %q requires appenderRefs", name)
	}
	delegates := make([]Appender, 0, len(refs))
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
		Type:    config.Type,
		Pattern: config.Pattern,
	})
}

func configFormat(path string) (string, error) {
	switch strings.ToLower(strings.TrimPrefix(filepathExt(path), ".")) {
	case "yml", "yaml":
		return "yaml", nil
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
	return firstRefs(c.AppenderRefs, c.AppenderRefsKebab, c.Refs)
}

func (c appenderConfig) filterRefs() []string {
	return firstRefs(c.Filters, c.FilterRefs, c.FilterRefsKebab)
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
		FileName:         c.fileName(),
		Layout:           layout,
		AppenderRefs:     c.refs(),
		Delegates:        append([]Appender(nil), delegates...),
		QueueSize:        c.queueSize(),
		OverflowStrategy: c.overflowStrategy(),
		BufferSize:       c.bufferSize(),
		FlushOnWrite:     c.flushOnWrite(),
		Rolling: RollingBuildConfig{
			FilePattern:     c.Rolling.filePattern(),
			MaxSize:         c.Rolling.maxSize(),
			Interval:        c.Rolling.interval(),
			TimeModulate:    c.Rolling.timeModulate(),
			OnStartup:       c.Rolling.onStartup(),
			MaxBackups:      c.Rolling.maxBackupsPointer(),
			MaxAge:          c.Rolling.maxAge(),
			FileIndex:       c.Rolling.fileIndex(),
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

func (c rollingConfig) timeModulate() *bool {
	return c.Policies.timePolicy().Modulate
}

func (c rollingConfig) maxAge() string {
	return firstNonBlank(c.Strategy.MaxAge, c.Strategy.MaxAgeKebab, c.MaxAge, c.MaxAgeKebab)
}

func (c rollingConfig) fileIndex() string {
	return firstNonBlank(c.Strategy.FileIndex, c.Strategy.FileIndexKebab)
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

func (c rollingDeleteActionConfig) empty() bool {
	return firstNonBlank(c.BasePath, c.BasePathKebab, c.Glob, c.Age,
		c.IfFileName.Glob, c.IfFileNameKebab.Glob,
		c.IfLastModified.Age, c.IfLastModifiedKebab.Age) == "" &&
		c.MaxDepth == nil && c.MaxDepthKebab == nil
}

func (c rollingDeleteActionConfig) build(defaultBase string) RollingDeleteBuildConfig {
	config := RollingDeleteBuildConfig{
		BasePath: firstNonBlank(c.BasePath, c.BasePathKebab, defaultBase),
		Glob:     firstNonBlank(c.Glob, c.IfFileName.Glob, c.IfFileNameKebab.Glob),
		MaxAge:   firstNonBlank(c.Age, c.IfLastModified.Age, c.IfLastModifiedKebab.Age),
	}
	if c.MaxDepth != nil {
		config.MaxDepth = *c.MaxDepth
	} else if c.MaxDepthKebab != nil {
		config.MaxDepth = *c.MaxDepthKebab
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
	return firstRefs(c.AppenderRefs, c.AppenderRefsKebab, c.Refs)
}

func (c loggerConfig) filterRefs() []string {
	return firstRefs(c.Filters, c.FilterRefs, c.FilterRefsKebab)
}

func (c *fileConfig) filterRefs() []string {
	if c == nil {
		return nil
	}
	return firstRefs(c.FilterRefs, c.FilterRefsKebab)
}

func (c *fileConfig) buildFilters(registry *PluginRegistry) (map[string]Filter, error) {
	if len(c.Filters) == 0 {
		return nil, nil
	}
	names := sortedFilterNames(c.Filters)
	filters := make(map[string]Filter, len(c.Filters))
	for _, name := range names {
		filter, err := buildFilter(name, c.Filters[name], registry)
		if err != nil {
			return nil, err
		}
		filters[name] = filter
	}
	return filters, nil
}

func buildFilter(name string, spec filterConfig, registry *PluginRegistry) (Filter, error) {
	if normalizeKind(spec.Type) == "" {
		return nil, fmt.Errorf("goark-log: filter %q type is empty", name)
	}
	factory, ok := registry.filterFactory(spec.Type)
	if !ok {
		return nil, fmt.Errorf("goark-log: unsupported filter %q type %q", name, spec.Type)
	}
	filter, err := factory(spec.filterBuildConfig(name))
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

func (c filterConfig) filterBuildConfig(name string) FilterBuildConfig {
	return FilterBuildConfig{
		Name:       name,
		Type:       c.Type,
		Level:      c.Level,
		MinLevel:   c.minLevel(),
		MaxLevel:   c.maxLevel(),
		Field:      c.Field,
		Key:        c.Key,
		Value:      c.Value,
		Pattern:    c.Pattern,
		OnMatch:    firstNonBlank(c.OnMatch, c.OnMatchKebab),
		OnMismatch: firstNonBlank(c.OnMismatch, c.OnMismatchKebab),
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

func firstRefs(groups ...[]string) []string {
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
