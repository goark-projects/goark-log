package asynclogger

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"goark.dev/log/internal/asyncruntime"
	"goark.dev/log/internal/disruptor"
	"goark.dev/log/internal/logevent"
)

// DispatchFunc 是 Handler 暴露给异步管线的最小调度函数。
type DispatchFunc func(ctx context.Context, event logevent.Event, levelAccepted bool) error

// Logger 是 Handler 层异步日志管线。
type Logger struct {
	dispatch DispatchFunc
	queue    *disruptor.RingBuffer[entry]
	closing  chan struct{}
	done     chan struct{}
	options  asyncruntime.LoggerOptions

	stateMu   sync.RWMutex
	closed    bool
	producers sync.WaitGroup
	workers   sync.WaitGroup

	queueSize int
	batchSize int
	strategy  asyncruntime.OverflowStrategy
	wait      asyncruntime.WaitStrategy
	waitOpts  asyncruntime.WaitOptions
	dropped   atomic.Uint64
	failed    atomic.Uint64
}

type entry struct {
	event         logevent.Event
	levelAccepted bool
}

// New 创建 Handler 层异步日志管线。
func New(dispatch DispatchFunc, options asyncruntime.LoggerOptions) (*Logger, error) {
	if dispatch == nil {
		return nil, fmt.Errorf("goark-log: async logger dispatcher is nil")
	}
	normalized, err := asyncruntime.NormalizeLoggerOptions(options)
	if err != nil {
		return nil, err
	}
	async := &Logger{
		dispatch:  dispatch,
		closing:   make(chan struct{}),
		done:      make(chan struct{}),
		options:   normalized,
		queueSize: normalized.QueueSize,
		batchSize: normalized.BatchSize,
		strategy:  normalized.OverflowStrategy,
		wait:      normalized.WaitStrategy,
		waitOpts:  normalized.WaitOptions,
	}
	async.queue, err = disruptor.NewRingBuffer[entry](normalized.QueueSize, asyncruntime.NewWaitStrategyWithOptions(normalized.WaitStrategy, normalized.WaitOptions))
	if err != nil {
		return nil, err
	}
	async.workers.Add(1)
	go async.run()
	return async, nil
}

// Append 把事件加入 Handler 异步队列。
func (a *Logger) Append(ctx context.Context, event logevent.Event, levelAccepted bool) error {
	if a == nil {
		return fmt.Errorf("goark-log: async logger is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	item := entry{event: event, levelAccepted: levelAccepted}
	if !a.beginAppend() {
		return fmt.Errorf("goark-log: async logger is closed")
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

// Close 停止异步管线并排空队列。
func (a *Logger) Close() error {
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
	return nil
}

// Options 返回归一化后的运行期配置快照。
func (a *Logger) Options() asyncruntime.LoggerOptions {
	if a == nil {
		return asyncruntime.LoggerOptions{}
	}
	return a.options
}

// Dropped 返回因队列满被丢弃的日志数量。
func (a *Logger) Dropped() uint64 {
	if a == nil {
		return 0
	}
	return a.dropped.Load()
}

// Failed 返回后台写入失败的批次数量。
func (a *Logger) Failed() uint64 {
	if a == nil {
		return 0
	}
	return a.failed.Load()
}

// IncludeLocation 返回异步管线是否要求调用位置。
func (a *Logger) IncludeLocation() bool {
	return a != nil && a.options.IncludeLocation
}

// RemainingCapacity 返回异步队列剩余容量。
func (a *Logger) RemainingCapacity() int64 {
	if a == nil || a.queue == nil {
		return 0
	}
	return a.queue.RemainingCapacity()
}

func (a *Logger) beginAppend() bool {
	a.stateMu.RLock()
	defer a.stateMu.RUnlock()
	if a.closed {
		return false
	}
	// producer 计数必须在状态锁内增加，避免 Close 与新的 Add 并发。
	a.producers.Add(1)
	return true
}

func (a *Logger) enqueueBlocking(ctx context.Context, item entry) error {
	for {
		if a.queue.TryPublish(item) {
			return nil
		}
		err := a.queue.WaitWritable(ctx, a.closing)
		if errors.Is(err, disruptor.ErrInterrupted) {
			return fmt.Errorf("goark-log: async logger is closed")
		}
		if err != nil {
			return err
		}
	}
}

func (a *Logger) enqueueOrDrop(item entry) error {
	select {
	case <-a.closing:
		return fmt.Errorf("goark-log: async logger is closed")
	default:
		if a.queue.TryPublish(item) {
			return nil
		}
		a.dropped.Add(1)
	}
	return nil
}

func (a *Logger) enqueueDropDebug(ctx context.Context, item entry) error {
	select {
	case <-a.closing:
		return fmt.Errorf("goark-log: async logger is closed")
	default:
		if a.queue.TryPublish(item) {
			return nil
		}
		if asyncruntime.LevelIsDebugOrLower(item.event.Level) {
			a.dropped.Add(1)
			return nil
		}
		return a.enqueueBlocking(ctx, item)
	}
}

func (a *Logger) enqueueOrSync(ctx context.Context, item entry) error {
	select {
	case <-a.closing:
		return fmt.Errorf("goark-log: async logger is closed")
	default:
		if a.queue.TryPublish(item) {
			return nil
		}
		event := item.event
		event.EndOfBatch = true
		return a.dispatch(ctx, event, item.levelAccepted)
	}
}

func (a *Logger) run() {
	defer a.workers.Done()
	batch := make([]entry, 0, a.batchSize)
	for {
		if a.queue.PopBatch(&batch, a.batchSize) {
			a.flushBatch(batch)
			batch = batch[:0]
			continue
		}
		err := a.queue.WaitReadable(context.Background(), a.done)
		if errors.Is(err, disruptor.ErrInterrupted) {
			a.drainAll(&batch)
			a.flushBatch(batch)
			return
		}
	}
}

func (a *Logger) drainAll(batch *[]entry) {
	for {
		if !a.queue.PopBatch(batch, a.batchSize) {
			return
		}
		if len(*batch) >= a.batchSize {
			a.flushBatch(*batch)
			*batch = (*batch)[:0]
		}
	}
}

func (a *Logger) flushBatch(batch []entry) {
	var joined error
	for index, item := range batch {
		event := item.event
		event.EndOfBatch = index == len(batch)-1
		if err := a.dispatch(context.Background(), event, item.levelAccepted); err != nil {
			joined = errors.Join(joined, err)
			a.handleAsyncError(context.Background(), err, event)
		}
	}
	if joined != nil {
		a.failed.Add(1)
	}
}

func (a *Logger) handleAsyncError(ctx context.Context, err error, event logevent.Event) {
	if a == nil || a.options.ErrorHandler == nil || err == nil {
		return
	}
	a.options.ErrorHandler.HandleAsyncError(ctx, err, event)
}
