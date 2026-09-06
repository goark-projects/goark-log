package asyncappender

import (
	"context"
	"errors"
	"fmt"

	"goark.dev/log/internal/asyncruntime"
	"goark.dev/log/internal/disruptor"
)

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
