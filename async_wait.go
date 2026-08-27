package goarklog

import (
	"context"
	"fmt"
	"strings"
	"time"

	"goark.dev/log/internal/disruptor"
)

// AsyncWaitStrategy 定义异步队列等待策略。
type AsyncWaitStrategy string

const (
	AsyncWaitBlock AsyncWaitStrategy = "block"
	AsyncWaitSleep AsyncWaitStrategy = "sleep"
	AsyncWaitYield AsyncWaitStrategy = "yield"
	AsyncWaitSpin  AsyncWaitStrategy = "spin"
)

// AsyncWaitOptions 描述异步等待策略的细粒度参数，零值保持默认行为。
type AsyncWaitOptions struct {
	Retries   int
	SleepTime time.Duration
	Timeout   time.Duration
}

// AsyncErrorHandler 处理异步后台写入失败。
type AsyncErrorHandler interface {
	HandleAsyncError(ctx context.Context, err error, event Event)
}

// AsyncErrorHandlerFunc 把函数适配为 AsyncErrorHandler。
type AsyncErrorHandlerFunc func(ctx context.Context, err error, event Event)

// HandleAsyncError 执行异步错误处理函数。
func (f AsyncErrorHandlerFunc) HandleAsyncError(ctx context.Context, err error, event Event) {
	if f != nil {
		f(ctx, err, event)
	}
}

func validateAsyncWaitOptions(options AsyncWaitOptions) error {
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

// ParseAsyncWaitStrategy 解析异步队列等待策略。
func ParseAsyncWaitStrategy(value string) (AsyncWaitStrategy, error) {
	switch AsyncWaitStrategy(strings.ToLower(strings.TrimSpace(value))) {
	case "", AsyncWaitBlock, "blocking", "timeout", "timeout-block", "timeoutblocking":
		return AsyncWaitBlock, nil
	case AsyncWaitSleep, "sleeping":
		return AsyncWaitSleep, nil
	case AsyncWaitYield, "yielding":
		return AsyncWaitYield, nil
	case AsyncWaitSpin, "busy-spin", "busyspin":
		return AsyncWaitSpin, nil
	default:
		return "", fmt.Errorf("goark-log: unsupported async wait strategy %q", value)
	}
}

func newAsyncWaitStrategy(strategy AsyncWaitStrategy) disruptor.WaitStrategy {
	return newAsyncWaitStrategyWithOptions(strategy, AsyncWaitOptions{})
}

func newAsyncWaitStrategyWithOptions(strategy AsyncWaitStrategy, options AsyncWaitOptions) disruptor.WaitStrategy {
	waitOptions := disruptor.WaitStrategyOptions{
		Retries:   options.Retries,
		SleepTime: options.SleepTime,
		Timeout:   options.Timeout,
	}
	switch strategy {
	case AsyncWaitSleep:
		return disruptor.NewWaitStrategyWithOptions(disruptor.WaitSleeping, waitOptions)
	case AsyncWaitYield:
		return disruptor.NewWaitStrategyWithOptions(disruptor.WaitYielding, waitOptions)
	case AsyncWaitSpin:
		return disruptor.NewWaitStrategyWithOptions(disruptor.WaitBusySpin, waitOptions)
	default:
		return disruptor.NewWaitStrategyWithOptions(disruptor.WaitBlocking, waitOptions)
	}
}

func normalizeAsyncQueueSize(size int, fallback int) (int, error) {
	if size <= 0 {
		size = fallback
	}
	return disruptor.NormalizeCapacity(size)
}
