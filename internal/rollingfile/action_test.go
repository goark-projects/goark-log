package rollingfile

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRollingFileAppender_whenAsyncActionQueueFull_shouldKeepActionsSerialized(t *testing.T) {
	appender := &RollingFileAppender{
		asyncActions:    true,
		actionQueueSize: 1,
	}
	appender.startActionWorker()
	defer func() {
		if err := appender.closeActionWorker(); err != nil {
			t.Fatalf("closeActionWorker() error = %v", err)
		}
	}()

	var running atomic.Int32
	var maxRunning atomic.Int32
	track := func(action func()) func() error {
		return func() error {
			current := running.Add(1)
			for {
				observed := maxRunning.Load()
				if current <= observed || maxRunning.CompareAndSwap(observed, current) {
					break
				}
			}
			defer running.Add(-1)
			action()
			return nil
		}
	}

	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	var releaseOnce sync.Once
	release := func() {
		releaseOnce.Do(func() {
			close(releaseFirst)
		})
	}
	defer release()
	if err := appender.enqueueRolloverAction(track(func() {
		close(firstStarted)
		<-releaseFirst
	})); err != nil {
		t.Fatalf("enqueue first action error = %v", err)
	}
	<-firstStarted

	if err := appender.enqueueRolloverAction(track(func() {})); err != nil {
		t.Fatalf("enqueue buffered action error = %v", err)
	}

	secondDone := make(chan struct{})
	enqueueErr := make(chan error, 1)
	go func() {
		enqueueErr <- appender.enqueueRolloverAction(track(func() {
			close(secondDone)
		}))
	}()

	select {
	case <-secondDone:
		release()
		t.Fatalf("async rolling action ran while another action was still active")
	case <-time.After(100 * time.Millisecond):
	}

	release()
	if err := <-enqueueErr; err != nil {
		t.Fatalf("enqueue blocked action error = %v", err)
	}
	select {
	case <-secondDone:
	case <-time.After(time.Second):
		t.Fatalf("blocked action was not drained")
	}
	if maxRunning.Load() != 1 {
		t.Fatalf("rolling actions should be serialized, max running = %d", maxRunning.Load())
	}
}
