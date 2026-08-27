package asyncruntime

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"goark.dev/log/internal/disruptor"
	"goark.dev/log/internal/logevent"
)

const (
	// DefaultAsyncQueueSize 是 AsyncAppender 默认有界队列长度。
	DefaultAsyncQueueSize = 1024
	// DefaultAsyncAppenderBatchSize 是 AsyncAppender 默认批量写出数量。
	DefaultAsyncAppenderBatchSize = 64
	// DefaultAsyncLoggerQueueSize 是 Handler 异步管线默认队列长度。
	DefaultAsyncLoggerQueueSize = 4096
	// DefaultAsyncLoggerBatchSize 是 Handler 异步管线默认批量写出数量。
	DefaultAsyncLoggerBatchSize = 64
)

// OverflowStrategy 定义异步队列满时的处理策略。
type OverflowStrategy string

const (
	OverflowBlock        OverflowStrategy = "block"
	OverflowDrop         OverflowStrategy = "drop"
	OverflowDropDebug    OverflowStrategy = "drop-debug"
	OverflowSyncFallback OverflowStrategy = "sync-fallback"
)

// ParseOverflowStrategy 解析异步队列满策略。
func ParseOverflowStrategy(value string) (OverflowStrategy, error) {
	switch OverflowStrategy(strings.ToLower(strings.TrimSpace(value))) {
	case "", OverflowBlock, "blocking":
		return OverflowBlock, nil
	case OverflowDrop, "discard", "discard-newest":
		return OverflowDrop, nil
	case OverflowDropDebug, "dropdebug", "discard-debug", "discarddebug":
		return OverflowDropDebug, nil
	case OverflowSyncFallback, "sync", "synchronous", "synchronize":
		return OverflowSyncFallback, nil
	default:
		return "", fmt.Errorf("goark-log: unsupported async overflow strategy %q", value)
	}
}

// WaitStrategy 定义异步队列等待策略。
type WaitStrategy string

const (
	WaitBlock WaitStrategy = "block"
	WaitSleep WaitStrategy = "sleep"
	WaitYield WaitStrategy = "yield"
	WaitSpin  WaitStrategy = "spin"
)

// WaitOptions 描述异步等待策略的细粒度参数，零值保持默认行为。
type WaitOptions struct {
	Retries   int
	SleepTime time.Duration
	Timeout   time.Duration
}

// ErrorHandler 处理异步后台写入失败。
type ErrorHandler interface {
	HandleAsyncError(ctx context.Context, err error, event logevent.Event)
}

// ErrorHandlerFunc 把函数适配为 ErrorHandler。
type ErrorHandlerFunc func(ctx context.Context, err error, event logevent.Event)

// HandleAsyncError 执行异步错误处理函数。
func (f ErrorHandlerFunc) HandleAsyncError(ctx context.Context, err error, event logevent.Event) {
	if f != nil {
		f(ctx, err, event)
	}
}

// LoggerOptions 描述 Handler 层异步日志管线。
type LoggerOptions struct {
	Enabled          bool
	QueueSize        int
	BatchSize        int
	OverflowStrategy OverflowStrategy
	WaitStrategy     WaitStrategy
	WaitOptions      WaitOptions
	IncludeLocation  bool
	ErrorHandler     ErrorHandler
}

// NormalizeLoggerOptions 把用户配置转成运行期稳定值，便于 reload 做精确一致性校验。
func NormalizeLoggerOptions(options LoggerOptions) (LoggerOptions, error) {
	if !options.Enabled {
		return LoggerOptions{}, nil
	}
	queueSize := options.QueueSize
	if queueSize <= 0 {
		queueSize = DefaultAsyncLoggerQueueSize
	}
	queueSize, err := NormalizeQueueSize(queueSize, DefaultAsyncLoggerQueueSize)
	if err != nil {
		return LoggerOptions{}, err
	}
	batchSize := options.BatchSize
	if batchSize <= 0 {
		batchSize = DefaultAsyncLoggerBatchSize
	}
	if batchSize > queueSize {
		batchSize = queueSize
	}
	strategy, err := ParseOverflowStrategy(string(options.OverflowStrategy))
	if err != nil {
		return LoggerOptions{}, err
	}
	wait, err := ParseWaitStrategy(string(options.WaitStrategy))
	if err != nil {
		return LoggerOptions{}, err
	}
	if err := ValidateWaitOptions(options.WaitOptions); err != nil {
		return LoggerOptions{}, err
	}
	return LoggerOptions{
		Enabled:          true,
		QueueSize:        queueSize,
		BatchSize:        batchSize,
		OverflowStrategy: strategy,
		WaitStrategy:     wait,
		WaitOptions:      options.WaitOptions,
		IncludeLocation:  options.IncludeLocation,
		ErrorHandler:     options.ErrorHandler,
	}, nil
}

