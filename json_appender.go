package goarklog

import (
	"bytes"
	"context"
	"errors"
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
	name           string
	writer         io.Writer
	externalWriter bool
	mu             sync.Mutex
	file           *os.File
	buffered       flushWriter
	bufferSize     int
	flushOnWrite   bool
	closed         bool
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
		appender.externalWriter = true
	}
}

// NewJSONAppender 创建 JSON 直写 appender。
func NewJSONAppender(options ...JSONAppenderOption) *JSONAppender {
	appender := &JSONAppender{
		name: "json",
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
	ctx = normalizeContext(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}
	return a.write(event.Time, event.Level, event.Logger, event.Message, event.Attrs)
}

func (a *JSONAppender) AppendAttrs(ctx context.Context, event attrEvent) error {
	if a == nil {
		return fmt.Errorf("goark-log: JSON appender is nil")
	}
	ctx = normalizeContext(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}
	return a.write(event.Time, event.Level, event.Logger, event.Message, event.Attrs)
}

func (a *JSONAppender) appendFixedAttrs(ctx context.Context, event fixedAttrEvent) error {
	if a == nil {
		return fmt.Errorf("goark-log: JSON appender is nil")
	}
	ctx = normalizeContext(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer releaseBuffer(buf)
	appendJSONFixedEvent(buf, event.Time, event.Level, event.Logger, event.Message, event.Attrs, event.Count)
	return a.writeBytes(buf.Bytes())
}

func (a *JSONAppender) write(when time.Time, level slog.Level, logger string, message string, attrs []slog.Attr) error {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer releaseBuffer(buf)
	appendJSONEvent(buf, when, level, logger, message, attrs)
	return a.writeBytes(buf.Bytes())
}

func (a *JSONAppender) writeBytes(data []byte) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed {
		return fmt.Errorf("goark-log: JSON appender %q is closed", a.Name())
	}
	if a.writer == nil {
		return fmt.Errorf("goark-log: JSON appender %q writer is nil", a.Name())
	}
	_, err := a.writer.Write(data)
	if err == nil && a.flushOnWrite && a.buffered != nil {
		err = a.buffered.Flush()
	}
	return err
}

func (a *JSONAppender) Close() error {
	if a == nil {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.closeLocked()
}

func (a *JSONAppender) closeLocked() error {
	if a.closed {
		return nil
	}
	a.closed = true
	if a.file == nil {
		return nil
	}
	flushErr := flushJSONAppenderWriter(a.buffered)
	closeErr := a.file.Close()
	a.file = nil
	a.buffered = nil
	a.writer = nil
	return errors.Join(flushErr, closeErr)
}

func (a *JSONAppender) Flush() error {
	if a == nil {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed {
		return fmt.Errorf("goark-log: JSON appender %q is closed", a.Name())
	}
	return flushJSONAppenderWriter(a.buffered)
}

func flushJSONAppenderWriter(writer flushWriter) error {
	if writer == nil {
		return nil
	}
	return writer.Flush()
}
