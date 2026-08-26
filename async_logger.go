package goarklog

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"

	"goark.dev/log/internal/disruptor"
)

const (
	// DefaultAsyncLoggerQueueSize 是 AsyncLogger 默认队列长度。
	DefaultAsyncLoggerQueueSize = 4096
	// DefaultAsyncLoggerBatchSize 是 AsyncLogger 默认批量写出数量。
	DefaultAsyncLoggerBatchSize = 64
)

// AsyncLoggerOptions 描述 Handler 层异步日志管线。
type AsyncLoggerOptions struct {
	Enabled          bool
	QueueSize        int
	BatchSize        int
	OverflowStrategy AsyncOverflowStrategy
	WaitStrategy     AsyncWaitStrategy
	WaitOptions      AsyncWaitOptions
	IncludeLocation  bool
	ErrorHandler     AsyncErrorHandler
}

type asyncLogger struct {
	handler *Handler
	queue   *disruptor.RingBuffer[asyncLoggerEntry]
	closing chan struct{}
	done    chan struct{}
	options AsyncLoggerOptions

	stateMu   sync.RWMutex
	closed    bool
	producers sync.WaitGroup
	workers   sync.WaitGroup

	queueSize int
	batchSize int
	strategy  AsyncOverflowStrategy
	wait      AsyncWaitStrategy
	waitOpts  AsyncWaitOptions
	dropped   atomic.Uint64
	failed    atomic.Uint64
}

type asyncLoggerEntry struct {
	event         Event
	levelAccepted bool
}

func newAsyncLogger(handler *Handler, options AsyncLoggerOptions) (*asyncLogger, error) {
	if handler == nil {
		return nil, fmt.Errorf("goark-log: async logger handler is nil")
	}
	normalized, err := normalizeAsyncLoggerOptions(options)
	if err != nil {
		return nil, err
	}
	async := &asyncLogger{
		handler:   handler,
		closing:   make(chan struct{}),
		done:      make(chan struct{}),
		options:   normalized,
		queueSize: normalized.QueueSize,
		batchSize: normalized.BatchSize,
		strategy:  normalized.OverflowStrategy,
		wait:      normalized.WaitStrategy,
		waitOpts:  normalized.WaitOptions,
	}
	async.queue, err = disruptor.NewRingBuffer[asyncLoggerEntry](normalized.QueueSize, newAsyncWaitStrategyWithOptions(normalized.WaitStrategy, normalized.WaitOptions))
	if err != nil {
		return nil, err
	}
	async.workers.Add(1)
	go async.run()
	return async, nil
}

