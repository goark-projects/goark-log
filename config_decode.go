package goarklog

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

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
