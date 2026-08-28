package asyncappender

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

// Option 调整 Appender。
type Option func(*Appender)

// WithName 设置 appender 名称。
func WithName(name string) Option {
	return func(appender *Appender) {
		appender.name = name
	}
}

// WithQueueSize 设置异步队列长度。
func WithQueueSize(size int) Option {
	return func(appender *Appender) {
		appender.queueSize = size
	}
}

// WithBatchSize 设置后台协程单次批量写出上限。
func WithBatchSize(size int) Option {
	return func(appender *Appender) {
		appender.batchSize = size
	}
}

// WithOverflowStrategy 设置队列满时的处理策略。
func WithOverflowStrategy(strategy asyncruntime.OverflowStrategy) Option {
	return func(appender *Appender) {
		appender.strategy = strategy
	}
}

// WithWaitStrategy 设置异步队列等待策略。
func WithWaitStrategy(strategy asyncruntime.WaitStrategy) Option {
	return func(appender *Appender) {
		appender.waitStrategy = strategy
	}
}

// WithWaitOptions 设置异步队列等待策略参数。
func WithWaitOptions(options asyncruntime.WaitOptions) Option {
	return func(appender *Appender) {
		appender.waitOptions = options
	}
}

// WithErrorHandler 设置异步后台写入失败处理器。
func WithErrorHandler(handler asyncruntime.ErrorHandler) Option {
	return func(appender *Appender) {
		appender.errorHandler = handler
	}
}

// WithCloseAppenders 设置关闭 async 时是否同时关闭下游 appender。
func WithCloseAppenders(enabled bool) Option {
	return func(appender *Appender) {
		appender.closeAppenders = enabled
	}
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
	normalizedQueueSize, err := asyncruntime.NormalizeQueueSize(appender.queueSize, asyncruntime.DefaultAsyncQueueSize)
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
	appender.queue, err = disruptor.NewRingBuffer[entry](appender.queueSize, asyncruntime.NewWaitStrategyWithOptions(appender.waitStrategy, appender.waitOptions))
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

func (a *Appender) beginAppend() bool {
	a.stateMu.RLock()
	defer a.stateMu.RUnlock()
	if a.closed {
		return false
	}
	// producer 计数必须在状态锁内增加，避免 Close 与新的 Add 并发。
	a.producers.Add(1)
	return true
}

func (a *Appender) validate(appenders []Sink) error {
	if strings.TrimSpace(a.name) == "" {
		return fmt.Errorf("goark-log: async appender name is empty")
	}
	if a.queueSize <= 0 {
		return fmt.Errorf("goark-log: async queue size must be > 0")
	}
	if a.batchSize <= 0 {
		return fmt.Errorf("goark-log: async appender batch size must be > 0")
	}
	if _, err := asyncruntime.ParseOverflowStrategy(string(a.strategy)); err != nil {
		return err
	}
	if err := asyncruntime.ValidateWaitOptions(a.waitOptions); err != nil {
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

func (a *Appender) enqueueBlocking(ctx context.Context, item entry) error {
	for {
		if a.queue.TryPublish(item) {
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

func (a *Appender) enqueueOrDrop(item entry) error {
	select {
	case <-a.closing:
		return fmt.Errorf("goark-log: async appender %q is closed", a.Name())
	default:
		if a.queue.TryPublish(item) {
			return nil
		}
		a.dropped.Add(1)
	}
	return nil
}

func (a *Appender) enqueueDropDebug(ctx context.Context, item entry) error {
	select {
	case <-a.closing:
		return fmt.Errorf("goark-log: async appender %q is closed", a.Name())
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

func (a *Appender) enqueueOrSync(ctx context.Context, item entry) error {
	select {
	case <-a.closing:
		return fmt.Errorf("goark-log: async appender %q is closed", a.Name())
	default:
		if a.queue.TryPublish(item) {
			return nil
		}
		event := item.event
		event.EndOfBatch = true
		return a.appendSync(ctx, event)
	}
}

func (a *Appender) run() {
	defer a.workers.Done()
	batch := make([]entry, 0, a.batchSize)
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

func (a *Appender) drain(batch *[]entry) {
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

func (a *Appender) flushBatch(batch []entry) {
	var joined error
	for index, item := range batch {
		event := item.event
		event.EndOfBatch = index == len(batch)-1
		if err := a.appendSync(context.Background(), event); err != nil {
			joined = errors.Join(joined, err)
			a.handleAsyncError(context.Background(), err, event)
		}
	}
	if joined != nil {
		a.failed.Add(1)
	}
}

func (a *Appender) appendSync(ctx context.Context, event Event) error {
	var joined error
	for _, appender := range a.appenders {
		if err := appender.Append(ctx, event); err != nil {
			joined = errors.Join(joined, err)
		}
	}
	return joined
}

func (a *Appender) closeDelegates() error {
	var joined error
	for _, appender := range a.appenders {
		joined = errors.Join(joined, appender.Close())
	}
	return joined
}

func (a *Appender) handleAsyncError(ctx context.Context, err error, event Event) {
	if a == nil || a.errorHandler == nil || err == nil {
		return
	}
	a.errorHandler.HandleAsyncError(ctx, err, event)
}
