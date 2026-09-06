package configfile

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
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

func loadConfigFile(
	ctx context.Context,
	path string,
	lookups *LookupResolver,
) (*fileConfig, error) {
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
	case "toml":
		return decodeTOMLConfig(reader, lookups)
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

func decodeTOMLConfig(reader io.Reader, lookups *LookupResolver) (*fileConfig, error) {
	var raw map[string]any
	decoder := toml.NewDecoder(reader)
	if err := decoder.Decode(&raw); err != nil {
		if errors.Is(err, io.EOF) {
			return &fileConfig{}, nil
		}
		return nil, err
	}
	if len(raw) == 0 {
		return finalizeDecodedConfig(fileConfig{}, lookups)
	}
	data, err := yaml.Marshal(raw)
	if err != nil {
		return nil, err
	}
	return decodeStructuredConfig(bytes.NewReader(data), lookups)
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
		return nil, fmt.Errorf(
			"goark-log: config must use either top-level fields, configuration, or goark.log",
		)
	}
	if wrappers > 1 {
		return nil, fmt.Errorf(
			"goark-log: config must use only one wrapper: configuration or goark.log",
		)
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
