package goarklog

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

func decodePropertiesConfig(reader io.Reader, lookups *LookupResolver) (*fileConfig, error) {
	values, err := readProperties(reader)
	if err != nil {
		return nil, err
	}
	config, err := propertiesToFileConfig(values)
	if err != nil {
		return nil, err
	}
	return finalizeDecodedConfig(config, lookups)
}

func readProperties(reader io.Reader) (map[string]string, error) {
	scanner := bufio.NewScanner(reader)
	values := make(map[string]string)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}
		key, value, ok := cutProperty(line)
		if !ok {
			return nil, fmt.Errorf("goark-log: properties line %d is invalid", lineNumber)
		}
		values[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return values, nil
}

func cutProperty(line string) (string, string, bool) {
	for _, separator := range []string{"=", ":"} {
		key, value, ok := strings.Cut(line, separator)
		if ok {
			return key, value, true
		}
	}
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return "", "", false
	}
	return fields[0], strings.Join(fields[1:], " "), true
}

func propertiesToFileConfig(values map[string]string) (fileConfig, error) {
	aliases := collectPropertyAliases(values)
	config := fileConfig{
		Properties:   make(map[string]string),
		CustomLevels: make(map[string]string),
		Appenders:    make(map[string]appenderConfig),
		Filters:      make(map[string]filterConfig),
		Loggers:      make(map[string]loggerConfig),
	}
	for key, value := range values {
		if err := applyProperty(&config, aliases, key, value); err != nil {
			return fileConfig{}, err
		}
	}
	if err := applyFilterKeyValuePairs(&config, values); err != nil {
		return fileConfig{}, err
	}
	if len(config.Properties) == 0 {
		config.Properties = nil
	}
	if len(config.CustomLevels) == 0 {
		config.CustomLevels = nil
	}
	return config, nil
}
