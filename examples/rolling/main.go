package main

import (
	"log/slog"
	"os"
	"path/filepath"

	goarklog "goark.dev/goark-log"
)

func main() {
	path := filepath.Join(os.TempDir(), "goark-log-example", "rolling.log")
	appender, err := goarklog.NewRollingFileAppender(path,
		goarklog.WithRollingFileLayout(goarklog.NewDefaultLayout()),
		goarklog.WithRollingMaxSize(10*1024*1024),
		goarklog.WithRollingMaxBackups(7),
		goarklog.WithRollingGzip(true),
		goarklog.WithRolloverOnStartup(true),
	)
	if err != nil {
		panic(err)
	}
	handler, err := goarklog.NewHandler(goarklog.Options{
		Appenders: []goarklog.Appender{appender},
		Root: goarklog.RootLogger{
			Level:        slog.LevelInfo,
			AppenderRefs: []string{"rollingFile"},
		},
	})
	if err != nil {
		panic(err)
	}
	defer handler.Close()

	goarklog.NewLogger(handler, "goark.rolling").Info("rolling appender ready", slog.String("path", path))
}
