package goarklog

import (
	"fmt"
	"net/url"
	"os"
	"runtime"
	"strings"
)

func readJSONTemplateFile(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("goark-log: JSON template file path is empty")
	}
	resolved, err := localTemplatePath(path)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return "", fmt.Errorf("goark-log: read JSON template file %q: %w", resolved, err)
	}
	return string(data), nil
}

func localTemplatePath(value string) (string, error) {
	if runtime.GOOS == "windows" && len(value) >= 2 && value[1] == ':' {
		return value, nil
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" {
		return value, nil
	}
	if parsed.Scheme != "file" {
		return "", fmt.Errorf("goark-log: JSON template URI scheme %q is not allowed in core", parsed.Scheme)
	}
	path := parsed.Path
	if parsed.Host != "" {
		path = "//" + parsed.Host + parsed.Path
	}
	if runtime.GOOS == "windows" && len(path) >= 3 && path[0] == '/' && path[2] == ':' {
		path = path[1:]
	}
	return path, nil
}
