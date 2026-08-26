package disruptor

import (
	"context"
	"sync/atomic"
)

const initialSequence = -1

// RingBuffer 是面向多生产者、单消费者的 Disruptor 风格有界环形队列。
type RingBuffer[T any] struct {
	capacity int64
	mask     int64
	cursor   *Sequence
	gating   *Sequence
	slots    []slot[T]
	wait     WaitStrategy
	readable chan struct{}
	writable chan struct{}
}

type slot[T any] struct {
	value     T
	published atomic.Int64
}

// NewRingBuffer 创建预分配环形队列，容量会规整为 2 的幂。
func NewRingBuffer[T any](capacity int, wait WaitStrategy) (*RingBuffer[T], error) {
	normalized, err := NormalizeCapacity(capacity)
	if err != nil {
		return nil, err
	}
	if wait == nil {
		wait = NewWaitStrategy(WaitBlocking)
	}
	ring := &RingBuffer[T]{
		capacity: int64(normalized),
		mask:     int64(normalized - 1),
		cursor:   NewSequence(initialSequence),
		gating:   NewSequence(initialSequence),
		slots:    make([]slot[T], normalized),
		wait:     wait,
		readable: make(chan struct{}, 1),
		writable: make(chan struct{}, 1),
	}
	for index := range ring.slots {
		ring.slots[index].published.Store(int64(index) - ring.capacity)
	}
	return ring, nil
}

// Capacity 返回规整后的实际容量。
func (r *RingBuffer[T]) Capacity() int {
	if r == nil {
		return 0
	}
	return int(r.capacity)
}

// TryPublish 尝试发布一个元素，满队列时立即返回 false。
func (r *RingBuffer[T]) TryPublish(value T) bool {
	if r == nil {
		return false
	}
	for {
		current := r.cursor.Load()
		next := current + 1
		wrapPoint := next - r.capacity
		if wrapPoint > r.gating.Load() {
			return false
		}
		if !r.cursor.CompareAndSwap(current, next) {
			continue
		}
		slot := &r.slots[next&r.mask]
		slot.value = value
		slot.published.Store(next)
		r.wait.Signal(r.readable)
		return true
	}
}

// PopBatch 按发布顺序批量读取已发布元素。
func (r *RingBuffer[T]) PopBatch(batch *[]T, max int) bool {
	if r == nil || batch == nil || max <= 0 {
		return false
	}
	start := r.gating.Load() + 1
	next := start
	limit := max - len(*batch)
	if limit <= 0 {
		return false
	}
	for consumed := 0; consumed < limit; consumed++ {
		slot := &r.slots[next&r.mask]
		if slot.published.Load() != next {
			break
		}
		*batch = append(*batch, slot.value)
		var zero T
		slot.value = zero
		slot.published.Store(next - r.capacity)
		next++
	}
	if next == start {
		return false
	}
	r.gating.Store(next - 1)
	r.wait.Signal(r.writable)
	return true
}

// HasCapacity 返回队列当前是否仍有可写槽位。
func (r *RingBuffer[T]) HasCapacity() bool {
	if r == nil {
		return false
	}
	next := r.cursor.Load() + 1
	return next-r.capacity <= r.gating.Load()
}

// HasReadable 返回队列当前是否存在可读元素。
func (r *RingBuffer[T]) HasReadable() bool {
	if r == nil {
		return false
	}
	next := r.gating.Load() + 1
	slot := &r.slots[next&r.mask]
	return slot.published.Load() == next
}

// WaitWritable 等待可写槽位出现。
func (r *RingBuffer[T]) WaitWritable(ctx context.Context, interrupt <-chan struct{}) error {
	if r == nil {
		return ErrInterrupted
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return r.wait.Wait(ctx, r.writable, interrupt, r.HasCapacity)
}

// WaitReadable 等待已发布元素出现。
func (r *RingBuffer[T]) WaitReadable(ctx context.Context, interrupt <-chan struct{}) error {
	if r == nil {
		return ErrInterrupted
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return r.wait.Wait(ctx, r.readable, interrupt, r.HasReadable)
}