// normalizeAsyncLoggerOptions 把用户配置转成运行期稳定值，便于 reload 做精确一致性校验。
func normalizeAsyncLoggerOptions(options AsyncLoggerOptions) (AsyncLoggerOptions, error) {
	if !options.Enabled {
		return AsyncLoggerOptions{}, nil
	}
	queueSize := options.QueueSize
	if queueSize <= 0 {
		queueSize = DefaultAsyncLoggerQueueSize
	}
	queueSize, err := normalizeAsyncQueueSize(queueSize, DefaultAsyncLoggerQueueSize)
	if err != nil {
		return AsyncLoggerOptions{}, err
	}
	batchSize := options.BatchSize
	if batchSize <= 0 {
		batchSize = DefaultAsyncLoggerBatchSize
	}
	if batchSize > queueSize {
		batchSize = queueSize
	}
	strategy, err := ParseAsyncOverflowStrategy(string(options.OverflowStrategy))
	if err != nil {
		return AsyncLoggerOptions{}, err
	}
	wait, err := ParseAsyncWaitStrategy(string(options.WaitStrategy))
	if err != nil {
		return AsyncLoggerOptions{}, err
	}
	if err := validateAsyncWaitOptions(options.WaitOptions); err != nil {
		return AsyncLoggerOptions{}, err
	}
	return AsyncLoggerOptions{
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

func sameAsyncLoggerRuntimeOptions(left AsyncLoggerOptions, right AsyncLoggerOptions) bool {
	return left.Enabled == right.Enabled &&
		left.QueueSize == right.QueueSize &&
		left.BatchSize == right.BatchSize &&
		left.OverflowStrategy == right.OverflowStrategy &&
		left.WaitStrategy == right.WaitStrategy &&
		left.WaitOptions == right.WaitOptions &&
		left.IncludeLocation == right.IncludeLocation
}

func (a *asyncLogger) append(ctx context.Context, event Event, levelAccepted bool) error {
	if a == nil {
		return fmt.Errorf("goark-log: async logger is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	entry := asyncLoggerEntry{event: event, levelAccepted: levelAccepted}
	if !a.beginAppend() {
		return fmt.Errorf("goark-log: async logger is closed")
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

func (a *asyncLogger) close() error {
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

func (a *asyncLogger) beginAppend() bool {
	a.stateMu.RLock()
	defer a.stateMu.RUnlock()
	if a.closed {
		return false
	}
	// producer 计数必须在状态锁内增加，避免 Close 与新的 Add 并发。
	a.producers.Add(1)
	return true
}

func (a *asyncLogger) enqueueBlocking(ctx context.Context, entry asyncLoggerEntry) error {
	for {
		if a.queue.TryPublish(entry) {
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

func (a *asyncLogger) enqueueOrDrop(entry asyncLoggerEntry) error {
	select {
	case <-a.closing:
		return fmt.Errorf("goark-log: async logger is closed")
	default:
		if a.queue.TryPublish(entry) {
			return nil
		}
		a.dropped.Add(1)
	}
	return nil
}

func (a *asyncLogger) enqueueDropDebug(ctx context.Context, entry asyncLoggerEntry) error {
	select {
	case <-a.closing:
		return fmt.Errorf("goark-log: async logger is closed")
	default:
		if a.queue.TryPublish(entry) {
			return nil
		}
		if entry.event.Level <= slog.LevelDebug {
			a.dropped.Add(1)
			return nil
		}
		return a.enqueueBlocking(ctx, entry)
	}
}

func (a *asyncLogger) enqueueOrSync(ctx context.Context, entry asyncLoggerEntry) error {
	select {
	case <-a.closing:
		return fmt.Errorf("goark-log: async logger is closed")
	default:
		if a.queue.TryPublish(entry) {
			return nil
		}
		event := entry.event
		event.EndOfBatch = true
		return a.handler.dispatch(ctx, event, entry.levelAccepted)
	}
}

func (a *asyncLogger) run() {
	defer a.workers.Done()
	batch := make([]asyncLoggerEntry, 0, a.batchSize)
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

func (a *asyncLogger) drainAll(batch *[]asyncLoggerEntry) {
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

func (a *asyncLogger) flushBatch(batch []asyncLoggerEntry) {
	var joined error
	for index, entry := range batch {
		event := entry.event
		event.EndOfBatch = index == len(batch)-1
		if err := a.handler.dispatch(context.Background(), event, entry.levelAccepted); err != nil {
			joined = errors.Join(joined, err)
			a.handleAsyncError(context.Background(), err, event)
		}
	}
	if joined != nil {
		a.failed.Add(1)
	}
}

func (a *asyncLogger) handleAsyncError(ctx context.Context, err error, event Event) {
	if a == nil || a.options.ErrorHandler == nil || err == nil {
		return
	}
	a.options.ErrorHandler.HandleAsyncError(ctx, err, event)
}

func (a *asyncLogger) droppedCount() uint64 {
	if a == nil {
		return 0
	}
	return a.dropped.Load()
}

func (a *asyncLogger) failedCount() uint64 {
	if a == nil {
		return 0
	}
	return a.failed.Load()
}

func (a *asyncLogger) includeLocation() bool {
	return a != nil && a.options.IncludeLocation
}

func (a *asyncLogger) remainingCapacity() int64 {
	if a == nil || a.queue == nil {
		return 0
	}
	return a.queue.RemainingCapacity()
}
