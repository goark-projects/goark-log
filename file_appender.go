package goarklog

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"sync"

	"goark.dev/log/internal/logfile"
)

const (
	// DefaultFileBufferSize 是文件 appender 默认缓冲大小。
	DefaultFileBufferSize = 256 * 1024
)

// FileAppender 把日志追加写入普通文件。
type FileAppender struct {
	name           string
	path           string
	layout         Layout
	bufferSize     int
	flushOnWrite   bool
	append         bool
	createOnDemand bool
	permissions    fs.FileMode
	permissionsSet bool
	mu             sync.Mutex
	file           *os.File
	writer         *bufio.Writer
	closed         bool
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

// WithFileAppend 设置打开文件时是否追加到已有内容。
func WithFileAppend(enabled bool) FileOption {
	return func(appender *FileAppender) {
		appender.append = enabled
	}
}

// WithFileCreateOnDemand 设置是否延迟到首次写入时创建文件。
func WithFileCreateOnDemand(enabled bool) FileOption {
	return func(appender *FileAppender) {
		appender.createOnDemand = enabled
	}
}

// WithFilePermissions 设置新建日志文件权限。
func WithFilePermissions(permissions fs.FileMode) FileOption {
	return func(appender *FileAppender) {
		appender.permissions = permissions.Perm()
		appender.permissionsSet = true
	}
}

// NewFileAppender 创建普通文件 appender。
func NewFileAppender(path string, options ...FileOption) (*FileAppender, error) {
	cleanPath, err := logfile.ValidatePath(path)
	if err != nil {
		return nil, err
	}
	appender := &FileAppender{
		name:        "file",
		path:        cleanPath,
		layout:      NewDefaultLayout(),
		bufferSize:  DefaultFileBufferSize,
		append:      true,
		permissions: logfile.DefaultPermissions,
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
	if !appender.permissionsSet && appender.permissions == 0 {
		appender.permissions = logfile.DefaultPermissions
	}
	if !appender.createOnDemand {
		if _, err := appender.openLocked(); err != nil {
			return nil, err
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
	ctx = normalizeContext(ctx)
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
	if a.closed {
		return fmt.Errorf("goark-log: file appender %q is closed", a.Name())
	}
	if a.file == nil {
		if _, err := a.openLocked(); err != nil {
			return err
		}
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

func (a *FileAppender) openLocked() (int64, error) {
	file, err := logfile.OpenWithOptions(a.path, logfile.OpenOptions{
		Append:         a.append,
		Permissions:    a.permissions,
		PermissionsSet: a.permissionsSet,
	})
	if err != nil {
		return 0, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return 0, fmt.Errorf("goark-log: stat log file %q: %w", a.path, err)
	}
	existingSize := info.Size()
	a.file = file
	if a.bufferSize > 0 {
		a.writer = bufio.NewWriterSize(file, a.bufferSize)
	}
	if existingSize == 0 {
		if _, err := a.writeHeaderLocked(); err != nil {
			_ = a.flushLocked()
			_ = file.Close()
			a.file = nil
			a.writer = nil
			return 0, fmt.Errorf("goark-log: write file appender %q header: %w", a.Name(), err)
		}
	}
	return existingSize, nil
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
