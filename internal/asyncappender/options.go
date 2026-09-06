package asyncappender

import "goark.dev/log/internal/asyncruntime"

// Option 调整 Appender。
type Option func(*Appender)

// WithName 设置 appender 名称。
func WithName(name string) Option {
	return func(appender *Appender) {
		appender.name = name
	}
}

// WithQueueSize 设置异步队列长度。
func WithQueueSize(size int) Option {
	return func(appender *Appender) {
		appender.queueSize = size
	}
}

// WithBatchSize 设置后台协程单次批量写出上限。
func WithBatchSize(size int) Option {
	return func(appender *Appender) {
		appender.batchSize = size
	}
}

// WithOverflowStrategy 设置队列满时的处理策略。
func WithOverflowStrategy(strategy asyncruntime.OverflowStrategy) Option {
	return func(appender *Appender) {
		appender.strategy = strategy
	}
}

// WithWaitStrategy 设置异步队列等待策略。
func WithWaitStrategy(strategy asyncruntime.WaitStrategy) Option {
	return func(appender *Appender) {
		appender.waitStrategy = strategy
	}
}

// WithWaitOptions 设置异步队列等待策略参数。
func WithWaitOptions(options asyncruntime.WaitOptions) Option {
	return func(appender *Appender) {
		appender.waitOptions = options
	}
}

// WithErrorHandler 设置异步后台写入失败处理器。
func WithErrorHandler(handler asyncruntime.ErrorHandler) Option {
	return func(appender *Appender) {
		appender.errorHandler = handler
	}
}

// WithCloseAppenders 设置关闭 async 时是否同时关闭下游 appender。
func WithCloseAppenders(enabled bool) Option {
	return func(appender *Appender) {
		appender.closeAppenders = enabled
	}
}
