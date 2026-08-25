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
	FilePattern      string `yaml:"filePattern"`
	FilePatternKebab string `yaml:"file-pattern"`
	MaxSize          string `yaml:"maxSize"`
	MaxSizeKebab     string `yaml:"max-size"`
	Interval         string `yaml:"interval"`
	OnStartup        bool   `yaml:"onStartup"`
	OnStartupKebab   bool   `yaml:"on-startup"`
	MaxBackups       *int   `yaml:"maxBackups"`
	MaxBackupsKebab  *int   `yaml:"max-backups"`
	MaxAge           string `yaml:"maxAge"`
	MaxAgeKebab      string `yaml:"max-age"`
	Gzip             bool   `yaml:"gzip"`
	Compress         bool   `yaml:"compress"`
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
	if err := effective.resolveLookups(lookups); err != nil {
		return nil, err
	}
	return effective, nil
}

func (c *fileConfig) options(registry *PluginRegistry) (Options, error) {
	if c == nil || c.empty() {
		return DefaultOptions(), nil
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

func (c *fileConfig) resolveLookups(lookups *LookupResolver) error {
	if c == nil {
		return nil
	}
	if lookups == nil {
		lookups = NewLookupResolver()
	}
	var err error
	c.FilterRefs, err = resolveStringListLookups(lookups, c.FilterRefs)
	if err != nil {
		return fmt.Errorf("goark-log: filterRefs: %w", err)
	}
	c.FilterRefsKebab, err = resolveStringListLookups(lookups, c.FilterRefsKebab)
	if err != nil {
		return fmt.Errorf("goark-log: filter-refs: %w", err)
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
	return wrapAppenderFilters(name, appender, spec.filterRefs(), filters)
}

func buildAsyncAppender(name string, spec appenderConfig, built map[string]Appender, filters map[string]Filter, registry *PluginRegistry) (Appender, error) {
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
	factory, ok := registry.appenderFactory(spec.Type)
	if !ok {
		return nil, fmt.Errorf("goark-log: unsupported appender %q type %q", name, spec.Type)
	}
	appender, err := factory(spec.appenderBuildConfig(name, nil, delegates))
	if err != nil {
		return nil, err
	}
	return wrapAppenderFilters(name, appender, spec.filterRefs(), filters)
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
		Rolling: RollingBuildConfig{
			FilePattern: c.Rolling.filePattern(),
			MaxSize:     c.Rolling.maxSize(),
			Interval:    c.Rolling.Interval,
			OnStartup:   c.Rolling.onStartup(),
			MaxBackups:  c.Rolling.maxBackupsPointer(),
			MaxAge:      c.Rolling.maxAge(),
			Gzip:        c.Rolling.gzipEnabled(),
		},
	}
}

func (c rollingConfig) filePattern() string {
	return firstNonBlank(c.FilePattern, c.FilePatternKebab)
}

func (c rollingConfig) maxSize() string {
	if strings.TrimSpace(c.MaxSize) != "" {
		return strings.TrimSpace(c.MaxSize)
	}
	return strings.TrimSpace(c.MaxSizeKebab)
}

func (c rollingConfig) maxAge() string {
	return firstNonBlank(c.MaxAge, c.MaxAgeKebab)
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

func (c rollingConfig) maxBackupsPointer() *int {
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
