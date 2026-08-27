package goarklog

import (
	"encoding/xml"
	"io"
	"strings"
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
