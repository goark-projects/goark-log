package main

import (
	"log/slog"
	"os"
	"path/filepath"

	goarklog "goark.dev/goark-log"
)

func main() {
	path := filepath.Join(os.TempDir(), "goark-log-example", "file.log")
	appender, err := goarklog.NewFileAppender(path, goarklog.WithFileLayout(goarklog.TextLayout{}))
	if err != nil {
		panic(err)
	}
	handler, err := goarklog.NewHandler(goarklog.Options{
		Appenders: []goarklog.Appender{appender},
		Root: goarklog.RootLogger{
			Level:        slog.LevelInfo,
			AppenderRefs: []string{"file"},
		},
	})
	if err != nil {
		panic(err)
	}
	defer handler.Close()

	goarklog.NewLogger(handler, "goark.file").Info("file appender ready", slog.String("path", path))
}
