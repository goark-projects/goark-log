package disruptor

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestSequenceBarrier_whenSequencePublished_shouldReturnHighestContiguous(t *testing.T) {
	ring, err := NewRingBuffer[int](8, NewWaitStrategy(WaitYielding))
	if err != nil {
		t.Fatalf("NewRingBuffer() error = %v", err)
	}
	barrier := NewSequenceBarrier(ring)
	for value := 0; value < 3; value++ {
		if !ring.TryPublish(value) {
			t.Fatalf("TryPublish(%d) = false", value)
		}
	}

	available, err := barrier.WaitFor(context.Background(), 1)
	if err != nil {
		t.Fatalf("WaitFor() error = %v", err)
	}
	if available != 2 {
		t.Fatalf("available = %d, want 2", available)
	}
}

func TestSequenceBarrier_whenAlerted_shouldInterruptWait(t *testing.T) {
	ring, err := NewRingBuffer[int](2, NewWaitStrategy(WaitBlocking))
	if err != nil {
		t.Fatalf("NewRingBuffer() error = %v", err)
	}
	barrier := NewSequenceBarrier(ring)
	done := make(chan error, 1)
	go func() {
		_, err := barrier.WaitFor(context.Background(), 0)
		done <- err
	}()
	barrier.Alert()

	select {
	case err := <-done:
		if !errors.Is(err, ErrAlerted) {
			t.Fatalf("WaitFor() error = %v, want ErrAlerted", err)
		}
	case <-time.After(time.Second):
		t.Fatalf("WaitFor() was not interrupted by Alert()")
	}
	barrier.ClearAlert()
	if barrier.IsAlerted() {
		t.Fatalf("IsAlerted() = true, want false")
	}
}

func TestBatchEventProcessor_whenHalted_shouldDrainQueuedEvents(t *testing.T) {
	ring, err := NewRingBuffer[int](8, NewWaitStrategy(WaitBlocking))
	if err != nil {
		t.Fatalf("NewRingBuffer() error = %v", err)
	}
	var (
		mu      sync.Mutex
		values  []int
		seqs    []int64
		endings []bool
	)
	processor, err := NewBatchEventProcessor(ring, EventHandlerFunc[int](func(_ context.Context, event int, sequence int64, endOfBatch bool) error {
		mu.Lock()
		defer mu.Unlock()
		values = append(values, event)
		seqs = append(seqs, sequence)
		endings = append(endings, endOfBatch)
		return nil
	}), WithBatchSize[int](3))
	if err != nil {
		t.Fatalf("NewBatchEventProcessor() error = %v", err)
	}
	done := make(chan error, 1)
	go func() {
		done <- processor.Run(context.Background())
	}()
	for value := 10; value < 15; value++ {
		if !ring.TryPublish(value) {
			t.Fatalf("TryPublish(%d) = false", value)
		}
	}
	processor.Halt()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatalf("Run() did not stop after Halt()")
	}

	mu.Lock()
	defer mu.Unlock()
	if !reflect.DeepEqual(values, []int{10, 11, 12, 13, 14}) {
		t.Fatalf("values = %v", values)
	}
	if !reflect.DeepEqual(seqs, []int64{0, 1, 2, 3, 4}) {
		t.Fatalf("sequences = %v", seqs)
	}
	if !reflect.DeepEqual(endings, []bool{false, false, true, false, true}) {
		t.Fatalf("endOfBatch flags = %v", endings)
	}
	if processor.Sequence() != 4 {
		t.Fatalf("processor sequence = %d, want 4", processor.Sequence())
	}
}

func TestBatchEventProcessor_whenHandlerFails_shouldCallExceptionHandler(t *testing.T) {
	ring, err := NewRingBuffer[int](2, NewWaitStrategy(WaitBlocking))
	if err != nil {
		t.Fatalf("NewRingBuffer() error = %v", err)
	}
	wantErr := errors.New("boom")
	called := make(chan int64, 1)
	processor, err := NewBatchEventProcessor(ring,
		EventHandlerFunc[int](func(context.Context, int, int64, bool) error {
			return wantErr
		}),
		WithExceptionHandler[int](ExceptionHandlerFunc[int](func(_ context.Context, err error, sequence int64, event int) {
			if errors.Is(err, wantErr) && event == 7 {
				called <- sequence
			}
		})),
	)
	if err != nil {
		t.Fatalf("NewBatchEventProcessor() error = %v", err)
	}
	done := make(chan error, 1)
	go func() {
		done <- processor.Run(context.Background())
	}()
	if !ring.TryPublish(7) {
		t.Fatalf("TryPublish() = false")
	}

	select {
	case sequence := <-called:
		if sequence != 0 {
			t.Fatalf("exception sequence = %d, want 0", sequence)
		}
	case <-time.After(time.Second):
		t.Fatalf("exception handler was not called")
	}
	processor.Halt()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatalf("Run() did not stop")
	}
}
