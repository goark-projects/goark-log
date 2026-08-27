package goarklog

import (
	"io/fs"
	"time"

	internalrollingfile "goark.dev/log/internal/rollingfile"
)

const (
	// DefaultRollingMaxSize 是 RollingFileAppender 默认按大小滚动阈值。
	DefaultRollingMaxSize = internalrollingfile.DefaultRollingMaxSize
	// DefaultRollingMaxBackups 是 RollingFileAppender 默认保留档案数量。
	DefaultRollingMaxBackups = internalrollingfile.DefaultRollingMaxBackups
	// DefaultRollingActionQueueSize 是异步滚动动作队列默认长度。
	DefaultRollingActionQueueSize = internalrollingfile.DefaultRollingActionQueueSize
)

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

func withRollingClock(clock func() time.Time) RollingFileOption {
	return internalrollingfile.WithRollingClock(clock)
}
