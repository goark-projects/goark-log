package disruptor

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"time"
)

// ErrInterrupted 表示等待被关闭信号中断。
var ErrInterrupted = errors.New("goark-log: disruptor wait interrupted")

// WaitStrategyName 是 Disruptor 等待策略名称。
type WaitStrategyName string

const (
	WaitBlocking WaitStrategyName = "block"
	WaitSleeping WaitStrategyName = "sleep"
	WaitYielding WaitStrategyName = "yield"
	WaitBusySpin WaitStrategyName = "spin"
)

// WaitStrategy 控制生产者或消费者在条件未满足时如何等待。
type WaitStrategy interface {
	Wait(ctx context.Context, signal <-chan struct{}, interrupt <-chan struct{}, ready func() bool) error
	Signal(signal chan<- struct{})
}

// WaitStrategyOptions 描述可调等待参数，零值保持默认策略。
type WaitStrategyOptions struct {
	Retries   int
	SleepTime time.Duration
	Timeout   time.Duration
}

// ParseWaitStrategyName 解析等待策略名称。
func ParseWaitStrategyName(value string) (WaitStrategyName, error) {
	switch WaitStrategyName(strings.ToLower(strings.TrimSpace(value))) {
	case "", WaitBlocking:
		return WaitBlocking, nil
	case WaitSleeping:
		return WaitSleeping, nil
	case WaitYielding:
		return WaitYielding, nil
	case WaitBusySpin, "busy-spin", "busyspin":
		return WaitBusySpin, nil
	default:
		return "", fmt.Errorf("goark-log: unsupported disruptor wait strategy %q", value)
	}
}

// NewWaitStrategy 创建等待策略。
func NewWaitStrategy(name WaitStrategyName) WaitStrategy {
	return NewWaitStrategyWithOptions(name, WaitStrategyOptions{})
}

// NewWaitStrategyWithOptions 创建带可调参数的等待策略。
func NewWaitStrategyWithOptions(name WaitStrategyName, options WaitStrategyOptions) WaitStrategy {
	switch name {
	case WaitSleeping:
		if options.Retries <= 0 {
			options.Retries = 200
		}
		if options.SleepTime <= 0 {
			options.SleepTime = 100 * time.Microsecond
		}
		return sleepingWaitStrategy{retries: options.Retries, sleepTime: options.SleepTime}
	case WaitYielding:
		return yieldingWaitStrategy{}
	case WaitBusySpin:
		return busySpinWaitStrategy{}
	default:
		return blockingWaitStrategy{timeout: options.Timeout}
	}
}

type blockingWaitStrategy struct {
	timeout time.Duration
}

func (s blockingWaitStrategy) Wait(ctx context.Context, signal <-chan struct{}, interrupt <-chan struct{}, ready func() bool) error {
	if s.timeout > 0 {
		return s.waitWithTimeout(ctx, signal, interrupt, ready)
	}
	for !ready() {
		select {
		case <-signal:
		case <-interrupt:
			return ErrInterrupted
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (s blockingWaitStrategy) waitWithTimeout(ctx context.Context, signal <-chan struct{}, interrupt <-chan struct{}, ready func() bool) error {
	timer := time.NewTimer(s.timeout)
	if !timer.Stop() {
		<-timer.C
	}
	defer timer.Stop()
	for !ready() {
		timer.Reset(s.timeout)
		select {
		case <-signal:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
		case <-timer.C:
		case <-interrupt:
			return ErrInterrupted
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (blockingWaitStrategy) Signal(signal chan<- struct{}) {
	signalOnce(signal)
}

type sleepingWaitStrategy struct {
	retries   int
	sleepTime time.Duration
}

func (s sleepingWaitStrategy) Wait(ctx context.Context, signal <-chan struct{}, interrupt <-chan struct{}, ready func() bool) error {
	spins := s.retries
	for !ready() {
		if err := checkInterrupted(ctx, interrupt); err != nil {
			return err
		}
		select {
		case <-signal:
		default:
		}
		if spins > 0 {
			spins--
			runtime.Gosched()
			continue
		}
		time.Sleep(s.sleepTime)
	}
	return nil
}

func (sleepingWaitStrategy) Signal(signal chan<- struct{}) {
	signalOnce(signal)
}

type yieldingWaitStrategy struct{}

func (yieldingWaitStrategy) Wait(ctx context.Context, signal <-chan struct{}, interrupt <-chan struct{}, ready func() bool) error {
	for !ready() {
		if err := checkInterrupted(ctx, interrupt); err != nil {
			return err
		}
		runtime.Gosched()
	}
	return nil
}

func (yieldingWaitStrategy) Signal(signal chan<- struct{}) {
	signalOnce(signal)
}

type busySpinWaitStrategy struct{}

func (busySpinWaitStrategy) Wait(ctx context.Context, _ <-chan struct{}, interrupt <-chan struct{}, ready func() bool) error {
	for !ready() {
		if err := checkInterrupted(ctx, interrupt); err != nil {
			return err
		}
	}
	return nil
}

func (busySpinWaitStrategy) Signal(signal chan<- struct{}) {
	signalOnce(signal)
}

func checkInterrupted(ctx context.Context, interrupt <-chan struct{}) error {
	select {
	case <-interrupt:
		return ErrInterrupted
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func signalOnce(signal chan<- struct{}) {
	select {
	case signal <- struct{}{}:
	default:
	}
}
