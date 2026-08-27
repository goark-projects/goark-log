package jsonappender

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	internallayout "goark.dev/log/internal/layout"
	"goark.dev/log/internal/logevent"
	"goark.dev/log/internal/logfile"
)

const defaultFileBufferSize = 256 * 1024

var bufferPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

// Event 是 JSON appender 接收的事件快照。
type Event = logevent.Event

// AttrEvent 是无上下文属性合并的快速 JSON 事件。
type AttrEvent struct {
	Time    time.Time
	Level   slog.Level
	Logger  string
	Message string
	Attrs   []slog.Attr
}

// AttrAppender 支持属性切片直写的 appender。
type AttrAppender interface {
	AppendAttrs(ctx context.Context, event AttrEvent) error
}

// FixedAttrEvent 是最多三个属性的快速 JSON 事件。
type FixedAttrEvent struct {
	Time    time.Time
	Level   slog.Level
	Logger  string
	Message string
	Attrs   [3]slog.Attr
	Count   int
}

// FixedAttrAppender 支持固定属性数组直写的 appender。
type FixedAttrAppender interface {
	AppendFixedAttrs(ctx context.Context, event FixedAttrEvent) error
}

type flushWriter interface {
	Flush() error
}

// Appender 将事件直接编码为单行 JSON，适合极低分配热路径。
type Appender struct {
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

// Option 调整 JSON appender。
type Option func(*Appender)

// WithName 设置 appender 名称。
func WithName(name string) Option {
	return func(appender *Appender) {
		appender.name = name
	}
}

// WithWriter 设置输出 writer，主要用于测试、基准和嵌入式直写场景。
func WithWriter(writer io.Writer) Option {
	return func(appender *Appender) {
		appender.writer = writer
		appender.externalWriter = true
	}
}

// WithBufferSize 设置文件输出缓冲大小，0 表示禁用应用层缓冲。
func WithBufferSize(size int) Option {
	return func(appender *Appender) {
		appender.bufferSize = size
	}
}

// WithFlushOnWrite 设置每次写入后立即刷新应用层缓冲。
func WithFlushOnWrite(enabled bool) Option {
	return func(appender *Appender) {
		appender.flushOnWrite = enabled
	}
}

// New 创建 JSON 直写 appender。
func New(options ...Option) *Appender {
	appender := &Appender{
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

// NewFile 创建面向文件的 JSON 直写 appender。
func NewFile(path string, options ...Option) (*Appender, error) {
	cleanPath, err := logfile.ValidatePath(path)
	if err != nil {
		return nil, err
	}
	appender := &Appender{
		name:       "json",
		bufferSize: defaultFileBufferSize,
	}
	for _, option := range options {
		if option != nil {
			option(appender)
		}
	}
	if strings.TrimSpace(appender.name) == "" {
		return nil, fmt.Errorf("goark-log: JSON appender name is empty")
	}
	if appender.externalWriter {
		return nil, fmt.Errorf("goark-log: JSON file appender %q cannot use an explicit writer", appender.Name())
	}
	if appender.bufferSize < 0 {
		return nil, fmt.Errorf("goark-log: JSON file buffer size must be >= 0")
	}
	file, err := logfile.Open(cleanPath)
	if err != nil {
		return nil, err
	}
	appender.file = file
	if appender.bufferSize > 0 {
		size := appender.bufferSize
		writer := bufio.NewWriterSize(file, size)
		appender.buffered = writer
		appender.writer = writer
	} else {
		appender.writer = file
	}
	return appender, nil
}

func (a *Appender) Name() string {
	if a == nil || a.name == "" {
		return "json"
	}
	return a.name
}

func (a *Appender) Append(ctx context.Context, event Event) error {
	if a == nil {
		return fmt.Errorf("goark-log: JSON appender is nil")
	}
	ctx = logevent.NormalizeContext(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}
	return a.write(event.Time, event.Level, event.Logger, event.Message, event.Attrs)
}

func (a *Appender) AppendAttrs(ctx context.Context, event AttrEvent) error {
	if a == nil {
		return fmt.Errorf("goark-log: JSON appender is nil")
	}
	ctx = logevent.NormalizeContext(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}
	return a.write(event.Time, event.Level, event.Logger, event.Message, event.Attrs)
}

// AppendFixedAttrs 写出固定属性数组事件，避免调用侧分配属性切片。
func (a *Appender) AppendFixedAttrs(ctx context.Context, event FixedAttrEvent) error {
	if a == nil {
		return fmt.Errorf("goark-log: JSON appender is nil")
	}
	ctx = logevent.NormalizeContext(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer releaseBuffer(buf)
	internallayout.AppendJSONFixedEvent(buf, event.Time, event.Level, event.Logger, event.Message, event.Attrs, event.Count)
	return a.writeBytes(buf.Bytes())
}

func (a *Appender) write(when time.Time, level slog.Level, logger string, message string, attrs []slog.Attr) error {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer releaseBuffer(buf)
	internallayout.AppendJSONEvent(buf, when, level, logger, message, attrs)
	return a.writeBytes(buf.Bytes())
}

func (a *Appender) writeBytes(data []byte) error {
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

func (a *Appender) Close() error {
	if a == nil {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.closeLocked()
}

func (a *Appender) closeLocked() error {
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

func (a *Appender) Flush() error {
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

func releaseBuffer(buf *bytes.Buffer) {
	if buf.Cap() > 64*1024 {
		return
	}
	buf.Reset()
	bufferPool.Put(buf)
}
