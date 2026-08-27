package goarklog

import (
	"io"
	"io/fs"
	"time"

	internalasyncappender "goark.dev/log/internal/asyncappender"
	internalasync "goark.dev/log/internal/asyncruntime"
	internaldelegate "goark.dev/log/internal/delegating"
	internalfileappender "goark.dev/log/internal/fileappender"
	internaljsonappender "goark.dev/log/internal/jsonappender"
	internalrollingfile "goark.dev/log/internal/rollingfile"
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
	// DefaultRollingMaxSize 是 RollingFileAppender 默认按大小滚动阈值。
	DefaultRollingMaxSize = internalrollingfile.DefaultRollingMaxSize
	// DefaultRollingMaxBackups 是 RollingFileAppender 默认保留档案数量。
	DefaultRollingMaxBackups = internalrollingfile.DefaultRollingMaxBackups
	// DefaultRollingActionQueueSize 是异步滚动动作队列默认长度。
	DefaultRollingActionQueueSize = internalrollingfile.DefaultRollingActionQueueSize
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

// FailoverAppender 在主 appender 写入失败时按顺序尝试备用 appender。
type FailoverAppender = internaldelegate.FailoverAppender

// FailoverOption 调整 FailoverAppender。
type FailoverOption = internaldelegate.FailoverOption

// RouteKeyFunc 从事件中计算路由键。
type RouteKeyFunc = internaldelegate.RouteKeyFunc

// RoutingAppender 按事件属性或自定义函数选择下游 appender。
type RoutingAppender = internaldelegate.RoutingAppender

// RoutingOption 调整 RoutingAppender。
type RoutingOption = internaldelegate.RoutingOption

// RewritePolicy 在写出前重写事件快照。
type RewritePolicy = internaldelegate.RewritePolicy

// RewriteAppender 在写出前执行事件重写。
type RewriteAppender = internaldelegate.RewriteAppender

// RewriteOption 调整 RewriteAppender。
type RewriteOption = internaldelegate.RewriteOption

// RollingFileIndexMode 定义 filePattern 中 %i 的分配策略。
type RollingFileIndexMode = internalrollingfile.RollingFileIndexMode

const (
	RollingFileIndexNoMax = internalrollingfile.RollingFileIndexNoMax
	RollingFileIndexMax   = internalrollingfile.RollingFileIndexMax
	RollingFileIndexMin   = internalrollingfile.RollingFileIndexMin
)

// RollingFileAppender 支持按大小、按时间和启动时滚动的文件 appender。
type RollingFileAppender = internalrollingfile.RollingFileAppender

// RollingFileOption 调整 RollingFileAppender。
type RollingFileOption = internalrollingfile.RollingFileOption

// RollingDeleteAction 描述滚动后执行的归档删除动作。
type RollingDeleteAction = internalrollingfile.RollingDeleteAction

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

// WithFailoverName 设置 failover appender 名称。
func WithFailoverName(name string) FailoverOption {
	return internaldelegate.WithFailoverName(name)
}

// WithFailoverCloseChildren 设置关闭 failover 时是否关闭下游 appender。
func WithFailoverCloseChildren(enabled bool) FailoverOption {
	return internaldelegate.WithFailoverCloseChildren(enabled)
}

// NewFailoverAppender 创建失败转移 appender。
func NewFailoverAppender(primary Appender, failovers []Appender, options ...FailoverOption) (*FailoverAppender, error) {
	return internaldelegate.NewFailoverAppender(primary, delegatingAppenders(failovers), options...)
}

// WithRoutingName 设置 routing appender 名称。
func WithRoutingName(name string) RoutingOption {
	return internaldelegate.WithRoutingName(name)
}

// WithRoutingAttrKey 设置按事件属性取路由键。
func WithRoutingAttrKey(key string) RoutingOption {
	return internaldelegate.WithRoutingAttrKey(key)
}

// WithRoutingDefault 设置未命中路由时的默认 appender。
func WithRoutingDefault(route Appender) RoutingOption {
	return internaldelegate.WithRoutingDefault(route)
}

// WithRoutingKeyFunc 设置自定义路由键函数。
func WithRoutingKeyFunc(routeKeyFunc RouteKeyFunc) RoutingOption {
	return internaldelegate.WithRoutingKeyFunc(routeKeyFunc)
}

// WithRoutingCloseChildren 设置关闭 routing 时是否关闭下游 appender。
func WithRoutingCloseChildren(enabled bool) RoutingOption {
	return internaldelegate.WithRoutingCloseChildren(enabled)
}

// NewRoutingAppender 创建路由 appender。
func NewRoutingAppender(routes map[string]Appender, options ...RoutingOption) (*RoutingAppender, error) {
	return internaldelegate.NewRoutingAppender(delegatingAppenderMap(routes), options...)
}

// WithRewriteName 设置 rewrite appender 名称。
func WithRewriteName(name string) RewriteOption {
	return internaldelegate.WithRewriteName(name)
}

// WithRewriteCloseDelegate 设置关闭 rewrite 时是否关闭下游 appender。
func WithRewriteCloseDelegate(enabled bool) RewriteOption {
	return internaldelegate.WithRewriteCloseDelegate(enabled)
}

// NewRewriteAppender 创建事件重写 appender。
func NewRewriteAppender(delegate Appender, policy RewritePolicy, options ...RewriteOption) (*RewriteAppender, error) {
	return internaldelegate.NewRewriteAppender(delegate, policy, options...)
}

// NewRollingFileAppender 创建滚动文件 appender。
func NewRollingFileAppender(path string, options ...RollingFileOption) (*RollingFileAppender, error) {
	return internalrollingfile.NewRollingFileAppender(path, options...)
}

// WithRollingFileName 设置 appender 名称。
func WithRollingFileName(name string) RollingFileOption {
	return internalrollingfile.WithRollingFileName(name)
}

// WithRollingFileLayout 设置日志布局。
func WithRollingFileLayout(layout Layout) RollingFileOption {
	return internalrollingfile.WithRollingFileLayout(layout)
}

// WithRollingFileBufferSize 设置滚动文件写缓冲大小，0 表示禁用缓冲。
func WithRollingFileBufferSize(size int) RollingFileOption {
	return internalrollingfile.WithRollingFileBufferSize(size)
}

// WithRollingFileFlushOnWrite 设置每次写入后立即 flush。
func WithRollingFileFlushOnWrite(enabled bool) RollingFileOption {
	return internalrollingfile.WithRollingFileFlushOnWrite(enabled)
}

// WithRollingFileAppend 设置打开活动文件时是否追加到已有内容。
func WithRollingFileAppend(enabled bool) RollingFileOption {
	return internalrollingfile.WithRollingFileAppend(enabled)
}

// WithRollingFileCreateOnDemand 设置是否延迟到首次写入时创建活动文件。
func WithRollingFileCreateOnDemand(enabled bool) RollingFileOption {
	return internalrollingfile.WithRollingFileCreateOnDemand(enabled)
}

// WithRollingFilePermissions 设置新建活动文件的权限。
func WithRollingFilePermissions(permissions fs.FileMode) RollingFileOption {
	return internalrollingfile.WithRollingFilePermissions(permissions)
}

// WithRollingMaxSize 设置按大小滚动阈值，0 表示禁用按大小滚动。
func WithRollingMaxSize(bytes int64) RollingFileOption {
	return internalrollingfile.WithRollingMaxSize(bytes)
}

// WithRollingInterval 设置按固定时间间隔滚动，0 表示禁用按时间滚动。
func WithRollingInterval(interval time.Duration) RollingFileOption {
	return internalrollingfile.WithRollingInterval(interval)
}

// WithRollingCronSchedule 设置 cron 触发滚动表达式。
func WithRollingCronSchedule(expression string) RollingFileOption {
	return internalrollingfile.WithRollingCronSchedule(expression)
}

// WithRollingTimeModulate 设置时间滚动是否对齐到时间边界。
func WithRollingTimeModulate(enabled bool) RollingFileOption {
	return internalrollingfile.WithRollingTimeModulate(enabled)
}

// WithRollingFilePattern 设置滚动档案路径模式，支持 %d{layout} 和 %i。
func WithRollingFilePattern(pattern string) RollingFileOption {
	return internalrollingfile.WithRollingFilePattern(pattern)
}

// WithRollingFileIndexMode 设置滚动索引分配策略。
func WithRollingFileIndexMode(mode RollingFileIndexMode) RollingFileOption {
	return internalrollingfile.WithRollingFileIndexMode(mode)
}

// WithRollingDirectWrite 设置是否直接写入 filePattern 指向的滚动文件。
func WithRollingDirectWrite(enabled bool) RollingFileOption {
	return internalrollingfile.WithRollingDirectWrite(enabled)
}

// WithRolloverOnStartup 设置启动时滚动已有文件。
func WithRolloverOnStartup(enabled bool) RollingFileOption {
	return internalrollingfile.WithRolloverOnStartup(enabled)
}

// WithRollingMaxBackups 设置保留档案数量，0 表示不保留历史档案。
func WithRollingMaxBackups(maxBackups int) RollingFileOption {
	return internalrollingfile.WithRollingMaxBackups(maxBackups)
}

// WithRollingMaxAge 设置档案最大保留时间，0 表示不按时间清理。
func WithRollingMaxAge(age time.Duration) RollingFileOption {
	return internalrollingfile.WithRollingMaxAge(age)
}

// WithRollingGzip 设置是否对滚动档案执行 gzip 压缩。
func WithRollingGzip(enabled bool) RollingFileOption {
	return internalrollingfile.WithRollingGzip(enabled)
}

// WithRollingAsyncActions 设置压缩和清理动作是否由后台串行执行。
func WithRollingAsyncActions(enabled bool) RollingFileOption {
	return internalrollingfile.WithRollingAsyncActions(enabled)
}

// WithRollingActionQueueSize 设置后台滚动动作队列长度。
func WithRollingActionQueueSize(size int) RollingFileOption {
	return internalrollingfile.WithRollingActionQueueSize(size)
}

// WithRollingDeleteActions 设置滚动后的归档删除动作。
func WithRollingDeleteActions(actions ...RollingDeleteAction) RollingFileOption {
	return internalrollingfile.WithRollingDeleteActions(actions...)
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

func delegatingAppenders(appenders []Appender) []internaldelegate.Appender {
	if len(appenders) == 0 {
		return nil
	}
	converted := make([]internaldelegate.Appender, 0, len(appenders))
	for _, appender := range appenders {
		converted = append(converted, appender)
	}
	return converted
}

func delegatingAppenderMap(routes map[string]Appender) map[string]internaldelegate.Appender {
	if len(routes) == 0 {
		return nil
	}
	converted := make(map[string]internaldelegate.Appender, len(routes))
	for key, appender := range routes {
		converted[key] = appender
	}
	return converted
}

func normalizeAsyncLoggerOptions(options AsyncLoggerOptions) (AsyncLoggerOptions, error) {
	return internalasync.NormalizeLoggerOptions(options)
}

func validateAsyncWaitOptions(options AsyncWaitOptions) error {
	return internalasync.ValidateWaitOptions(options)
}

func normalizeAsyncQueueSize(size int, fallback int) (int, error) {
	return internalasync.NormalizeQueueSize(size, fallback)
}

func sameAsyncLoggerRuntimeOptions(left AsyncLoggerOptions, right AsyncLoggerOptions) bool {
	return internalasync.SameLoggerRuntimeOptions(left, right)
}

func withRollingClock(clock func() time.Time) RollingFileOption {
	return internalrollingfile.WithRollingClock(clock)
}
