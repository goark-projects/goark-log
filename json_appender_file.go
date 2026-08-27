package goarklog

import (
	"bufio"
	"fmt"
	"strings"

	"goark.dev/log/internal/logfile"
)

type flushWriter interface {
	Flush() error
}

// WithJSONAppenderBufferSize 设置文件输出缓冲大小，0 表示禁用应用层缓冲。
func WithJSONAppenderBufferSize(size int) JSONAppenderOption {
	return func(appender *JSONAppender) {
		appender.bufferSize = size
	}
}

// WithJSONAppenderFlushOnWrite 设置每次写入后立即刷新应用层缓冲。
func WithJSONAppenderFlushOnWrite(enabled bool) JSONAppenderOption {
	return func(appender *JSONAppender) {
		appender.flushOnWrite = enabled
	}
}

// NewJSONFileAppender 创建面向文件的 JSON 直写 appender。
func NewJSONFileAppender(path string, options ...JSONAppenderOption) (*JSONAppender, error) {
	cleanPath, err := logfile.ValidatePath(path)
	if err != nil {
		return nil, err
	}
	appender := &JSONAppender{
		name:       "json",
		bufferSize: DefaultFileBufferSize,
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
