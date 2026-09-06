package goarklog

import (
	"io"
	"io/fs"
	"log/slog"

	internalasyncappender "goark.dev/log/internal/asyncappender"
	internalasync "goark.dev/log/internal/asyncruntime"
	internalfileappender "goark.dev/log/internal/fileappender"
	internaljsonappender "goark.dev/log/internal/jsonappender"
	internalrouter "goark.dev/log/internal/router"
)

const (
	// DefaultFileBufferSize 是文件 appender 默认缓冲大小。
	DefaultFileBufferSize = internalfileappender.DefaultFileBufferSize
	// DefaultAsyncQueueSize 是 AsyncAppender 默认有界队列长度。
	DefaultAsyncQueueSize = internalasync.DefaultAsyncQueueSize
	// DefaultAsyncAppenderBatchSize 是 AsyncAppender 默认批量写出数量。
	DefaultAsyncAppenderBatchSize = internalasync.DefaultAsyncAppenderBatchSize
	// DefaultAsyncLoggerQueueSize 是 AsyncLogger 默认队列长度。
	DefaultAsyncLoggerQueueSize = internalasync.DefaultAsyncLoggerQueueSize
	// DefaultAsyncLoggerBatchSize 是 AsyncLogger 默认批量写出数量。
	DefaultAsyncLoggerBatchSize = internalasync.DefaultAsyncLoggerBatchSize
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

// AsyncOverflowStrategy 定义异步队列满时的处理策略。
type AsyncOverflowStrategy = internalasync.OverflowStrategy

const (
	AsyncOverflowBlock        AsyncOverflowStrategy = internalasync.OverflowBlock
	AsyncOverflowDrop         AsyncOverflowStrategy = internalasync.OverflowDrop
	AsyncOverflowDropDebug    AsyncOverflowStrategy = internalasync.OverflowDropDebug
	AsyncOverflowSyncFallback AsyncOverflowStrategy = internalasync.OverflowSyncFallback
)

// AsyncWaitStrategy 定义异步队列等待策略。
type AsyncWaitStrategy = internalasync.WaitStrategy

const (
	AsyncWaitBlock AsyncWaitStrategy = internalasync.WaitBlock
	AsyncWaitSleep AsyncWaitStrategy = internalasync.WaitSleep
	AsyncWaitYield AsyncWaitStrategy = internalasync.WaitYield
	AsyncWaitSpin  AsyncWaitStrategy = internalasync.WaitSpin
)

// AsyncWaitOptions 描述异步等待策略的细粒度参数，零值保持默认行为。
type AsyncWaitOptions = internalasync.WaitOptions

// AsyncErrorHandler 处理异步后台写入失败。
type AsyncErrorHandler = internalasync.ErrorHandler

// AsyncErrorHandlerFunc 把函数适配为 AsyncErrorHandler。
type AsyncErrorHandlerFunc = internalasync.ErrorHandlerFunc

// AsyncLoggerOptions 描述 Handler 层异步日志管线。
type AsyncLoggerOptions = internalasync.LoggerOptions

// AsyncAppender 使用后台 goroutine 串行写入下游 appender。
type AsyncAppender = internalasyncappender.Appender

// AsyncOption 调整 AsyncAppender。
type AsyncOption = internalasyncappender.Option

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

// ParseAsyncOverflowStrategy 解析异步队列满策略。
func ParseAsyncOverflowStrategy(value string) (AsyncOverflowStrategy, error) {
	return internalasync.ParseOverflowStrategy(value)
}

// ParseAsyncWaitStrategy 解析异步队列等待策略。
func ParseAsyncWaitStrategy(value string) (AsyncWaitStrategy, error) {
	return internalasync.ParseWaitStrategy(value)
}

// WithAsyncName 设置 appender 名称。
func WithAsyncName(name string) AsyncOption {
	return internalasyncappender.WithName(name)
}

// WithAsyncQueueSize 设置异步队列长度。
func WithAsyncQueueSize(size int) AsyncOption {
	return internalasyncappender.WithQueueSize(size)
}

// WithAsyncBatchSize 设置后台协程单次批量写出上限。
func WithAsyncBatchSize(size int) AsyncOption {
	return internalasyncappender.WithBatchSize(size)
}

// WithAsyncOverflowStrategy 设置队列满时的处理策略。
func WithAsyncOverflowStrategy(strategy AsyncOverflowStrategy) AsyncOption {
	return internalasyncappender.WithOverflowStrategy(strategy)
}

// WithAsyncWaitStrategy 设置异步队列等待策略。
func WithAsyncWaitStrategy(strategy AsyncWaitStrategy) AsyncOption {
	return internalasyncappender.WithWaitStrategy(strategy)
}

// WithAsyncWaitOptions 设置异步队列等待策略参数。
func WithAsyncWaitOptions(options AsyncWaitOptions) AsyncOption {
	return internalasyncappender.WithWaitOptions(options)
}

// WithAsyncErrorHandler 设置异步后台写入失败处理器。
func WithAsyncErrorHandler(handler AsyncErrorHandler) AsyncOption {
	return internalasyncappender.WithErrorHandler(handler)
}

// WithAsyncCloseAppenders 设置关闭 async 时是否同时关闭下游 appender。
func WithAsyncCloseAppenders(enabled bool) AsyncOption {
	return internalasyncappender.WithCloseAppenders(enabled)
}

// NewAsyncAppender 创建异步 appender。
func NewAsyncAppender(appenders []Appender, options ...AsyncOption) (*AsyncAppender, error) {
	return internalasyncappender.New(asyncAppenderSinks(appenders), options...)
}

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

func asyncAppenderSinks(appenders []Appender) []internalasyncappender.Sink {
	if len(appenders) == 0 {
		return nil
	}
	converted := make([]internalasyncappender.Sink, 0, len(appenders))
	for _, appender := range appenders {
		converted = append(converted, appender)
	}
	return converted
}

func normalizeAsyncLoggerOptions(options AsyncLoggerOptions) (AsyncLoggerOptions, error) {
	return internalasync.NormalizeLoggerOptions(options)
}

func sameAsyncLoggerRuntimeOptions(left AsyncLoggerOptions, right AsyncLoggerOptions) bool {
	return internalasync.SameLoggerRuntimeOptions(left, right)
}

func isAsyncAppender(appender Appender) bool {
	switch value := appender.(type) {
	case *AsyncAppender:
		return true
	case *FilteredAppender:
		return isAsyncAppender(value.Delegate())
	default:
		return false
	}
}
