package goarklog

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
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
}

type asyncLogger struct {
	handler *Handler
	queue   *asyncRingBuffer
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
	dropped   atomic.Uint64
	failed    atomic.Uint64
}

type asyncLoggerEntry struct {
	event Event
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
		queue:     newAsyncRingBuffer(normalized.QueueSize),
		closing:   make(chan struct{}),
		done:      make(chan struct{}),
		options:   normalized,
		queueSize: normalized.QueueSize,
		batchSize: normalized.BatchSize,
		strategy:  normalized.OverflowStrategy,
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
	return AsyncLoggerOptions{
		Enabled:          true,
		QueueSize:        queueSize,
		BatchSize:        batchSize,
		OverflowStrategy: strategy,
	}, nil
}

func (a *asyncLogger) append(ctx context.Context, event Event) error {
	if a == nil {
		return fmt.Errorf("goark-log: async logger is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	entry := asyncLoggerEntry{event: event}
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
		if a.queue.tryPush(entry) {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-a.closing:
			return fmt.Errorf("goark-log: async logger is closed")
		case <-a.queue.writableSignal():
		}
	}
}

func (a *asyncLogger) enqueueOrDrop(entry asyncLoggerEntry) error {
	select {
	case <-a.closing:
		return fmt.Errorf("goark-log: async logger is closed")
	default:
		if a.queue.tryPush(entry) {
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
		if a.queue.tryPush(entry) {
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
		if a.queue.tryPush(entry) {
			return nil
		}
		return a.handler.dispatch(ctx, entry.event)
	}
}

func (a *asyncLogger) run() {
	defer a.workers.Done()
	batch := make([]asyncLoggerEntry, 0, a.batchSize)
	for {
		if a.queue.popBatch(&batch, a.batchSize) {
			a.flushBatch(batch)
			batch = batch[:0]
			continue
		}
		select {
		case <-a.queue.readableSignal():
		case <-a.done:
			a.drainAll(&batch)
			a.flushBatch(batch)
			return
		}
	}
}

func (a *asyncLogger) drainAll(batch *[]asyncLoggerEntry) {
	for {
		if !a.queue.popBatch(batch, a.batchSize) {
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
	for _, entry := range batch {
		if err := a.handler.dispatch(context.Background(), entry.event); err != nil {
			joined = errors.Join(joined, err)
		}
	}
	if joined != nil {
		a.failed.Add(1)
	}
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
