package goarklog

// AsyncOption 调整 AsyncAppender。
type AsyncOption func(*AsyncAppender)

// WithAsyncName 设置 appender 名称。
func WithAsyncName(name string) AsyncOption {
	return func(appender *AsyncAppender) {
		appender.name = name
	}
}

// WithAsyncQueueSize 设置异步队列长度。
func WithAsyncQueueSize(size int) AsyncOption {
	return func(appender *AsyncAppender) {
		appender.queueSize = size
	}
}

// WithAsyncBatchSize 设置后台协程单次批量写出上限。
func WithAsyncBatchSize(size int) AsyncOption {
	return func(appender *AsyncAppender) {
		appender.batchSize = size
	}
}

// WithAsyncOverflowStrategy 设置队列满时的处理策略。
func WithAsyncOverflowStrategy(strategy AsyncOverflowStrategy) AsyncOption {
	return func(appender *AsyncAppender) {
		appender.strategy = strategy
	}
}

// WithAsyncWaitStrategy 设置异步队列等待策略。
func WithAsyncWaitStrategy(strategy AsyncWaitStrategy) AsyncOption {
	return func(appender *AsyncAppender) {
		appender.waitStrategy = strategy
	}
}

// WithAsyncWaitOptions 设置异步队列等待策略参数。
func WithAsyncWaitOptions(options AsyncWaitOptions) AsyncOption {
	return func(appender *AsyncAppender) {
		appender.waitOptions = options
	}
}

// WithAsyncErrorHandler 设置异步后台写入失败处理器。
func WithAsyncErrorHandler(handler AsyncErrorHandler) AsyncOption {
	return func(appender *AsyncAppender) {
		appender.errorHandler = handler
	}
}

// WithAsyncCloseAppenders 设置关闭 async 时是否同时关闭下游 appender。
func WithAsyncCloseAppenders(enabled bool) AsyncOption {
	return func(appender *AsyncAppender) {
		appender.closeAppenders = enabled
	}
}
