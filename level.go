package goarklog

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
)

const (
	LevelTrace slog.Level = -8
)

// ParseLevel 解析日志级别名称。
func ParseLevel(value string) (slog.Level, error) {
	text := strings.ToLower(strings.TrimSpace(value))
	switch text {
	case "trace":
		return LevelTrace, nil
	case "debug":
		return slog.LevelDebug, nil
	case "info", "":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		level, err := strconv.Atoi(text)
		if err != nil {
			return slog.LevelInfo, fmt.Errorf("goark-log: unsupported log level %q", value)
		}
		return slog.Level(level), nil
	}
}

func levelName(level slog.Level) string {
	switch {
	case level <= LevelTrace:
		return "TRACE"
	case level < slog.LevelInfo:
		return "DEBUG"
	case level < slog.LevelWarn:
		return "INFO"
	case level < slog.LevelError:
		return "WARN"
	default:
		return "ERROR"
	}
}

func levelPointer(level slog.Level) *slog.Level {
	copied := level
	return &copied
}
