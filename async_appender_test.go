package goarklog

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestAsyncAppender_whenBlockStrategyAndQueueFull_shouldRespectContext(t *testing.T) {
	delegate := newGatedAppender("delegate")
	t.Cleanup(delegate.releaseGate)
	appender, err := NewAsyncAppender([]Appender{delegate},
		WithAsyncQueueSize(1),
		WithAsyncOverflowStrategy(AsyncOverflowBlock),
	)
	if err != nil {
		t.Fatalf("NewAsyncAppender() error = %v", err)
	}

	if err := appender.Append(context.Background(), testEvent("first", fixedTestTime())); err != nil {
		t.Fatalf("Append(first) error = %v", err)
	}
	<-delegate.started
	if err := appender.Append(context.Background(), testEvent("second", fixedTestTime())); err != nil {
		t.Fatalf("Append(second) error = %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err = appender.Append(ctx, testEvent("third", fixedTestTime()))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Append(third) error = %v, want context deadline", err)
	}
	delegate.releaseGate()
	if err := appender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestAsyncAppender_whenDropStrategyAndQueueFull_shouldDropNewest(t *testing.T) {
	delegate := newGatedAppender("delegate")
	t.Cleanup(delegate.releaseGate)
	appender, err := NewAsyncAppender([]Appender{delegate},
		WithAsyncQueueSize(1),
		WithAsyncOverflowStrategy(AsyncOverflowDrop),
	)
	if err != nil {
		t.Fatalf("NewAsyncAppender() error = %v", err)
	}

	if err := appender.Append(context.Background(), testEvent("first", fixedTestTime())); err != nil {
		t.Fatalf("Append(first) error = %v", err)
	}
	<-delegate.started
	if err := appender.Append(context.Background(), testEvent("second", fixedTestTime())); err != nil {
		t.Fatalf("Append(second) error = %v", err)
	}
	if err := appender.Append(context.Background(), testEvent("third", fixedTestTime())); err != nil {
		t.Fatalf("Append(third) error = %v", err)
	}
	if appender.Dropped() != 1 {
		t.Fatalf("Dropped() = %d, want 1", appender.Dropped())
	}
	delegate.releaseGate()
	if err := appender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if got := len(delegate.Events()); got != 2 {
		t.Fatalf("delegate event count = %d, want 2", got)
	}
}

func TestAsyncAppender_whenDropDebugStrategyAndQueueFull_shouldDropDebugOnly(t *testing.T) {
	delegate := newGatedAppender("delegate")
	t.Cleanup(delegate.releaseGate)
	appender, err := NewAsyncAppender([]Appender{delegate},
		WithAsyncQueueSize(1),
		WithAsyncOverflowStrategy(AsyncOverflowDropDebug),
	)
	if err != nil {
		t.Fatalf("NewAsyncAppender() error = %v", err)
	}

	if err := appender.Append(context.Background(), testEvent("first", fixedTestTime())); err != nil {
		t.Fatalf("Append(first) error = %v", err)
	}
	<-delegate.started
	if err := appender.Append(context.Background(), testEvent("second", fixedTestTime())); err != nil {
		t.Fatalf("Append(second) error = %v", err)
	}
	debugEvent := testEvent("debug", fixedTestTime())
	debugEvent.Level = slog.LevelDebug
	if err := appender.Append(context.Background(), debugEvent); err != nil {
		t.Fatalf("Append(debug) error = %v", err)
	}
	if appender.Dropped() != 1 {
		t.Fatalf("Dropped() = %d, want 1", appender.Dropped())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err = appender.Append(ctx, testEvent("info", fixedTestTime()))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Append(info) error = %v, want context deadline", err)
	}
	delegate.releaseGate()
	if err := appender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestAsyncAppender_whenSyncFallbackStrategyAndQueueFull_shouldWriteSynchronously(t *testing.T) {
	delegate := newGatedAppender("delegate")
	t.Cleanup(delegate.releaseGate)
	appender, err := NewAsyncAppender([]Appender{delegate},
		WithAsyncQueueSize(1),
		WithAsyncOverflowStrategy(AsyncOverflowSyncFallback),
	)
	if err != nil {
		t.Fatalf("NewAsyncAppender() error = %v", err)
	}

	if err := appender.Append(context.Background(), testEvent("first", fixedTestTime())); err != nil {
		t.Fatalf("Append(first) error = %v", err)
	}
	<-delegate.started
	if err := appender.Append(context.Background(), testEvent("second", fixedTestTime())); err != nil {
		t.Fatalf("Append(second) error = %v", err)
	}
	if err := appender.Append(context.Background(), testEvent("fallback", fixedTestTime())); err != nil {
		t.Fatalf("Append(fallback) error = %v", err)
	}
	if !delegate.Contains("fallback") {
		t.Fatalf("sync fallback should write immediately, events=%v", delegate.Events())
	}
	delegate.releaseGate()
	if err := appender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if got := len(delegate.Events()); got != 3 {
		t.Fatalf("delegate event count = %d, want 3", got)
	}
}

func TestAsyncAppender_whenCloseCalled_shouldDrainQueueAndCloseDelegateWhenEnabled(t *testing.T) {
	delegate := newRecordingAppender("delegate")
	appender, err := NewAsyncAppender([]Appender{delegate},
		WithAsyncQueueSize(8),
		WithAsyncWaitStrategy(AsyncWaitYield),
		WithAsyncCloseAppenders(true),
	)
	if err != nil {
		t.Fatalf("NewAsyncAppender() error = %v", err)
	}
	for index := 0; index < 5; index++ {
		if err := appender.Append(context.Background(), testEvent("event", fixedTestTime())); err != nil {
			t.Fatalf("Append(%d) error = %v", index, err)
		}
	}
	if err := appender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if got := len(delegate.Events()); got != 5 {
		t.Fatalf("delegate event count = %d, want 5", got)
	}
	events := delegate.Events()
	if !events[len(events)-1].EndOfBatch {
		t.Fatalf("last async appender event should be marked EndOfBatch, events=%+v", events)
	}
	if delegate.CloseCount() != 1 {
		t.Fatalf("delegate CloseCount() = %d, want 1", delegate.CloseCount())
	}
}

func TestNewAsyncAppender_whenBatchSizeConfigured_shouldNormalizeAgainstQueue(t *testing.T) {
	delegate := newRecordingAppender("delegate")
	appender, err := NewAsyncAppender([]Appender{delegate},
		WithAsyncQueueSize(3),
		WithAsyncBatchSize(8),
	)
	if err != nil {
		t.Fatalf("NewAsyncAppender() error = %v", err)
	}
	if appender.queueSize != 4 {
		t.Fatalf("queueSize = %d, want normalized size 4", appender.queueSize)
	}
	if appender.batchSize != 4 {
		t.Fatalf("batchSize = %d, want capped queue size 4", appender.batchSize)
	}
	if err := appender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestNewAsyncAppender_whenBatchSizeInvalid_shouldReject(t *testing.T) {
	_, err := NewAsyncAppender([]Appender{newRecordingAppender("delegate")},
		WithAsyncBatchSize(0),
	)
	if err == nil || !strings.Contains(err.Error(), "batch size") {
		t.Fatalf("NewAsyncAppender() error = %v, want batch size rejection", err)
	}
}

func TestNewAsyncAppender_whenWaitStrategyInvalid_shouldReject(t *testing.T) {
	_, err := NewAsyncAppender([]Appender{newRecordingAppender("delegate")},
		WithAsyncWaitStrategy("park-forever"),
	)
	if err == nil || !strings.Contains(err.Error(), "wait strategy") {
		t.Fatalf("NewAsyncAppender() error = %v, want wait strategy rejection", err)
	}
}

func TestAsyncAppender_whenErrorHandlerConfigured_shouldReceiveWriteFailure(t *testing.T) {
	errCh := make(chan error, 1)
	appender, err := NewAsyncAppender([]Appender{failingAppender{name: "broken"}},
		WithAsyncQueueSize(4),
		WithAsyncErrorHandler(AsyncErrorHandlerFunc(func(_ context.Context, err error, event Event) {
			if event.Message == "broken event" {
				errCh <- err
			}
		})),
	)
	if err != nil {
		t.Fatalf("NewAsyncAppender() error = %v", err)
	}
	if err := appender.Append(context.Background(), testEvent("broken event", fixedTestTime())); err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	if err := appender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	select {
	case err := <-errCh:
		if err == nil || !strings.Contains(err.Error(), "forced failure") {
			t.Fatalf("async error = %v, want forced failure", err)
		}
	default:
		t.Fatalf("async error handler was not called")
	}
	if appender.Failed() != 1 {
		t.Fatalf("Failed() = %d, want 1", appender.Failed())
	}
}

func TestAsyncAppender_whenCloseWhileAppendBlocked_shouldUnblockAppend(t *testing.T) {
	delegate := newGatedAppender("delegate")
	t.Cleanup(delegate.releaseGate)
	appender, err := NewAsyncAppender([]Appender{delegate},
		WithAsyncQueueSize(1),
		WithAsyncOverflowStrategy(AsyncOverflowBlock),
	)
	if err != nil {
		t.Fatalf("NewAsyncAppender() error = %v", err)
	}

	if err := appender.Append(context.Background(), testEvent("first", fixedTestTime())); err != nil {
		t.Fatalf("Append(first) error = %v", err)
	}
	<-delegate.started
	if err := appender.Append(context.Background(), testEvent("second", fixedTestTime())); err != nil {
		t.Fatalf("Append(second) error = %v", err)
	}

	appendDone := make(chan error, 1)
	go func() {
		appendDone <- appender.Append(context.Background(), testEvent("blocked", fixedTestTime()))
	}()
	closeDone := make(chan error, 1)
	go func() {
		closeDone <- appender.Close()
	}()

	select {
	case err := <-appendDone:
		if err == nil || !strings.Contains(err.Error(), "closed") {
			t.Fatalf("blocked Append() error = %v, want closed error", err)
		}
	case <-time.After(time.Second):
		t.Fatalf("blocked Append() was not unblocked by Close()")
	}

	delegate.releaseGate()
	select {
	case err := <-closeDone:
		if err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatalf("Close() did not finish after delegate was released")
	}
}

type recordingAppender struct {
	name       string
	mu         sync.Mutex
	events     []Event
	closeCount int
}

func newRecordingAppender(name string) *recordingAppender {
	return &recordingAppender{name: name}
}

func (a *recordingAppender) Name() string {
	return a.name
}

func (a *recordingAppender) Append(ctx context.Context, event Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.events = append(a.events, event)
	return nil
}

func (a *recordingAppender) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.closeCount++
	return nil
}

func (a *recordingAppender) Events() []Event {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]Event(nil), a.events...)
}

func (a *recordingAppender) Contains(message string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, event := range a.events {
		if event.Message == message {
			return true
		}
	}
	return false
}

func (a *recordingAppender) CloseCount() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.closeCount
}

type gatedAppender struct {
	*recordingAppender
	started     chan struct{}
	release     chan struct{}
	blocked     atomic.Bool
	releaseOnce sync.Once
}

func newGatedAppender(name string) *gatedAppender {
	return &gatedAppender{
		recordingAppender: newRecordingAppender(name),
		started:           make(chan struct{}),
		release:           make(chan struct{}),
	}
}

func (a *gatedAppender) Append(ctx context.Context, event Event) error {
	if a.blocked.CompareAndSwap(false, true) {
		close(a.started)
		select {
		case <-a.release:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return a.recordingAppender.Append(ctx, event)
}

func (a *gatedAppender) releaseGate() {
	a.releaseOnce.Do(func() {
		close(a.release)
	})
}
