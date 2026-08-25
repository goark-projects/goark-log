package main

import (
	"log/slog"
	"os"
	"path/filepath"

	goarklog "goark.dev/goark-log"
)

func main() {
	path := filepath.Join(os.TempDir(), "goark-log-example", "async-rolling.log")
	rolling, err := goarklog.NewRollingFileAppender(path,
		goarklog.WithRollingFileName("rolling"),
		goarklog.WithRollingFileLayout(goarklog.TextLayout{}),
		goarklog.WithRollingMaxSize(10*1024*1024),
		goarklog.WithRollingMaxBackups(7),
		goarklog.WithRollingGzip(true),
	)
	if err != nil {
		panic(err)
	}
	async, err := goarklog.NewAsyncAppender([]goarklog.Appender{rolling},
		goarklog.WithAsyncName("async"),
		goarklog.WithAsyncQueueSize(8192),
		goarklog.WithAsyncOverflowStrategy(goarklog.AsyncOverflowBlock),
	)
	if err != nil {
		panic(err)
	}
	handler, err := goarklog.NewHandler(goarklog.Options{
		Appenders: []goarklog.Appender{rolling, async},
		Root: goarklog.RootLogger{
			Level:        slog.LevelInfo,
			AppenderRefs: []string{"async"},
		},
	})
	if err != nil {
		panic(err)
	}
	defer handler.Close()

	goarklog.NewLogger(handler, "goark.async").Info("async rolling ready", slog.String("path", path))
}
