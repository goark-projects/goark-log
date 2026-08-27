package status

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"

	"goark.dev/log/internal/level"
	"goark.dev/log/internal/timepattern"
)

const defaultBufferSize = 128

// Event 是 goark-log 内部状态事件。
type Event struct {
	Time    time.Time
	Level   slog.Level
	Message string
	Err     error
}

// Logger 记录 goark-log 内部配置、重载和写出错误。
type Logger struct {
	mu         sync.Mutex
	level      slog.Level
	writer     io.Writer
	maxRecords int
	records    []Event
	now        func() time.Time
}

// Option 调整 Logger。
type Option func(*Logger)

// WithLevel 设置状态日志级别。
func WithLevel(level slog.Level) Option {
	return func(logger *Logger) {
		logger.level = level
	}
}

// WithWriter 设置状态日志 writer。
func WithWriter(writer io.Writer) Option {
	return func(logger *Logger) {
		logger.writer = writer
	}
}

// WithBufferSize 设置保留的内存状态事件数量。
func WithBufferSize(size int) Option {
	return func(logger *Logger) {
		logger.maxRecords = size
	}
}

// New 创建状态日志器。
func New(options ...Option) *Logger {
	logger := &Logger{
		level:      slog.LevelWarn,
		writer:     os.Stderr,
		maxRecords: defaultBufferSize,
		now:        time.Now,
	}
	for _, option := range options {
		if option != nil {
			option(logger)
		}
	}
	if logger.writer == nil {
		logger.writer = io.Discard
	}
	if logger.maxRecords < 0 {
		logger.maxRecords = 0
	}
	if logger.now == nil {
		logger.now = time.Now
	}
	return logger
}

// Log 写入一条内部状态事件。
func (l *Logger) Log(ctx context.Context, level slog.Level, message string, err error) {
	if l == nil || level < l.level {
		return
	}
	if ctx != nil {
		if ctxErr := ctx.Err(); ctxErr != nil && err == nil {
			err = ctxErr
		}
	}
	event := Event{
		Time:    l.now(),
		Level:   level,
		Message: message,
		Err:     err,
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.maxRecords > 0 {
		if len(l.records) == l.maxRecords {
			copy(l.records, l.records[1:])
			l.records[len(l.records)-1] = event
		} else {
			l.records = append(l.records, event)
		}
	}
	if l.writer != nil {
		writeEvent(l.writer, event)
	}
}

// Events 返回已保留的状态事件快照。
func (l *Logger) Events() []Event {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]Event(nil), l.records...)
}

// Debug 写入 DEBUG 状态事件。
func (l *Logger) Debug(ctx context.Context, message string) {
	l.Log(ctx, slog.LevelDebug, message, nil)
}

// Info 写入 INFO 状态事件。
func (l *Logger) Info(ctx context.Context, message string) {
	l.Log(ctx, slog.LevelInfo, message, nil)
}

// Warn 写入 WARN 状态事件。
func (l *Logger) Warn(ctx context.Context, message string, err error) {
	l.Log(ctx, slog.LevelWarn, message, err)
}

// Error 写入 ERROR 状态事件。
func (l *Logger) Error(ctx context.Context, message string, err error) {
	l.Log(ctx, slog.LevelError, message, err)
}

func writeEvent(writer io.Writer, event Event) {
	if event.Err != nil {
		_, _ = fmt.Fprintf(writer, "%s %5s goark-log status: %s: %v\n",
			event.Time.Format(timepattern.DefaultLayout),
			level.NameDefault(event.Level),
			event.Message,
			event.Err,
		)
		return
	}
	_, _ = fmt.Fprintf(writer, "%s %5s goark-log status: %s\n",
		event.Time.Format(timepattern.DefaultLayout),
		level.NameDefault(event.Level),
		event.Message,
	)
}
