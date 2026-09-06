package integration

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	. "goark.dev/log/internal/testsupport"
)

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

	if err := appender.Append(context.Background(), TestEvent("first", FixedTestTime())); err != nil {
		t.Fatalf("Append(first) error = %v", err)
	}
	<-delegate.started
	if err := appender.Append(context.Background(), TestEvent("second", FixedTestTime())); err != nil {
		t.Fatalf("Append(second) error = %v", err)
	}

	appendDone := make(chan error, 1)
	go func() {
		appendDone <- appender.Append(context.Background(), TestEvent("blocked", FixedTestTime()))
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

	if err := appender.Append(context.Background(), TestEvent("first", FixedTestTime())); err != nil {
		t.Fatalf("Append(first) error = %v", err)
	}
	<-delegate.started
	if err := appender.Append(context.Background(), TestEvent("second", FixedTestTime())); err != nil {
		t.Fatalf("Append(second) error = %v", err)
	}
	if err := appender.Append(
		context.Background(),
		TestEvent("fallback", FixedTestTime()),
	); err != nil {
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

	if err := appender.Append(context.Background(), TestEvent("first", FixedTestTime())); err != nil {
		t.Fatalf("Append(first) error = %v", err)
	}
	<-delegate.started
	if err := appender.Append(context.Background(), TestEvent("second", FixedTestTime())); err != nil {
		t.Fatalf("Append(second) error = %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err = appender.Append(ctx, TestEvent("third", FixedTestTime()))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Append(third) error = %v, want context deadline", err)
	}
	delegate.releaseGate()
	if err := appender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestNewAsyncAppender_whenBatchSizeConfigured_shouldNormalizeAgainstQueue(t *testing.T) {
	delegate := NewRecordingAppender("delegate")
	appender, err := NewAsyncAppender([]Appender{delegate},
		WithAsyncQueueSize(3),
		WithAsyncBatchSize(8),
	)
	if err != nil {
		t.Fatalf("NewAsyncAppender() error = %v", err)
	}
	if appender.QueueSize() != 4 {
		t.Fatalf("queueSize = %d, want normalized size 4", appender.QueueSize())
	}
	if appender.BatchSize() != 4 {
		t.Fatalf("batchSize = %d, want capped queue size 4", appender.BatchSize())
	}
	if err := appender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestAsyncAppender_whenOverflowAliasConfigured_shouldNormalizeStrategy(t *testing.T) {
	delegate := NewRecordingAppender("delegate")
	appender, err := NewAsyncAppender([]Appender{delegate},
		WithAsyncOverflowStrategy("discard-newest"),
	)
	if err != nil {
		t.Fatalf("NewAsyncAppender() error = %v", err)
	}
	if err := appender.Append(
		context.Background(),
		TestEvent("aliased strategy", FixedTestTime()),
	); err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	if err := appender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if !delegate.Contains("aliased strategy") {
		t.Fatalf("delegate events = %+v, want aliased strategy event", delegate.Events())
	}
}

func TestNewAsyncAppender_whenWaitStrategyInvalid_shouldReject(t *testing.T) {
	_, err := NewAsyncAppender([]Appender{NewRecordingAppender("delegate")},
		WithAsyncWaitStrategy("park-forever"),
	)
	if err == nil || !strings.Contains(err.Error(), "wait strategy") {
		t.Fatalf("NewAsyncAppender() error = %v, want wait strategy rejection", err)
	}
}

func newGatedAppender(name string) *gatedAppender {
	return &gatedAppender{
		RecordingAppender: NewRecordingAppender(name),
		started:           make(chan struct{}),
		release:           make(chan struct{}),
	}
}
