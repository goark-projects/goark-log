package goarklog

import (
	"fmt"
	"strings"

	"goark.dev/goark-log/internal/disruptor"
)

// AsyncWaitStrategy 定义异步队列等待策略。
type AsyncWaitStrategy string

const (
	AsyncWaitBlock AsyncWaitStrategy = "block"
	AsyncWaitSleep AsyncWaitStrategy = "sleep"
	AsyncWaitYield AsyncWaitStrategy = "yield"
	AsyncWaitSpin  AsyncWaitStrategy = "spin"
)

// ParseAsyncWaitStrategy 解析异步队列等待策略。
func ParseAsyncWaitStrategy(value string) (AsyncWaitStrategy, error) {
	switch AsyncWaitStrategy(strings.ToLower(strings.TrimSpace(value))) {
	case "", AsyncWaitBlock:
		return AsyncWaitBlock, nil
	case AsyncWaitSleep:
		return AsyncWaitSleep, nil
	case AsyncWaitYield:
		return AsyncWaitYield, nil
	case AsyncWaitSpin, "busy-spin", "busyspin":
		return AsyncWaitSpin, nil
	default:
		return "", fmt.Errorf("goark-log: unsupported async wait strategy %q", value)
	}
}

func newAsyncWaitStrategy(strategy AsyncWaitStrategy) disruptor.WaitStrategy {
	switch strategy {
	case AsyncWaitSleep:
		return disruptor.NewWaitStrategy(disruptor.WaitSleeping)
	case AsyncWaitYield:
		return disruptor.NewWaitStrategy(disruptor.WaitYielding)
	case AsyncWaitSpin:
		return disruptor.NewWaitStrategy(disruptor.WaitBusySpin)
	default:
		return disruptor.NewWaitStrategy(disruptor.WaitBlocking)
	}
}

func normalizeAsyncQueueSize(size int, fallback int) (int, error) {
	if size <= 0 {
		size = fallback
	}
	return disruptor.NormalizeCapacity(size)
}
