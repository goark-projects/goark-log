package corebench

import (
	"context"
	"sync"
	"sync/atomic"

	"goark.dev/log"
)

const (
	AsyncOverflowBlock        = log.AsyncOverflowBlock
	AsyncOverflowDrop         = log.AsyncOverflowDrop
	AsyncOverflowDropDebug    = log.AsyncOverflowDropDebug
	AsyncOverflowSyncFallback = log.AsyncOverflowSyncFallback

	AsyncWaitBlock = log.AsyncWaitBlock
	AsyncWaitYield = log.AsyncWaitYield
)

type (
	Appender              = log.Appender
	AsyncLoggerOptions    = log.AsyncLoggerOptions
	AsyncOverflowStrategy = log.AsyncOverflowStrategy
	AsyncWaitOptions      = log.AsyncWaitOptions
	AsyncWaitStrategy     = log.AsyncWaitStrategy
	Event                 = log.Event
	JSONLayout            = log.JSONLayout
	Layout                = log.Layout
	Logger                = log.Logger
	Options               = log.Options
	RootLogger            = log.RootLogger
	TextLayout            = log.TextLayout
)

var (
	NewAsyncAppender             = log.NewAsyncAppender
	NewConsoleAppender           = log.NewConsoleAppender
	NewDefaultLayout             = log.NewDefaultLayout
	NewFileAppender              = log.NewFileAppender
	NewHandler                   = log.NewHandler
	NewJSONAppender              = log.NewJSONAppender
	NewJSONFileAppender          = log.NewJSONFileAppender
	NewJSONTemplateLayout        = log.NewJSONTemplateLayout
	NewLogger                    = log.NewLogger
	NewNativeLogger              = log.NewNativeLogger
	NewRollingFileAppender       = log.NewRollingFileAppender
	WithAsyncOverflowStrategy    = log.WithAsyncOverflowStrategy
	WithAsyncQueueSize           = log.WithAsyncQueueSize
	WithAsyncWaitStrategy        = log.WithAsyncWaitStrategy
	WithConsoleLayout            = log.WithConsoleLayout
	WithConsoleWriter            = log.WithConsoleWriter
	WithFileBufferSize           = log.WithFileBufferSize
	WithFileLayout               = log.WithFileLayout
	WithJSONAppenderBufferSize   = log.WithJSONAppenderBufferSize
	WithJSONAppenderFlushOnWrite = log.WithJSONAppenderFlushOnWrite
	WithJSONAppenderWriter       = log.WithJSONAppenderWriter
	WithLoggerCaller             = log.WithLoggerCaller
	WithRollingAsyncActions      = log.WithRollingAsyncActions
	WithRollingFileBufferSize    = log.WithRollingFileBufferSize
	WithRollingFileLayout        = log.WithRollingFileLayout
	WithRollingFilePattern       = log.WithRollingFilePattern
	WithRollingGzip              = log.WithRollingGzip
	WithRollingMaxBackups        = log.WithRollingMaxBackups
	WithRollingMaxSize           = log.WithRollingMaxSize
)

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
	if ctx == nil {
		ctx = context.Background()
	}
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
	if ctx == nil {
		ctx = context.Background()
	}
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
