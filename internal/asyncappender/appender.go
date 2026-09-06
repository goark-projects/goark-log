package asyncappender

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"goark.dev/log/internal/asyncruntime"
	"goark.dev/log/internal/disruptor"
	"goark.dev/log/internal/logevent"
)

// Event 是异步 appender 处理的事件快照。
type Event = logevent.Event

// Sink 是异步 appender 依赖的最小下游输出端接口。
type Sink interface {
	Name() string
	Append(ctx context.Context, event Event) error
	Close() error
}

// Appender 使用后台 goroutine 串行写入下游 appender。
type Appender struct {
	name           string
	appenders      []Sink
	queueSize      int
	batchSize      int
	waitStrategy   asyncruntime.WaitStrategy
	waitOptions    asyncruntime.WaitOptions
	strategy       asyncruntime.OverflowStrategy
	errorHandler   asyncruntime.ErrorHandler
	closeAppenders bool

	queue     *disruptor.RingBuffer[entry]
	closing   chan struct{}
	done      chan struct{}
	stateMu   sync.RWMutex
	closed    bool
	producers sync.WaitGroup
	workers   sync.WaitGroup
	dropped   atomic.Uint64
	failed    atomic.Uint64
}

type entry struct {
	event Event
}

// New 创建异步 appender。
func New(appenders []Sink, options ...Option) (*Appender, error) {
	appender := &Appender{
		name:         "async",
		queueSize:    asyncruntime.DefaultAsyncQueueSize,
		batchSize:    asyncruntime.DefaultAsyncAppenderBatchSize,
		waitStrategy: asyncruntime.WaitBlock,
		strategy:     asyncruntime.OverflowBlock,
	}
	for _, option := range options {
		if option != nil {
			option(appender)
		}
	}
	if err := appender.validate(appenders); err != nil {
		return nil, err
	}
	strategy, err := asyncruntime.ParseOverflowStrategy(string(appender.strategy))
	if err != nil {
		return nil, err
	}
	normalizedQueueSize, err := asyncruntime.NormalizeQueueSize(
		appender.queueSize,
		asyncruntime.DefaultAsyncQueueSize,
	)
	if err != nil {
		return nil, err
	}
	waitStrategy, err := asyncruntime.ParseWaitStrategy(string(appender.waitStrategy))
	if err != nil {
		return nil, err
	}
	appender.queueSize = normalizedQueueSize
	if appender.batchSize > appender.queueSize {
		appender.batchSize = appender.queueSize
	}
	appender.strategy = strategy
	appender.waitStrategy = waitStrategy
	appender.appenders = append([]Sink(nil), appenders...)
	appender.queue, err = disruptor.NewRingBuffer[entry](
		appender.queueSize,
		asyncruntime.NewWaitStrategyWithOptions(appender.waitStrategy, appender.waitOptions),
	)
	if err != nil {
		return nil, err
	}
	appender.closing = make(chan struct{})
	appender.done = make(chan struct{})
	appender.workers.Add(1)
	go appender.run()
	return appender, nil
}

func (a *Appender) Name() string {
	if a == nil || a.name == "" {
		return "async"
	}
	return a.name
}

func (a *Appender) Append(ctx context.Context, event Event) error {
	if a == nil {
		return fmt.Errorf("goark-log: async appender is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	item := entry{event: event}
	if !a.beginAppend() {
		return fmt.Errorf("goark-log: async appender %q is closed", a.Name())
	}
	defer a.producers.Done()
	switch a.strategy {
	case asyncruntime.OverflowBlock:
		return a.enqueueBlocking(ctx, item)
	case asyncruntime.OverflowDrop:
		return a.enqueueOrDrop(item)
	case asyncruntime.OverflowDropDebug:
		return a.enqueueDropDebug(ctx, item)
	case asyncruntime.OverflowSyncFallback:
		return a.enqueueOrSync(ctx, item)
	default:
		return fmt.Errorf("goark-log: unsupported async overflow strategy %q", a.strategy)
	}
}

func (a *Appender) Close() error {
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
func (a *Appender) Dropped() uint64 {
	if a == nil {
		return 0
	}
	return a.dropped.Load()
}

// Failed 返回后台写入失败的日志数量。
func (a *Appender) Failed() uint64 {
	if a == nil {
		return 0
	}
	return a.failed.Load()
}

// QueueSize 返回运行期归一化后的队列长度。
func (a *Appender) QueueSize() int {
	if a == nil {
		return 0
	}
	return a.queueSize
}

// BatchSize 返回运行期批量写出上限。
func (a *Appender) BatchSize() int {
	if a == nil {
		return 0
	}
	return a.batchSize
}

// WaitStrategy 返回运行期等待策略。
func (a *Appender) WaitStrategy() asyncruntime.WaitStrategy {
	if a == nil {
		return ""
	}
	return a.waitStrategy
}

// WaitOptions 返回运行期等待策略参数。
func (a *Appender) WaitOptions() asyncruntime.WaitOptions {
	if a == nil {
		return asyncruntime.WaitOptions{}
	}
	return a.waitOptions
}
