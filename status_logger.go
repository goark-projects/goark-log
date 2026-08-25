package goarklog

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"
)

const defaultStatusBufferSize = 128

// StatusEvent 是 goark-log 内部状态事件。
type StatusEvent struct {
	Time    time.Time
	Level   slog.Level
	Message string
	Err     error
}

// StatusLogger 记录 goark-log 内部配置、重载和写出错误。
type StatusLogger struct {
	mu         sync.Mutex
	level      slog.Level
	writer     io.Writer
	maxRecords int
	records    []StatusEvent
	now        func() time.Time
}

// StatusOption 调整 StatusLogger。
type StatusOption func(*StatusLogger)

// WithStatusLevel 设置状态日志级别。
func WithStatusLevel(level slog.Level) StatusOption {
	return func(logger *StatusLogger) {
		logger.level = level
	}
}

// WithStatusWriter 设置状态日志 writer。
func WithStatusWriter(writer io.Writer) StatusOption {
	return func(logger *StatusLogger) {
		logger.writer = writer
	}
}

// WithStatusBufferSize 设置保留的内存状态事件数量。
func WithStatusBufferSize(size int) StatusOption {
	return func(logger *StatusLogger) {
		logger.maxRecords = size
	}
}

// NewStatusLogger 创建状态日志器。
func NewStatusLogger(options ...StatusOption) *StatusLogger {
	logger := &StatusLogger{
		level:      slog.LevelWarn,
		writer:     os.Stderr,
		maxRecords: defaultStatusBufferSize,
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
func (s *StatusLogger) Log(ctx context.Context, level slog.Level, message string, err error) {
	if s == nil || level < s.level {
		return
	}
	if ctx != nil {
		if ctxErr := ctx.Err(); ctxErr != nil && err == nil {
			err = ctxErr
		}
	}
	event := StatusEvent{
		Time:    s.now(),
		Level:   level,
		Message: message,
		Err:     err,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.maxRecords > 0 {
		if len(s.records) == s.maxRecords {
			copy(s.records, s.records[1:])
			s.records[len(s.records)-1] = event
		} else {
			s.records = append(s.records, event)
		}
	}
	if s.writer != nil {
		writeStatusEvent(s.writer, event)
	}
}

// Events 返回已保留的状态事件快照。
func (s *StatusLogger) Events() []StatusEvent {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]StatusEvent(nil), s.records...)
}

// Debug 写入 DEBUG 状态事件。
func (s *StatusLogger) Debug(ctx context.Context, message string) {
	s.Log(ctx, slog.LevelDebug, message, nil)
}

// Info 写入 INFO 状态事件。
func (s *StatusLogger) Info(ctx context.Context, message string) {
	s.Log(ctx, slog.LevelInfo, message, nil)
}

// Warn 写入 WARN 状态事件。
func (s *StatusLogger) Warn(ctx context.Context, message string, err error) {
	s.Log(ctx, slog.LevelWarn, message, err)
}

// Error 写入 ERROR 状态事件。
func (s *StatusLogger) Error(ctx context.Context, message string, err error) {
	s.Log(ctx, slog.LevelError, message, err)
}

func writeStatusEvent(writer io.Writer, event StatusEvent) {
	if event.Err != nil {
		_, _ = fmt.Fprintf(writer, "%s %5s goark-log status: %s: %v\n",
			event.Time.Format(defaultTimeFormat),
			levelName(event.Level),
			event.Message,
			event.Err,
		)
		return
	}
	_, _ = fmt.Fprintf(writer, "%s %5s goark-log status: %s\n",
		event.Time.Format(defaultTimeFormat),
		levelName(event.Level),
		event.Message,
	)
}
