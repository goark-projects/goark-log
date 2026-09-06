package goarklog

import (
	"io"
	"io/fs"
	"log/slog"

	internalfileappender "goark.dev/log/internal/fileappender"
	internaljsonappender "goark.dev/log/internal/jsonappender"
	internalrouter "goark.dev/log/internal/router"
)

const (
	// DefaultFileBufferSize 是文件 appender 默认缓冲大小。
	DefaultFileBufferSize = internalfileappender.DefaultFileBufferSize
)

// Appender 是日志事件的最终写出端。
type Appender = internalrouter.Appender

// AppenderRef 描述一次到 appender 的结构化引用。
type AppenderRef = internalrouter.AppenderRef

// AppenderRefOption 调整结构化 appender 引用。
type AppenderRefOption = internalrouter.AppenderRefOption

// FilteredAppender 为任意 appender 增加过滤器链。
type FilteredAppender = internalrouter.FilteredAppender

// NewAppenderRef 创建结构化 appender 引用。
func NewAppenderRef(ref string, options ...AppenderRefOption) AppenderRef {
	return internalrouter.NewAppenderRef(ref, options...)
}

// WithAppenderRefLevel 设置当前引用独有的级别下限。
func WithAppenderRefLevel(level slog.Level) AppenderRefOption {
	return internalrouter.WithAppenderRefLevel(level)
}

// WithAppenderRefLocation 设置当前引用是否采集调用位置。
func WithAppenderRefLocation(enabled bool) AppenderRefOption {
	return internalrouter.WithAppenderRefLocation(enabled)
}

// WithAppenderRefFilters 设置当前引用独有的过滤器链。
func WithAppenderRefFilters(filters ...Filter) AppenderRefOption {
	return internalrouter.WithAppenderRefFilters(filters...)
}

// NewFilteredAppender 创建带过滤器链的 appender。
func NewFilteredAppender(delegate Appender, filters ...Filter) (*FilteredAppender, error) {
	return internalrouter.NewFilteredAppender(delegate, filters...)
}

// ConsoleAppender 把日志写入 stdout、stderr 或自定义 writer。
type ConsoleAppender = internalfileappender.ConsoleAppender

// ConsoleOption 调整 ConsoleAppender。
type ConsoleOption = internalfileappender.ConsoleOption

// FileAppender 把日志追加写入普通文件。
type FileAppender = internalfileappender.FileAppender

// FileOption 调整 FileAppender。
type FileOption = internalfileappender.FileOption

// JSONAppender 将事件直接编码为单行 JSON，适合极低分配热路径。
type JSONAppender = internaljsonappender.Appender

// JSONAppenderOption 调整 JSONAppender。
type JSONAppenderOption = internaljsonappender.Option

// WithConsoleName 设置 appender 名称。
func WithConsoleName(name string) ConsoleOption {
	return internalfileappender.WithConsoleName(name)
}

// WithConsoleWriter 设置输出 writer，主要用于测试和嵌入式场景。
func WithConsoleWriter(writer io.Writer) ConsoleOption {
	return internalfileappender.WithConsoleWriter(writer)
}

// WithConsoleLayout 设置日志布局。
func WithConsoleLayout(layout Layout) ConsoleOption {
	return internalfileappender.WithConsoleLayout(layout)
}

// NewConsoleAppender 创建控制台 appender。
func NewConsoleAppender(options ...ConsoleOption) *ConsoleAppender {
	return internalfileappender.NewConsoleAppender(options...)
}

// WithFileName 设置 appender 名称。
func WithFileName(name string) FileOption {
	return internalfileappender.WithFileName(name)
}

// WithFileLayout 设置日志布局。
func WithFileLayout(layout Layout) FileOption {
	return internalfileappender.WithFileLayout(layout)
}

// WithFileBufferSize 设置文件写缓冲大小，0 表示禁用缓冲。
func WithFileBufferSize(size int) FileOption {
	return internalfileappender.WithFileBufferSize(size)
}

// WithFileFlushOnWrite 设置每次写入后立即 flush。
func WithFileFlushOnWrite(enabled bool) FileOption {
	return internalfileappender.WithFileFlushOnWrite(enabled)
}

// WithFileAppend 设置打开文件时是否追加到已有内容。
func WithFileAppend(enabled bool) FileOption {
	return internalfileappender.WithFileAppend(enabled)
}

// WithFileCreateOnDemand 设置是否延迟到首次写入时创建文件。
func WithFileCreateOnDemand(enabled bool) FileOption {
	return internalfileappender.WithFileCreateOnDemand(enabled)
}

// WithFilePermissions 设置新建日志文件权限。
func WithFilePermissions(permissions fs.FileMode) FileOption {
	return internalfileappender.WithFilePermissions(permissions)
}

// NewFileAppender 创建普通文件 appender。
func NewFileAppender(path string, options ...FileOption) (*FileAppender, error) {
	return internalfileappender.NewFileAppender(path, options...)
}

// WithJSONAppenderName 设置 appender 名称。
func WithJSONAppenderName(name string) JSONAppenderOption {
	return internaljsonappender.WithName(name)
}

// WithJSONAppenderWriter 设置输出 writer，主要用于测试、基准和嵌入式直写场景。
func WithJSONAppenderWriter(writer io.Writer) JSONAppenderOption {
	return internaljsonappender.WithWriter(writer)
}

// WithJSONAppenderBufferSize 设置文件输出缓冲大小，0 表示禁用应用层缓冲。
func WithJSONAppenderBufferSize(size int) JSONAppenderOption {
	return internaljsonappender.WithBufferSize(size)
}

// WithJSONAppenderFlushOnWrite 设置每次写入后立即刷新应用层缓冲。
func WithJSONAppenderFlushOnWrite(enabled bool) JSONAppenderOption {
	return internaljsonappender.WithFlushOnWrite(enabled)
}

// NewJSONAppender 创建 JSON 直写 appender。
func NewJSONAppender(options ...JSONAppenderOption) *JSONAppender {
	return internaljsonappender.New(options...)
}

// NewJSONFileAppender 创建面向文件的 JSON 直写 appender。
func NewJSONFileAppender(path string, options ...JSONAppenderOption) (*JSONAppender, error) {
	return internaljsonappender.NewFile(path, options...)
}
