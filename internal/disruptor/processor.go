package disruptor

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

const defaultProcessorBatchSize = 64

// EventHandler 处理 RingBuffer 中的事件。
type EventHandler[T any] interface {
	OnEvent(ctx context.Context, event T, sequence int64, endOfBatch bool) error
}

// EventHandlerFunc 把函数适配为 EventHandler。
type EventHandlerFunc[T any] func(ctx context.Context, event T, sequence int64, endOfBatch bool) error

// OnEvent 执行事件处理函数。
func (f EventHandlerFunc[T]) OnEvent(ctx context.Context, event T, sequence int64, endOfBatch bool) error {
	if f == nil {
		return nil
	}
	return f(ctx, event, sequence, endOfBatch)
}

// ExceptionHandler 处理事件消费失败。
type ExceptionHandler[T any] interface {
	HandleEventException(ctx context.Context, err error, sequence int64, event T)
}

// ExceptionHandlerFunc 把函数适配为 ExceptionHandler。
type ExceptionHandlerFunc[T any] func(ctx context.Context, err error, sequence int64, event T)

// HandleEventException 执行异常处理函数。
func (f ExceptionHandlerFunc[T]) HandleEventException(ctx context.Context, err error, sequence int64, event T) {
	if f != nil {
		f(ctx, err, sequence, event)
	}
}

// BatchEventProcessorOption 调整 BatchEventProcessor。
type BatchEventProcessorOption[T any] func(*BatchEventProcessor[T])

// WithBatchSize 设置单次批量消费上限。
func WithBatchSize[T any](size int) BatchEventProcessorOption[T] {
	return func(processor *BatchEventProcessor[T]) {
		processor.batchSize = size
	}
}

// WithExceptionHandler 设置事件处理异常回调。
func WithExceptionHandler[T any](handler ExceptionHandler[T]) BatchEventProcessorOption[T] {
	return func(processor *BatchEventProcessor[T]) {
		processor.exceptionHandler = handler
	}
}

// BatchEventProcessor 按批次从 RingBuffer 读取并处理事件。
type BatchEventProcessor[T any] struct {
	ring             *RingBuffer[T]
	handler          EventHandler[T]
	exceptionHandler ExceptionHandler[T]
	batchSize        int
	sequence         *Sequence

	haltOnce sync.Once
	halted   chan struct{}
	running  atomic.Bool
}

// NewBatchEventProcessor 创建批量事件处理器。
func NewBatchEventProcessor[T any](ring *RingBuffer[T], handler EventHandler[T], options ...BatchEventProcessorOption[T]) (*BatchEventProcessor[T], error) {
	if ring == nil {
		return nil, fmt.Errorf("goark-log: disruptor processor ring buffer is nil")
	}
	if handler == nil {
		return nil, fmt.Errorf("goark-log: disruptor processor handler is nil")
	}
	processor := &BatchEventProcessor[T]{
		ring:      ring,
		handler:   handler,
		batchSize: defaultProcessorBatchSize,
		sequence:  NewSequence(initialSequence),
		halted:    make(chan struct{}),
	}
	for _, option := range options {
		if option != nil {
			option(processor)
		}
	}
	if processor.batchSize <= 0 {
		return nil, fmt.Errorf("goark-log: disruptor processor batch size must be > 0")
	}
	return processor, nil
}

// Run 在当前 goroutine 中运行事件处理循环。
func (p *BatchEventProcessor[T]) Run(ctx context.Context) error {
	if p == nil {
		return fmt.Errorf("goark-log: disruptor processor is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if !p.running.CompareAndSwap(false, true) {
		return fmt.Errorf("goark-log: disruptor processor is already running")
	}
	defer p.running.Store(false)

	batch := make([]BatchEvent[T], 0, p.batchSize)
	for {
		if p.ring.PopBatchEvents(&batch, p.batchSize) {
			p.handleBatch(ctx, batch)
			batch = batch[:0]
			continue
		}
		err := p.ring.WaitReadable(ctx, p.halted)
		if errors.Is(err, ErrInterrupted) {
			p.drain(ctx, &batch)
			return nil
		}
		if err != nil {
			return err
		}
	}
}

// Halt 请求处理器停止，Run 会先排空当前队列再返回。
func (p *BatchEventProcessor[T]) Halt() {
	if p == nil {
		return
	}
	p.haltOnce.Do(func() {
		close(p.halted)
	})
}

// Sequence 返回最后成功尝试处理的序号。
func (p *BatchEventProcessor[T]) Sequence() int64 {
	if p == nil || p.sequence == nil {
		return initialSequence
	}
	return p.sequence.Load()
}

func (p *BatchEventProcessor[T]) drain(ctx context.Context, batch *[]BatchEvent[T]) {
	for {
		if !p.ring.PopBatchEvents(batch, p.batchSize) {
			return
		}
		p.handleBatch(ctx, *batch)
		*batch = (*batch)[:0]
	}
}

func (p *BatchEventProcessor[T]) handleBatch(ctx context.Context, batch []BatchEvent[T]) {
	for index, entry := range batch {
		endOfBatch := index == len(batch)-1
		err := p.handler.OnEvent(ctx, entry.Value, entry.Sequence, endOfBatch)
		p.sequence.Store(entry.Sequence)
		if err != nil && p.exceptionHandler != nil {
			p.exceptionHandler.HandleEventException(ctx, err, entry.Sequence, entry.Value)
		}
	}
}
