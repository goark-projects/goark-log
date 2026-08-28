package exampleutil

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ConfigPath 返回仓库 docs/examples 下的配置文件路径。
func ConfigPath(name string) string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return filepath.Join("docs", "examples", name)
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
	return filepath.Join(root, "docs", "examples", name)
}

// PrepareLogDir 为 demo 准备隔离日志目录，并通过 GOARK_LOG_DIR 注入配置。
func PrepareLogDir(prefix string) (string, func(), error) {
	if existing := strings.TrimSpace(os.Getenv("GOARK_LOG_DIR")); existing != "" {
		return existing, func() {}, nil
	}
	dir, err := os.MkdirTemp("", "goark-log-"+prefix+"-*")
	if err != nil {
		return "", nil, err
	}
	if err := os.Setenv("GOARK_LOG_DIR", dir); err != nil {
		_ = os.RemoveAll(dir)
		return "", nil, err
	}
	return dir, func() {
		_ = os.Unsetenv("GOARK_LOG_DIR")
		_ = os.RemoveAll(dir)
	}, nil
}
