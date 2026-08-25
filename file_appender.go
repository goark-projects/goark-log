package goarklog

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// FileAppender 把日志追加写入普通文件。
type FileAppender struct {
	name   string
	path   string
	layout Layout
	mu     sync.Mutex
	file   *os.File
	closed bool
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

// NewFileAppender 创建普通文件 appender。
func NewFileAppender(path string, options ...FileOption) (*FileAppender, error) {
	cleanPath, err := validateLogFilePath(path)
	if err != nil {
		return nil, err
	}
	appender := &FileAppender{
		name:   "file",
		path:   cleanPath,
		layout: NewDefaultLayout(),
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
	file, err := openLogFile(cleanPath)
	if err != nil {
		return nil, err
	}
	appender.file = file
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
	_, err := a.file.Write(buf.Bytes())
	return err
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
	if a.file == nil {
		return nil
	}
	err := a.file.Close()
	a.file = nil
	return err
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
