package goarklog

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"
)

type attrEvent struct {
	Time    time.Time
	Level   slog.Level
	Logger  string
	Message string
	Attrs   []slog.Attr
}

type attrAppender interface {
	AppendAttrs(ctx context.Context, event attrEvent) error
}

type fixedAttrEvent struct {
	Time    time.Time
	Level   slog.Level
	Logger  string
	Message string
	Attrs   [3]slog.Attr
	Count   int
}

// JSONAppender 将事件直接编码为单行 JSON，适合极低分配热路径。
type JSONAppender struct {
	name   string
	writer io.Writer
	mu     sync.Mutex
}

// JSONAppenderOption 调整 JSONAppender。
type JSONAppenderOption func(*JSONAppender)

// WithJSONAppenderName 设置 appender 名称。
func WithJSONAppenderName(name string) JSONAppenderOption {
	return func(appender *JSONAppender) {
		appender.name = name
	}
}

// WithJSONAppenderWriter 设置输出 writer，主要用于测试、基准和嵌入式直写场景。
func WithJSONAppenderWriter(writer io.Writer) JSONAppenderOption {
	return func(appender *JSONAppender) {
		appender.writer = writer
	}
}

// NewJSONAppender 创建 JSON 直写 appender。
func NewJSONAppender(options ...JSONAppenderOption) *JSONAppender {
	appender := &JSONAppender{
		name:   "json",
		writer: os.Stderr,
	}
	for _, option := range options {
		if option != nil {
			option(appender)
		}
	}
	if appender.writer == nil {
		appender.writer = os.Stderr
	}
	return appender
}

func (a *JSONAppender) Name() string {
	if a == nil || a.name == "" {
		return "json"
	}
	return a.name
}

func (a *JSONAppender) Append(ctx context.Context, event Event) error {
	if a == nil {
		return fmt.Errorf("goark-log: JSON appender is nil")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return a.write(event.Time, event.Level, event.Logger, event.Message, event.Attrs)
}

func (a *JSONAppender) AppendAttrs(ctx context.Context, event attrEvent) error {
	if a == nil {
		return fmt.Errorf("goark-log: JSON appender is nil")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return a.write(event.Time, event.Level, event.Logger, event.Message, event.Attrs)
}

func (a *JSONAppender) appendFixedAttrs(ctx context.Context, event fixedAttrEvent) error {
	if a == nil {
		return fmt.Errorf("goark-log: JSON appender is nil")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer releaseBuffer(buf)
	appendJSONFixedEvent(buf, event.Time, event.Level, event.Logger, event.Message, event.Attrs, event.Count)
	a.mu.Lock()
	defer a.mu.Unlock()
	_, err := a.writer.Write(buf.Bytes())
	return err
}

func (a *JSONAppender) write(when time.Time, level slog.Level, logger string, message string, attrs []slog.Attr) error {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer releaseBuffer(buf)
	appendJSONEvent(buf, when, level, logger, message, attrs)
	a.mu.Lock()
	defer a.mu.Unlock()
	_, err := a.writer.Write(buf.Bytes())
	return err
}

func (a *JSONAppender) Close() error {
	return nil
}
