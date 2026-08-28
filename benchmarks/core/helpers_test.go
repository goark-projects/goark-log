package corebench

import (
	"context"
	"sync"
	"sync/atomic"

	goarklog "goark.dev/log"
)

const (
	AsyncOverflowBlock        = goarklog.AsyncOverflowBlock
	AsyncOverflowDrop         = goarklog.AsyncOverflowDrop
	AsyncOverflowDropDebug    = goarklog.AsyncOverflowDropDebug
	AsyncOverflowSyncFallback = goarklog.AsyncOverflowSyncFallback

	AsyncWaitBlock = goarklog.AsyncWaitBlock
	AsyncWaitYield = goarklog.AsyncWaitYield
)

type (
	Appender              = goarklog.Appender
	AsyncLoggerOptions    = goarklog.AsyncLoggerOptions
	AsyncOverflowStrategy = goarklog.AsyncOverflowStrategy
	AsyncWaitOptions      = goarklog.AsyncWaitOptions
	AsyncWaitStrategy     = goarklog.AsyncWaitStrategy
	Event                 = goarklog.Event
	JSONLayout            = goarklog.JSONLayout
	Layout                = goarklog.Layout
	Logger                = goarklog.Logger
	Options               = goarklog.Options
	RootLogger            = goarklog.RootLogger
	TextLayout            = goarklog.TextLayout
)

var (
	NewAsyncAppender             = goarklog.NewAsyncAppender
	NewConsoleAppender           = goarklog.NewConsoleAppender
	NewDefaultLayout             = goarklog.NewDefaultLayout
	NewFileAppender              = goarklog.NewFileAppender
	NewHandler                   = goarklog.NewHandler
	NewJSONAppender              = goarklog.NewJSONAppender
	NewJSONFileAppender          = goarklog.NewJSONFileAppender
	NewJSONTemplateLayout        = goarklog.NewJSONTemplateLayout
	NewLogger                    = goarklog.NewLogger
	NewNativeLogger              = goarklog.NewNativeLogger
	NewRollingFileAppender       = goarklog.NewRollingFileAppender
	WithAsyncOverflowStrategy    = goarklog.WithAsyncOverflowStrategy
	WithAsyncQueueSize           = goarklog.WithAsyncQueueSize
	WithAsyncWaitStrategy        = goarklog.WithAsyncWaitStrategy
	WithConsoleLayout            = goarklog.WithConsoleLayout
	WithConsoleWriter            = goarklog.WithConsoleWriter
	WithFileBufferSize           = goarklog.WithFileBufferSize
	WithFileLayout               = goarklog.WithFileLayout
	WithJSONAppenderBufferSize   = goarklog.WithJSONAppenderBufferSize
	WithJSONAppenderFlushOnWrite = goarklog.WithJSONAppenderFlushOnWrite
	WithJSONAppenderWriter       = goarklog.WithJSONAppenderWriter
	WithLoggerCaller             = goarklog.WithLoggerCaller
	WithRollingAsyncActions      = goarklog.WithRollingAsyncActions
	WithRollingFileBufferSize    = goarklog.WithRollingFileBufferSize
	WithRollingFileLayout        = goarklog.WithRollingFileLayout
	WithRollingFilePattern       = goarklog.WithRollingFilePattern
	WithRollingGzip              = goarklog.WithRollingGzip
	WithRollingMaxBackups        = goarklog.WithRollingMaxBackups
	WithRollingMaxSize           = goarklog.WithRollingMaxSize
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
