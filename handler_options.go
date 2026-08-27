package goarklog

import (
	"log/slog"
)

// DefaultOptions 返回默认 Spring Boot 风格 stderr 配置。
func DefaultOptions() Options {
	return Options{
		Appenders: []Appender{NewConsoleAppender()},
		Root: RootLogger{
			Level:        slog.LevelInfo,
			AppenderRefs: []string{"console"},
		},
	}
}
