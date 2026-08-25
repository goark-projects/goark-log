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
	Appenders       map[string]appenderConfig `yaml:"appenders"`
	Filters         map[string]filterConfig   `yaml:"filters"`
	FilterRefs      []string                  `yaml:"filterRefs"`
	FilterRefsKebab []string                  `yaml:"filter-refs"`
	Root            loggerConfig              `yaml:"root"`
	Loggers         map[string]loggerConfig   `yaml:"loggers"`
	Goark           struct {
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
	Filters               []string      `yaml:"filters"`
	FilterRefs            []string      `yaml:"filterRefs"`
	FilterRefsKebab       []string      `yaml:"filter-refs"`
}

type layoutConfig struct {
	Type    string `yaml:"type"`
	Pattern string `yaml:"pattern"`
}

type rollingConfig struct {
	MaxSize         string `yaml:"maxSize"`
	MaxSizeKebab    string `yaml:"max-size"`
	Interval        string `yaml:"interval"`
	OnStartup       bool   `yaml:"onStartup"`
	OnStartupKebab  bool   `yaml:"on-startup"`
	MaxBackups      *int   `yaml:"maxBackups"`
	MaxBackupsKebab *int   `yaml:"max-backups"`
	Gzip            bool   `yaml:"gzip"`
	Compress        bool   `yaml:"compress"`
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

func loadConfigFile(ctx context.Context, path string) (*fileConfig, error) {
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
	config, err := decodeConfig(file)
	if err != nil {
		return nil, fmt.Errorf("goark-log: parse config file %q: %w", path, err)
	}
	return config, nil
}

func decodeConfig(reader io.Reader) (*fileConfig, error) {
	var config fileConfig
	decoder := yaml.NewDecoder(reader)
	decoder.KnownFields(true)
	if err := decoder.Decode(&config); err != nil {
		if errors.Is(err, io.EOF) {
			return &fileConfig{}, nil
		}
		return nil, err
	}
	return config.effective()
}

func (c *fileConfig) options() (Options, error) {
	if c == nil || c.empty() {
		return DefaultOptions(), nil
	}
	filters, err := c.buildFilters()
	if err != nil {
		return Options{}, err
	}
	appenders, err := c.buildAppenders(filters)
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

func (c *fileConfig) effective() (*fileConfig, error) {
	topLevelUsed := !c.withoutGoark().empty()
	if c.Goark.Log == nil {
		return c, nil
	}
	if topLevelUsed {
		return nil, fmt.Errorf("goark-log: config must use either top-level fields or goark.log, not both")
	}
	return c.Goark.Log, nil
}

func (c *fileConfig) withoutGoark() *fileConfig {
	if c == nil {
		return nil
	}
	return &fileConfig{
		Appenders:       c.Appenders,
		Filters:         c.Filters,
		FilterRefs:      c.FilterRefs,
		FilterRefsKebab: c.FilterRefsKebab,
		Root:            c.Root,
		Loggers:         c.Loggers,
	}
}

func (c *fileConfig) empty() bool {
	if c == nil {
		return true
	}
	return len(c.Appenders) == 0 &&
		len(c.Filters) == 0 &&
		len(c.FilterRefs) == 0 &&
		len(c.FilterRefsKebab) == 0 &&
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

func (c *fileConfig) buildAppenders(filters map[string]Filter) ([]Appender, error) {
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
		appender, err := buildConcreteAppender(name, spec, filters)
		if err != nil {
			_ = closeAppenderList(appenders)
			return nil, err
		}
		built[name] = appender
		appenders = append(appenders, appender)
	}
	for _, name := range asyncNames {
		appender, err := buildAsyncAppender(name, c.Appenders[name], built, filters)
		if err != nil {
			_ = closeAppenderList(appenders)
			return nil, err
		}
		built[name] = appender
		appenders = append(appenders, appender)
	}
	return appenders, nil
}

func buildConcreteAppender(name string, spec appenderConfig, filters map[string]Filter) (Appender, error) {
	layout, err := buildLayout(spec.Layout)
	if err != nil {
		return nil, fmt.Errorf("goark-log: appender %q: %w", name, err)
	}
	var appender Appender
	switch normalizeKind(spec.Type) {
	case "console":
		appender, err = buildConsoleAppender(name, spec, layout)
	case "file":
		appender, err = buildFileAppender(name, spec, layout)
	case "rolling", "rollingfile":
		appender, err = buildRollingAppender(name, spec, layout)
	case "":
		return nil, fmt.Errorf("goark-log: appender %q type is empty", name)
	default:
		return nil, fmt.Errorf("goark-log: unsupported appender %q type %q", name, spec.Type)
	}
	if err != nil {
		return nil, err
	}
	return wrapAppenderFilters(name, appender, spec.filterRefs(), filters)
}

func buildConsoleAppender(name string, spec appenderConfig, layout Layout) (Appender, error) {
	target := strings.ToLower(strings.TrimSpace(spec.Target))
	switch target {
	case "", "stderr":
		return NewConsoleAppender(WithConsoleName(name), WithConsoleLayout(layout), WithConsoleWriter(os.Stderr)), nil
	case "stdout":
		return NewConsoleAppender(WithConsoleName(name), WithConsoleLayout(layout), WithConsoleWriter(os.Stdout)), nil
	default:
		return nil, fmt.Errorf("goark-log: appender %q console target %q is invalid", name, spec.Target)
	}
}

func buildFileAppender(name string, spec appenderConfig, layout Layout) (Appender, error) {
	path := spec.fileName()
	if path == "" {
		return nil, fmt.Errorf("goark-log: appender %q fileName is empty", name)
	}
	return NewFileAppender(path, WithFileName(name), WithFileLayout(layout))
}

func buildRollingAppender(name string, spec appenderConfig, layout Layout) (Appender, error) {
	path := spec.fileName()
	if path == "" {
		return nil, fmt.Errorf("goark-log: appender %q fileName is empty", name)
	}
	options := []RollingFileOption{
		WithRollingFileName(name),
		WithRollingFileLayout(layout),
	}
	if value := spec.Rolling.maxSize(); value != "" {
		size, err := ParseByteSize(value)
		if err != nil {
			return nil, fmt.Errorf("goark-log: appender %q: %w", name, err)
		}
		options = append(options, WithRollingMaxSize(size))
	}
	if value := spec.Rolling.Interval; strings.TrimSpace(value) != "" {
		interval, err := ParseRollingInterval(value)
		if err != nil {
			return nil, fmt.Errorf("goark-log: appender %q: %w", name, err)
		}
		options = append(options, WithRollingInterval(interval))
	}
	if spec.Rolling.onStartup() {
		options = append(options, WithRolloverOnStartup(true))
	}
	if maxBackups, ok := spec.Rolling.maxBackups(); ok {
		options = append(options, WithRollingMaxBackups(maxBackups))
	}
	if spec.Rolling.gzipEnabled() {
		options = append(options, WithRollingGzip(true))
	}
	return NewRollingFileAppender(path, options...)
}

func buildAsyncAppender(name string, spec appenderConfig, built map[string]Appender, filters map[string]Filter) (Appender, error) {
	refs := spec.refs()
	if len(refs) == 0 {
		return nil, fmt.Errorf("goark-log: async appender %q requires appenderRefs", name)
	}
	delegates := make([]Appender, 0, len(refs))
	for _, ref := range refs {
		appender, ok := built[ref]
		if !ok {
			return nil, fmt.Errorf("goark-log: async appender %q references unknown or async appender %q", name, ref)
		}
		delegates = append(delegates, appender)
	}
	strategy, err := ParseAsyncOverflowStrategy(spec.overflowStrategy())
	if err != nil {
		return nil, fmt.Errorf("goark-log: async appender %q: %w", name, err)
	}
	options := []AsyncOption{
		WithAsyncName(name),
		WithAsyncOverflowStrategy(strategy),
	}
	if queueSize := spec.queueSize(); queueSize != 0 {
		options = append(options, WithAsyncQueueSize(queueSize))
	}
	appender, err := NewAsyncAppender(delegates, options...)
	if err != nil {
		return nil, err
	}
	return wrapAppenderFilters(name, appender, spec.filterRefs(), filters)
}

func buildLayout(config layoutConfig) (Layout, error) {
	switch normalizeKind(config.Type) {
	case "", "pattern":
		return NewPatternLayout(config.Pattern)
	case "text":
		return TextLayout{}, nil
	case "json":
		return JSONLayout{}, nil
	default:
		return nil, fmt.Errorf("unsupported layout type %q", config.Type)
	}
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

func (c rollingConfig) maxSize() string {
	if strings.TrimSpace(c.MaxSize) != "" {
		return strings.TrimSpace(c.MaxSize)
	}
	return strings.TrimSpace(c.MaxSizeKebab)
}

func (c rollingConfig) onStartup() bool {
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

func (c rollingConfig) gzipEnabled() bool {
	return c.Gzip || c.Compress
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

func (c *fileConfig) buildFilters() (map[string]Filter, error) {
	if len(c.Filters) == 0 {
		return nil, nil
	}
	names := sortedFilterNames(c.Filters)
	filters := make(map[string]Filter, len(c.Filters))
	for _, name := range names {
		filter, err := buildFilter(name, c.Filters[name])
		if err != nil {
			return nil, err
		}
		filters[name] = filter
	}
	return filters, nil
}

func buildFilter(name string, spec filterConfig) (Filter, error) {
	switch normalizeKind(spec.Type) {
	case "threshold":
		level, err := ParseLevel(spec.Level)
		if err != nil {
			return nil, fmt.Errorf("goark-log: filter %q: %w", name, err)
		}
		options, err := spec.filterOptions()
		if err != nil {
			return nil, fmt.Errorf("goark-log: filter %q: %w", name, err)
		}
		return NewThresholdFilter(level, options...), nil
	case "level":
		level, err := ParseLevel(spec.Level)
		if err != nil {
			return nil, fmt.Errorf("goark-log: filter %q: %w", name, err)
		}
		options, err := spec.filterOptions()
		if err != nil {
			return nil, fmt.Errorf("goark-log: filter %q: %w", name, err)
		}
		return NewLevelFilter(level, options...), nil
	case "levelrange":
		if spec.minLevel() == "" || spec.maxLevel() == "" {
			return nil, fmt.Errorf("goark-log: filter %q level range requires minLevel and maxLevel", name)
		}
		min, err := ParseLevel(spec.minLevel())
		if err != nil {
			return nil, fmt.Errorf("goark-log: filter %q: %w", name, err)
		}
		max, err := ParseLevel(spec.maxLevel())
		if err != nil {
			return nil, fmt.Errorf("goark-log: filter %q: %w", name, err)
		}
		options, err := spec.filterOptions()
		if err != nil {
			return nil, fmt.Errorf("goark-log: filter %q: %w", name, err)
		}
		return NewLevelRangeFilter(min, max, options...)
	case "regex":
		if strings.TrimSpace(spec.Pattern) == "" {
			return nil, fmt.Errorf("goark-log: filter %q regex pattern is empty", name)
		}
		options, err := spec.regexOutcomeOptions()
		if err != nil {
			return nil, fmt.Errorf("goark-log: filter %q: %w", name, err)
		}
		if strings.TrimSpace(spec.Field) != "" {
			field, err := parseRegexFilterField(spec.Field)
			if err != nil {
				return nil, fmt.Errorf("goark-log: filter %q: %w", name, err)
			}
			options = append(options, WithRegexField(field))
		}
		if strings.TrimSpace(spec.Key) != "" {
			options = append(options, WithRegexAttrKey(spec.Key))
		}
		return NewRegexFilter(spec.Pattern, options...)
	case "attr", "attribute":
		options, err := spec.filterOptions()
		if err != nil {
			return nil, fmt.Errorf("goark-log: filter %q: %w", name, err)
		}
		return NewAttrFilter(spec.Key, spec.Value, options...)
	case "":
		return nil, fmt.Errorf("goark-log: filter %q type is empty", name)
	default:
		return nil, fmt.Errorf("goark-log: unsupported filter %q type %q", name, spec.Type)
	}
}

func (c filterConfig) filterOptions() ([]FilterOption, error) {
	onMatch, err := c.onMatch(FilterNeutral)
	if err != nil {
		return nil, err
	}
	onMismatch, err := c.onMismatch(FilterDeny)
	if err != nil {
		return nil, err
	}
	return []FilterOption{
		WithFilterOnMatch(onMatch),
		WithFilterOnMismatch(onMismatch),
	}, nil
}

func (c filterConfig) regexOutcomeOptions() ([]RegexFilterOption, error) {
	onMatch, err := c.onMatch(FilterNeutral)
	if err != nil {
		return nil, err
	}
	onMismatch, err := c.onMismatch(FilterDeny)
	if err != nil {
		return nil, err
	}
	return []RegexFilterOption{
		WithRegexOnMatch(onMatch),
		WithRegexOnMismatch(onMismatch),
	}, nil
}

func (c filterConfig) onMatch(fallback FilterDecision) (FilterDecision, error) {
	return parseFilterDecisionOrDefault(firstNonBlank(c.OnMatch, c.OnMatchKebab), fallback)
}

func (c filterConfig) onMismatch(fallback FilterDecision) (FilterDecision, error) {
	return parseFilterDecisionOrDefault(firstNonBlank(c.OnMismatch, c.OnMismatchKebab), fallback)
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
			out := make([]string, 0, len(refs))
			for _, ref := range refs {
				if strings.TrimSpace(ref) != "" {
					out = append(out, strings.TrimSpace(ref))
				}
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