// ValidateWaitOptions 校验异步等待策略参数。
func ValidateWaitOptions(options WaitOptions) error {
	if options.Retries < 0 {
		return fmt.Errorf("goark-log: async wait retries must be >= 0")
	}
	if options.SleepTime < 0 {
		return fmt.Errorf("goark-log: async wait sleepTime is invalid")
	}
	if options.Timeout < 0 {
		return fmt.Errorf("goark-log: async wait timeout is invalid")
	}
	return nil
}

// ParseWaitStrategy 解析异步队列等待策略。
func ParseWaitStrategy(value string) (WaitStrategy, error) {
	switch WaitStrategy(strings.ToLower(strings.TrimSpace(value))) {
	case "", WaitBlock, "blocking", "timeout", "timeout-block", "timeoutblocking":
		return WaitBlock, nil
	case WaitSleep, "sleeping":
		return WaitSleep, nil
	case WaitYield, "yielding":
		return WaitYield, nil
	case WaitSpin, "busy-spin", "busyspin":
		return WaitSpin, nil
	default:
		return "", fmt.Errorf("goark-log: unsupported async wait strategy %q", value)
	}
}

// NewWaitStrategy 创建默认参数的 disruptor 等待策略。
func NewWaitStrategy(strategy WaitStrategy) disruptor.WaitStrategy {
	return NewWaitStrategyWithOptions(strategy, WaitOptions{})
}

// NewWaitStrategyWithOptions 创建带参数的 disruptor 等待策略。
func NewWaitStrategyWithOptions(strategy WaitStrategy, options WaitOptions) disruptor.WaitStrategy {
	waitOptions := disruptor.WaitStrategyOptions{
		Retries:   options.Retries,
		SleepTime: options.SleepTime,
		Timeout:   options.Timeout,
	}
	switch strategy {
	case WaitSleep:
		return disruptor.NewWaitStrategyWithOptions(disruptor.WaitSleeping, waitOptions)
	case WaitYield:
		return disruptor.NewWaitStrategyWithOptions(disruptor.WaitYielding, waitOptions)
	case WaitSpin:
		return disruptor.NewWaitStrategyWithOptions(disruptor.WaitBusySpin, waitOptions)
	default:
		return disruptor.NewWaitStrategyWithOptions(disruptor.WaitBlocking, waitOptions)
	}
}

// NormalizeQueueSize 规范化队列长度到 disruptor 要求的容量。
func NormalizeQueueSize(size int, fallback int) (int, error) {
	if size <= 0 {
		size = fallback
	}
	return disruptor.NormalizeCapacity(size)
}

// SameLoggerRuntimeOptions 比较 reload 不允许变化的异步运行期参数。
func SameLoggerRuntimeOptions(left LoggerOptions, right LoggerOptions) bool {
	return left.Enabled == right.Enabled &&
		left.QueueSize == right.QueueSize &&
		left.BatchSize == right.BatchSize &&
		left.OverflowStrategy == right.OverflowStrategy &&
		left.WaitStrategy == right.WaitStrategy &&
		left.WaitOptions == right.WaitOptions &&
		left.IncludeLocation == right.IncludeLocation
}

// LevelIsDebugOrLower 判断事件是否属于可被 drop-debug 策略丢弃的低级别日志。
func LevelIsDebugOrLower(level slog.Level) bool {
	return level <= slog.LevelDebug
}
