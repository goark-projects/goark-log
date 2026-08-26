package goarklog

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	// DefaultFileBufferSize 是文件 appender 默认缓冲大小。
	DefaultFileBufferSize = 256 * 1024
)

// FileAppender 把日志追加写入普通文件。
type FileAppender struct {
	name         string
	path         string
	layout       Layout
	bufferSize   int
	flushOnWrite bool
	mu           sync.Mutex
	file         *os.File
	writer       *bufio.Writer
	closed       bool
}

// FileOption 调整 FileAppender。
type FileOption func(*FileAppender)

// WithFileName 设置 appender 名称。
func WithFileName(name string) FileOption {
	return func(appender *FileAppender) {
		appender.name = name
	}
}

// WithFileLayout 设置日志布局。
func WithFileLayout(layout Layout) FileOption {
	return func(appender *FileAppender) {
		appender.layout = layout
	}
}

// WithFileBufferSize 设置文件写缓冲大小，0 表示禁用缓冲。
func WithFileBufferSize(size int) FileOption {
	return func(appender *FileAppender) {
		appender.bufferSize = size
	}
}

// WithFileFlushOnWrite 设置每次写入后立即 flush。
func WithFileFlushOnWrite(enabled bool) FileOption {
	return func(appender *FileAppender) {
		appender.flushOnWrite = enabled
	}
}

// NewFileAppender 创建普通文件 appender。
func NewFileAppender(path string, options ...FileOption) (*FileAppender, error) {
	cleanPath, err := validateLogFilePath(path)
	if err != nil {
		return nil, err
	}
	appender := &FileAppender{
		name:       "file",
		path:       cleanPath,
		layout:     NewDefaultLayout(),
		bufferSize: DefaultFileBufferSize,
	}
	for _, option := range options {
		if option != nil {
			option(appender)
		}
	}
	if strings.TrimSpace(appender.name) == "" {
		return nil, fmt.Errorf("goark-log: file appender name is empty")
	}
	if appender.layout == nil {
		appender.layout = NewDefaultLayout()
	}
	if appender.bufferSize < 0 {
		return nil, fmt.Errorf("goark-log: file buffer size must be >= 0")
	}
	file, err := openLogFile(cleanPath)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("goark-log: stat log file %q: %w", cleanPath, err)
	}
	appender.file = file
	if appender.bufferSize > 0 {
		appender.writer = bufio.NewWriterSize(file, appender.bufferSize)
	}
	if info.Size() == 0 {
		if _, err := appender.writeHeaderLocked(); err != nil {
			_ = appender.flushLocked()
			_ = file.Close()
			return nil, fmt.Errorf("goark-log: write file appender %q header: %w", appender.Name(), err)
		}
	}
	return appender, nil
}

func (a *FileAppender) Name() string {
	if a == nil || a.name == "" {
		return "file"
	}
	return a.name
}

func (a *FileAppender) Append(ctx context.Context, event Event) error {
	if a == nil {
		return fmt.Errorf("goark-log: file appender is nil")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer releaseBuffer(buf)
	if err := a.layout.Format(buf, event); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed || a.file == nil {
		return fmt.Errorf("goark-log: file appender %q is closed", a.Name())
	}
	var err error
	if a.writer != nil {
		_, err = a.writer.Write(buf.Bytes())
		if err == nil && a.flushOnWrite {
			err = a.writer.Flush()
		}
		return err
	}
	_, err = a.file.Write(buf.Bytes())
	return err
}

// Flush 把缓冲日志刷入操作系统文件缓存。
func (a *FileAppender) Flush() error {
	if a == nil {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.flushLocked()
}

func (a *FileAppender) Close() error {
	if a == nil {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed {
		return nil
	}
	a.closed = true
	_, footerErr := a.writeFooterLocked()
	flushErr := a.flushLocked()
	if a.file == nil {
		return errors.Join(footerErr, flushErr)
	}
	err := a.file.Close()
	a.file = nil
	a.writer = nil
	return errors.Join(footerErr, flushErr, err)
}

func (a *FileAppender) flushLocked() error {
	if a == nil || a.writer == nil {
		return nil
	}
	return a.writer.Flush()
}

func (a *FileAppender) writeHeaderLocked() (int, error) {
	writer := a.outputWriterLocked()
	if writer == nil {
		return 0, nil
	}
	return writeLayoutHeader(writer, a.layout)
}

func (a *FileAppender) writeFooterLocked() (int, error) {
	writer := a.outputWriterLocked()
	if writer == nil {
		return 0, nil
	}
	return writeLayoutFooter(writer, a.layout)
}

func (a *FileAppender) outputWriterLocked() io.Writer {
	if a == nil {
		return nil
	}
	if a.writer != nil {
		return a.writer
	}
	return a.file
}

func validateLogFilePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("goark-log: log file path is empty")
	}
	cleanPath := filepath.Clean(path)
	info, err := os.Stat(cleanPath)
	if err == nil && info.IsDir() {
		return "", fmt.Errorf("goark-log: log file path %q is a directory", path)
	}
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("goark-log: stat log file %q: %w", path, err)
	}
	return cleanPath, nil
}

func openLogFile(path string) (*os.File, error) {
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("goark-log: create log directory %q: %w", dir, err)
		}
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("goark-log: open log file %q: %w", path, err)
	}
	return file, nil
}
