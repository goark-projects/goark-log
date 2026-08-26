package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	goarklog "goark.dev/log"
)

func main() {
	ctx := context.Background()
	dir, err := os.MkdirTemp("", "goark-log-reload-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	configPath := filepath.Join(dir, "goark-log.yml")
	logPath := filepath.Join(dir, "reload.log")
	writeConfig(configPath, logPath, "info")

	logger, handler, _, err := goarklog.NewConfigured(ctx, goarklog.WithConfigPath(configPath))
	if err != nil {
		panic(err)
	}
	defer handler.Close()

	logger.Info("visible before reload")
	logger.Debug("hidden before reload")

	writeConfig(configPath, logPath, "debug")
	reloader, err := goarklog.NewConfigReloader(handler, goarklog.WithConfigPath(configPath))
	if err != nil {
		panic(err)
	}
	if _, err := reloader.Reload(ctx); err != nil {
		panic(err)
	}
	logger.Debug("visible after reload", slog.String("path", logPath))
}

func writeConfig(configPath string, logPath string, level string) {
	content := `
appenders:
  file:
    type: file
    fileName: "` + filepath.ToSlash(logPath) + `"
    layout:
      type: text
root:
  level: ` + level + `
  appenderRefs: [file]
`
	if err := os.WriteFile(configPath, []byte(strings.TrimSpace(content)+"\n"), 0o644); err != nil {
		panic(err)
	}
}
