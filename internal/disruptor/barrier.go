package disruptor

import (
	"context"
	"errors"
	"sync"
)

// ErrAlerted 表示序号屏障被显式唤醒并终止等待。
var ErrAlerted = errors.New("goark-log: disruptor barrier alerted")

// SequenceBarrier 等待指定序号完成发布。
type SequenceBarrier[T any] struct {
	ring *RingBuffer[T]
	mu   sync.Mutex

	alerted bool
	alert   chan struct{}
}

// NewSequenceBarrier 创建绑定到 RingBuffer 的序号屏障。
func NewSequenceBarrier[T any](ring *RingBuffer[T]) *SequenceBarrier[T] {
	return &SequenceBarrier[T]{
		ring:  ring,
		alert: make(chan struct{}),
	}
}

// WaitFor 等待目标序号发布，并返回连续可读的最高序号。
func (b *SequenceBarrier[T]) WaitFor(ctx context.Context, sequence int64) (int64, error) {
	if b == nil || b.ring == nil {
		return initialSequence, ErrInterrupted
	}
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		if b.IsAlerted() {
			return initialSequence, ErrAlerted
		}
		if available := b.ring.HighestPublished(sequence); available >= sequence {
			return available, nil
		}
		alert := b.alertChannel()
		err := b.ring.wait.Wait(ctx, b.ring.readable, alert, func() bool {
			return b.ring.HighestPublished(sequence) >= sequence || b.IsAlerted()
		})
		if err == nil {
			continue
		}
		if errors.Is(err, ErrInterrupted) && b.IsAlerted() {
			return initialSequence, ErrAlerted
		}
		return initialSequence, err
	}
}

// Alert 唤醒所有等待者并使后续 WaitFor 返回 ErrAlerted。
func (b *SequenceBarrier[T]) Alert() {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.alerted {
		return
	}
	b.alerted = true
	close(b.alert)
}

// ClearAlert 清除告警状态，允许屏障重新等待。
func (b *SequenceBarrier[T]) ClearAlert() {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.alerted {
		return
	}
	b.alerted = false
	b.alert = make(chan struct{})
}

// IsAlerted 返回屏障是否处于告警状态。
func (b *SequenceBarrier[T]) IsAlerted() bool {
	if b == nil {
		return false
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.alerted
}

func (b *SequenceBarrier[T]) alertChannel() <-chan struct{} {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.alert
}
