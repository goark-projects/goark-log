package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"

	goarklog "goark.dev/log"
	"goark.dev/log/examples/internal/exampleutil"
)

func main() {
	dir, err := os.MkdirTemp("", "goark-log-reload-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	configPath := filepath.Join(dir, "goark-log.yml")
	logPath := filepath.Join(dir, "reload.log")
	writeConfig(configPath, logPath, "info")

	logger, handler, _, err := goarklog.NewConfigured(context.Background(), goarklog.WithConfigPath(configPath))
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
	if _, err := reloader.Reload(context.Background()); err != nil {
		panic(err)
	}
	logger.Debug("visible after reload", slog.String("path", logPath))
	slog.Info("reload demo completed", slog.String("template", exampleutil.ConfigPath("production-service.yml")))
}

func writeConfig(configPath string, logPath string, level string) {
	content := "configuration:\n" +
		"  appenders:\n" +
		"    json:\n" +
		"      type: json\n" +
		"      fileName: " + strconv.Quote(filepath.ToSlash(logPath)) + "\n" +
		"      flushOnWrite: true\n" +
		"  root:\n" +
		"    level: " + level + "\n" +
		"    appenderRefs: [json]\n"
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		panic(err)
	}
}
