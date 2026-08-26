package disruptor

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestRingBuffer_whenCapacityNotPowerOfTwo_shouldRoundUp(t *testing.T) {
	ring, err := NewRingBuffer[int](3, nil)
	if err != nil {
		t.Fatalf("NewRingBuffer() error = %v", err)
	}
	if ring.Capacity() != 4 {
		t.Fatalf("Capacity() = %d, want 4", ring.Capacity())
	}
}

func TestRingBuffer_whenFull_shouldReportBackpressureUntilConsumed(t *testing.T) {
	ring, err := NewRingBuffer[int](2, nil)
	if err != nil {
		t.Fatalf("NewRingBuffer() error = %v", err)
	}
	if !ring.TryPublish(1) || !ring.TryPublish(2) {
		t.Fatalf("first two publishes should fit")
	}
	if ring.TryPublish(3) {
		t.Fatalf("third publish should see full ring")
	}
	batch := make([]int, 0, 2)
	if !ring.PopBatch(&batch, 1) {
		t.Fatalf("PopBatch() should read one element")
	}
	if !ring.TryPublish(3) {
		t.Fatalf("publish after consume should fit")
	}
}

func TestRingBuffer_whenPublishedConcurrently_shouldConsumeEveryValue(t *testing.T) {
	const producers = 8
	const perProducer = 256
	total := producers * perProducer
	ring, err := NewRingBuffer[int](64, NewWaitStrategy(WaitYielding))
	if err != nil {
		t.Fatalf("NewRingBuffer() error = %v", err)
	}
	done := make(chan struct{})
	var wait sync.WaitGroup
	for producer := 0; producer < producers; producer++ {
		producer := producer
		wait.Add(1)
		go func() {
			defer wait.Done()
			for index := 0; index < perProducer; index++ {
				value := producer*perProducer + index
				for !ring.TryPublish(value) {
					if err := ring.WaitWritable(context.Background(), done); err != nil {
						t.Errorf("WaitWritable() error = %v", err)
						return
					}
				}
			}
		}()
	}

	seen := make([]bool, total)
	batch := make([]int, 0, 32)
	for consumed := 0; consumed < total; {
		if !ring.PopBatch(&batch, 32) {
			if err := ring.WaitReadable(context.Background(), done); err != nil {
				t.Fatalf("WaitReadable() error = %v", err)
			}
			continue
		}
		for _, value := range batch {
			if value < 0 || value >= total {
				t.Fatalf("consumed value %d out of range", value)
			}
			if seen[value] {
				t.Fatalf("consumed duplicate value %d", value)
			}
			seen[value] = true
			consumed++
		}
		batch = batch[:0]
	}
	wait.Wait()
	close(done)
	for value, ok := range seen {
		if !ok {
			t.Fatalf("value %d was not consumed", value)
		}
	}
}

func TestRingBuffer_whenWaitingInterrupted_shouldReturnInterrupted(t *testing.T) {
	ring, err := NewRingBuffer[int](1, nil)
	if err != nil {
		t.Fatalf("NewRingBuffer() error = %v", err)
	}
	interrupt := make(chan struct{})
	close(interrupt)
	err = ring.WaitReadable(context.Background(), interrupt)
	if !errors.Is(err, ErrInterrupted) {
		t.Fatalf("WaitReadable() error = %v, want interrupted", err)
	}
}

func TestRingBuffer_whenWaitingWithContextDeadline_shouldReturnDeadline(t *testing.T) {
	ring, err := NewRingBuffer[int](1, nil)
	if err != nil {
		t.Fatalf("NewRingBuffer() error = %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	err = ring.WaitReadable(ctx, make(chan struct{}))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("WaitReadable() error = %v, want deadline", err)
	}
}
