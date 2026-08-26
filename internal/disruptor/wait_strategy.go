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
	switch name {
	case WaitSleeping:
		return sleepingWaitStrategy{}
	case WaitYielding:
		return yieldingWaitStrategy{}
	case WaitBusySpin:
		return busySpinWaitStrategy{}
	default:
		return blockingWaitStrategy{}
	}
}

type blockingWaitStrategy struct{}

func (blockingWaitStrategy) Wait(ctx context.Context, signal <-chan struct{}, interrupt <-chan struct{}, ready func() bool) error {
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

func (blockingWaitStrategy) Signal(signal chan<- struct{}) {
	signalOnce(signal)
}

type sleepingWaitStrategy struct{}

func (sleepingWaitStrategy) Wait(ctx context.Context, signal <-chan struct{}, interrupt <-chan struct{}, ready func() bool) error {
	spins := 200
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
		time.Sleep(100 * time.Microsecond)
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
