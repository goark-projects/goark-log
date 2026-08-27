package goarklog

import (
	"context"
	"errors"
	"fmt"
	"goark.dev/log/internal/disruptor"
	"log/slog"
)

func (a *AsyncAppender) enqueueBlocking(ctx context.Context, entry asyncEntry) error {
	for {
		if a.queue.TryPublish(entry) {
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

func (a *AsyncAppender) enqueueOrDrop(entry asyncEntry) error {
	select {
	case <-a.closing:
		return fmt.Errorf("goark-log: async appender %q is closed", a.Name())
	default:
		if a.queue.TryPublish(entry) {
			return nil
		}
		a.dropped.Add(1)
	}
	return nil
}

func (a *AsyncAppender) enqueueDropDebug(ctx context.Context, entry asyncEntry) error {
	select {
	case <-a.closing:
		return fmt.Errorf("goark-log: async appender %q is closed", a.Name())
	default:
		if a.queue.TryPublish(entry) {
			return nil
		}
		if entry.event.Level <= slog.LevelDebug {
			a.dropped.Add(1)
			return nil
		}
		return a.enqueueBlocking(ctx, entry)
	}
}

func (a *AsyncAppender) enqueueOrSync(ctx context.Context, entry asyncEntry) error {
	select {
	case <-a.closing:
		return fmt.Errorf("goark-log: async appender %q is closed", a.Name())
	default:
		if a.queue.TryPublish(entry) {
			return nil
		}
		event := entry.event
		event.EndOfBatch = true
		return a.appendSync(ctx, event)
	}
}
