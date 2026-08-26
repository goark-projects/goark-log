package goarklog

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"

	"goark.dev/log/internal/disruptor"
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
	case "", AsyncOverflowBlock:
		return AsyncOverflowBlock, nil
	case AsyncOverflowDrop:
		return AsyncOverflowDrop, nil
	case AsyncOverflowDropDebug:
		return AsyncOverflowDropDebug, nil
	case AsyncOverflowSyncFallback:
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
	waitStrategy   AsyncWaitStrategy
	strategy       AsyncOverflowStrategy
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

// AsyncOption 调整 AsyncAppender。
type AsyncOption func(*AsyncAppender)

// WithAsyncName 设置 appender 名称。
func WithAsyncName(name string) AsyncOption {
	return func(appender *AsyncAppender) {
		appender.name = name
	}
}

// WithAsyncQueueSize 设置异步队列长度。
func WithAsyncQueueSize(size int) AsyncOption {
	return func(appender *AsyncAppender) {
		appender.queueSize = size
	}
}

// WithAsyncOverflowStrategy 设置队列满时的处理策略。
func WithAsyncOverflowStrategy(strategy AsyncOverflowStrategy) AsyncOption {
	return func(appender *AsyncAppender) {
		appender.strategy = strategy
	}
}

// WithAsyncWaitStrategy 设置异步队列等待策略。
func WithAsyncWaitStrategy(strategy AsyncWaitStrategy) AsyncOption {
	return func(appender *AsyncAppender) {
		appender.waitStrategy = strategy
	}
}

// WithAsyncCloseAppenders 设置关闭 async 时是否同时关闭下游 appender。
func WithAsyncCloseAppenders(enabled bool) AsyncOption {
	return func(appender *AsyncAppender) {
		appender.closeAppenders = enabled
	}
}

// NewAsyncAppender 创建异步 appender。
func NewAsyncAppender(appenders []Appender, options ...AsyncOption) (*AsyncAppender, error) {
	appender := &AsyncAppender{
		name:         "async",
		queueSize:    DefaultAsyncQueueSize,
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
	appender.waitStrategy = waitStrategy
	appender.appenders = append([]Appender(nil), appenders...)
	appender.queue, err = disruptor.NewRingBuffer[asyncEntry](appender.queueSize, newAsyncWaitStrategy(appender.waitStrategy))
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
	if _, err := ParseAsyncOverflowStrategy(string(a.strategy)); err != nil {
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

func (a *AsyncAppender) enqueueBlocking(ctx context.Context, entry asyncEntry) error {
	for {
		if a.queue.TryPublish(entry) {
			return nil
		}
		err := a.queue.WaitWritable(ctx, a.closing)
		if errors.Is(err, disruptor.ErrInterrupted) {
			return fmt.Errorf("goark-log: async appender %q is closed", a.Name())
		}
		if err != nil {
			return err
		}
	}
}

func (a *AsyncAppender) enqueueOrDrop(entry asyncEntry) error {
	select {
	case <-a.closing:
		return fmt.Errorf("goark-log: async appender %q is closed", a.Name())
	default:
		if a.queue.TryPublish(entry) {
			return nil
		}
		a.dropped.Add(1)
	}
	return nil
}

func (a *AsyncAppender) enqueueDropDebug(ctx context.Context, entry asyncEntry) error {
	select {
	case <-a.closing:
		return fmt.Errorf("goark-log: async appender %q is closed", a.Name())
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

func (a *AsyncAppender) enqueueOrSync(ctx context.Context, entry asyncEntry) error {
	select {
	case <-a.closing:
		return fmt.Errorf("goark-log: async appender %q is closed", a.Name())
	default:
		if a.queue.TryPublish(entry) {
			return nil
		}
		event := entry.event
		event.EndOfBatch = true
		return a.appendSync(ctx, event)
	}
}

func (a *AsyncAppender) run() {
	defer a.workers.Done()
	batch := make([]asyncEntry, 0, min(a.queueSize, DefaultAsyncAppenderBatchSize))
	for {
		if a.queue.PopBatch(&batch, cap(batch)) {
			a.flushBatch(batch)
			batch = batch[:0]
			continue
		}
		err := a.queue.WaitReadable(context.Background(), a.done)
		if errors.Is(err, disruptor.ErrInterrupted) {
			a.drain(&batch)
			a.flushBatch(batch)
			return
		}
	}
}

func (a *AsyncAppender) drain(batch *[]asyncEntry) {
	for {
		if !a.queue.PopBatch(batch, cap(*batch)) {
			return
		}
		if len(*batch) >= cap(*batch) {
			a.flushBatch(*batch)
			*batch = (*batch)[:0]
		}
	}
}

func (a *AsyncAppender) flushBatch(batch []asyncEntry) {
	var joined error
	for index, entry := range batch {
		event := entry.event
		event.EndOfBatch = index == len(batch)-1
		if err := a.appendSync(context.Background(), event); err != nil {
			joined = errors.Join(joined, err)
		}
	}
	if joined != nil {
		a.failed.Add(1)
	}
}

func (a *AsyncAppender) appendSync(ctx context.Context, event Event) error {
	var joined error
	for _, appender := range a.appenders {
		if err := appender.Append(ctx, event); err != nil {
			joined = errors.Join(joined, err)
		}
	}
	return joined
}

func (a *AsyncAppender) closeDelegates() error {
	var joined error
	for _, appender := range a.appenders {
		joined = errors.Join(joined, appender.Close())
	}
	return joined
}
