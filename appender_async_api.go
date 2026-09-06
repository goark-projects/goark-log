package goarklog

import (
	internalasyncappender "goark.dev/log/internal/asyncappender"
	internalasync "goark.dev/log/internal/asyncruntime"
)

const (
	// DefaultAsyncQueueSize 是 AsyncAppender 默认有界队列长度。
	DefaultAsyncQueueSize = internalasync.DefaultAsyncQueueSize
	// DefaultAsyncAppenderBatchSize 是 AsyncAppender 默认批量写出数量。
	DefaultAsyncAppenderBatchSize = internalasync.DefaultAsyncAppenderBatchSize
	// DefaultAsyncLoggerQueueSize 是 AsyncLogger 默认队列长度。
	DefaultAsyncLoggerQueueSize = internalasync.DefaultAsyncLoggerQueueSize
	// DefaultAsyncLoggerBatchSize 是 AsyncLogger 默认批量写出数量。
	DefaultAsyncLoggerBatchSize = internalasync.DefaultAsyncLoggerBatchSize
)

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
