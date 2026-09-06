package asyncappender

import (
	"context"
	"errors"

	"goark.dev/log/internal/disruptor"
)

func (a *Appender) run() {
	defer a.workers.Done()
	batch := make([]entry, 0, a.batchSize)
	for {
		if a.queue.PopBatch(&batch, cap(batch)) {
			a.flushBatch(batch)
			batch = batch[:0]
			continue
		}
		err := a.queue.WaitReadable(context.Background(), a.done)
		if errors.Is(err, disruptor.ErrInterrupted) {
			a.drain(&batch)
			a.flushBatch(batch)
			return
		}
	}
}

func (a *Appender) drain(batch *[]entry) {
	for {
		if !a.queue.PopBatch(batch, cap(*batch)) {
			return
		}
		if len(*batch) >= cap(*batch) {
			a.flushBatch(*batch)
			*batch = (*batch)[:0]
		}
	}
}

func (a *Appender) flushBatch(batch []entry) {
	var joined error
	for index, item := range batch {
		event := item.event
		event.EndOfBatch = index == len(batch)-1
		if err := a.appendSync(context.Background(), event); err != nil {
			joined = errors.Join(joined, err)
			a.handleAsyncError(context.Background(), err, event)
		}
	}
	if joined != nil {
		a.failed.Add(1)
	}
}

func (a *Appender) appendSync(ctx context.Context, event Event) error {
	var joined error
	for _, appender := range a.appenders {
		if err := appender.Append(ctx, event); err != nil {
			joined = errors.Join(joined, err)
		}
	}
	return joined
}

func (a *Appender) closeDelegates() error {
	var joined error
	for _, appender := range a.appenders {
		joined = errors.Join(joined, appender.Close())
	}
	return joined
}

func (a *Appender) handleAsyncError(ctx context.Context, err error, event Event) {
	if a == nil || a.errorHandler == nil || err == nil {
		return
	}
	a.errorHandler.HandleAsyncError(ctx, err, event)
}
