package goarklog

import (
	"context"
	"fmt"
	"goark.dev/log/internal/disruptor"
	"strings"
	"sync"
	"sync/atomic"
)

const (
	// DefaultAsyncQueueSize 是 AsyncAppender 默认有界队列长度。
	DefaultAsyncQueueSize = 1024
	// DefaultAsyncAppenderBatchSize 是 AsyncAppender 默认批量写出数量。
	DefaultAsyncAppenderBatchSize = 64
)

// AsyncOverflowStrategy 定义异步队列满时的处理策略。
type AsyncOverflowStrategy string

const (
	AsyncOverflowBlock        AsyncOverflowStrategy = "block"
	AsyncOverflowDrop         AsyncOverflowStrategy = "drop"
	AsyncOverflowDropDebug    AsyncOverflowStrategy = "drop-debug"
	AsyncOverflowSyncFallback AsyncOverflowStrategy = "sync-fallback"
)

// ParseAsyncOverflowStrategy 解析异步队列满策略。
func ParseAsyncOverflowStrategy(value string) (AsyncOverflowStrategy, error) {
	switch AsyncOverflowStrategy(strings.ToLower(strings.TrimSpace(value))) {
	case "", AsyncOverflowBlock, "blocking":
		return AsyncOverflowBlock, nil
	case AsyncOverflowDrop, "discard", "discard-newest":
		return AsyncOverflowDrop, nil
	case AsyncOverflowDropDebug, "dropdebug", "discard-debug", "discarddebug":
		return AsyncOverflowDropDebug, nil
	case AsyncOverflowSyncFallback, "sync", "synchronous", "synchronize":
		return AsyncOverflowSyncFallback, nil
	default:
		return "", fmt.Errorf("goark-log: unsupported async overflow strategy %q", value)
	}
}

// AsyncAppender 使用后台 goroutine 串行写入下游 appender。
type AsyncAppender struct {
	name           string
	appenders      []Appender
	queueSize      int
	batchSize      int
	waitStrategy   AsyncWaitStrategy
	waitOptions    AsyncWaitOptions
	strategy       AsyncOverflowStrategy
	errorHandler   AsyncErrorHandler
	closeAppenders bool

	queue     *disruptor.RingBuffer[asyncEntry]
	closing   chan struct{}
	done      chan struct{}
	stateMu   sync.RWMutex
	closed    bool
	producers sync.WaitGroup
	workers   sync.WaitGroup
	dropped   atomic.Uint64
	failed    atomic.Uint64
}

type asyncEntry struct {
	event Event
}

// NewAsyncAppender 创建异步 appender。
func NewAsyncAppender(appenders []Appender, options ...AsyncOption) (*AsyncAppender, error) {
	appender := &AsyncAppender{
		name:         "async",
		queueSize:    DefaultAsyncQueueSize,
		batchSize:    DefaultAsyncAppenderBatchSize,
		waitStrategy: AsyncWaitBlock,
		strategy:     AsyncOverflowBlock,
	}
	for _, option := range options {
		if option != nil {
			option(appender)
		}
	}
	if err := appender.validate(appenders); err != nil {
		return nil, err
	}
	normalizedQueueSize, err := normalizeAsyncQueueSize(appender.queueSize, DefaultAsyncQueueSize)
	if err != nil {
		return nil, err
	}
	waitStrategy, err := ParseAsyncWaitStrategy(string(appender.waitStrategy))
	if err != nil {
		return nil, err
	}
	appender.queueSize = normalizedQueueSize
	if appender.batchSize > appender.queueSize {
		appender.batchSize = appender.queueSize
	}
	appender.waitStrategy = waitStrategy
	appender.appenders = append([]Appender(nil), appenders...)
	appender.queue, err = disruptor.NewRingBuffer[asyncEntry](appender.queueSize, newAsyncWaitStrategyWithOptions(appender.waitStrategy, appender.waitOptions))
	if err != nil {
		return nil, err
	}
	appender.closing = make(chan struct{})
	appender.done = make(chan struct{})
	appender.workers.Add(1)
	go appender.run()
	return appender, nil
}

func (a *AsyncAppender) Name() string {
	if a == nil || a.name == "" {
		return "async"
	}
	return a.name
}

func (a *AsyncAppender) Append(ctx context.Context, event Event) error {
	if a == nil {
		return fmt.Errorf("goark-log: async appender is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	entry := asyncEntry{event: event}
	if !a.beginAppend() {
		return fmt.Errorf("goark-log: async appender %q is closed", a.Name())
	}
	defer a.producers.Done()
	switch a.strategy {
	case AsyncOverflowBlock:
		return a.enqueueBlocking(ctx, entry)
	case AsyncOverflowDrop:
		return a.enqueueOrDrop(entry)
	case AsyncOverflowDropDebug:
		return a.enqueueDropDebug(ctx, entry)
	case AsyncOverflowSyncFallback:
		return a.enqueueOrSync(ctx, entry)
	default:
		return fmt.Errorf("goark-log: unsupported async overflow strategy %q", a.strategy)
	}
}

func (a *AsyncAppender) Close() error {
	if a == nil {
		return nil
	}
	a.stateMu.Lock()
	if a.closed {
		a.stateMu.Unlock()
		return nil
	}
	a.closed = true
	close(a.closing)
	a.stateMu.Unlock()
	a.producers.Wait()
	close(a.done)
	a.workers.Wait()
	if !a.closeAppenders {
		return nil
	}
	return a.closeDelegates()
}

// Dropped 返回因队列满被丢弃的日志数量。
func (a *AsyncAppender) Dropped() uint64 {
	if a == nil {
		return 0
	}
	return a.dropped.Load()
}

// Failed 返回后台写入失败的日志数量。
func (a *AsyncAppender) Failed() uint64 {
	if a == nil {
		return 0
	}
	return a.failed.Load()
}

func (a *AsyncAppender) beginAppend() bool {
	a.stateMu.RLock()
	defer a.stateMu.RUnlock()
	if a.closed {
		return false
	}
	// producer 计数必须在状态锁内增加，避免 Close 与新的 Add 并发。
	a.producers.Add(1)
	return true
}

func (a *AsyncAppender) validate(appenders []Appender) error {
	if strings.TrimSpace(a.name) == "" {
		return fmt.Errorf("goark-log: async appender name is empty")
	}
	if a.queueSize <= 0 {
		return fmt.Errorf("goark-log: async queue size must be > 0")
	}
	if a.batchSize <= 0 {
		return fmt.Errorf("goark-log: async appender batch size must be > 0")
	}
	if _, err := ParseAsyncOverflowStrategy(string(a.strategy)); err != nil {
		return err
	}
	if err := validateAsyncWaitOptions(a.waitOptions); err != nil {
		return err
	}
	if len(appenders) == 0 {
		return fmt.Errorf("goark-log: async appender requires at least one delegate appender")
	}
	for _, appender := range appenders {
		if appender == nil {
			return fmt.Errorf("goark-log: async delegate appender is nil")
		}
	}
	return nil
}
