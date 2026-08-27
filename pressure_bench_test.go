package goarklog

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"
)

func BenchmarkPressureAsyncLoggerQueueMatrix(b *testing.B) {
	cases := []struct {
		name     string
		queue    int
		batch    int
		overflow AsyncOverflowStrategy
		wait     AsyncWaitStrategy
		options  AsyncWaitOptions
	}{
		{name: "q1024-b64-block-block", queue: 1024, batch: 64, overflow: AsyncOverflowBlock, wait: AsyncWaitBlock},
		{name: "q8192-b256-block-yield", queue: 8192, batch: 256, overflow: AsyncOverflowBlock, wait: AsyncWaitYield},
		{name: "q8192-b256-block-timeout", queue: 8192, batch: 256, overflow: AsyncOverflowBlock, wait: AsyncWaitBlock, options: AsyncWaitOptions{Timeout: time.Millisecond}},
		{name: "q65536-b1024-block-yield", queue: 65536, batch: 1024, overflow: AsyncOverflowBlock, wait: AsyncWaitYield},
		{name: "q8192-b256-drop-yield", queue: 8192, batch: 256, overflow: AsyncOverflowDrop, wait: AsyncWaitYield},
		{name: "q8192-b256-sync-yield", queue: 8192, batch: 256, overflow: AsyncOverflowSyncFallback, wait: AsyncWaitYield},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			handler, err := NewHandler(Options{
				Appenders: []Appender{NewJSONAppender(WithJSONAppenderWriter(io.Discard))},
				Root:      RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"json"}},
				Async: AsyncLoggerOptions{
					Enabled:          true,
					QueueSize:        tc.queue,
					BatchSize:        tc.batch,
					OverflowStrategy: tc.overflow,
					WaitStrategy:     tc.wait,
					WaitOptions:      tc.options,
				},
			})
			if err != nil {
				b.Fatalf("NewHandler() error = %v", err)
			}
			logger, err := NewNativeLogger(handler, "goark.pressure.async")
			if err != nil {
				b.Fatalf("NewNativeLogger() error = %v", err)
			}
			benchmarkNativeLoggerParallel3(b, logger)
			if err := handler.Close(); err != nil {
				b.Fatalf("Close() error = %v", err)
			}
			b.ReportMetric(float64(handler.AsyncDropped()), "dropped")
			b.ReportMetric(float64(handler.AsyncFailed()), "failed")
		})
	}
}

func BenchmarkPressureJSONFileParallel(b *testing.B) {
	cases := []struct {
		name         string
		bufferSize   int
		flushOnWrite bool
	}{
		{name: "unbuffered", bufferSize: 0},
		{name: "buffered-256k", bufferSize: 256 * 1024},
		{name: "buffered-256k-flush", bufferSize: 256 * 1024, flushOnWrite: true},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			appender, err := NewJSONFileAppender(filepath.Join(b.TempDir(), "pressure.json"),
				WithJSONAppenderBufferSize(tc.bufferSize),
				WithJSONAppenderFlushOnWrite(tc.flushOnWrite),
			)
			if err != nil {
				b.Fatalf("NewJSONFileAppender() error = %v", err)
			}
			handler, err := NewHandler(Options{
				Appenders: []Appender{appender},
				Root:      RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"json"}},
			})
			if err != nil {
				b.Fatalf("NewHandler() error = %v", err)
			}
			logger, err := NewNativeLogger(handler, "goark.pressure.file")
			if err != nil {
				b.Fatalf("NewNativeLogger() error = %v", err)
			}
			benchmarkNativeLoggerParallel3(b, logger)
			if err := handler.Close(); err != nil {
				b.Fatalf("Close() error = %v", err)
			}
		})
	}
}

func BenchmarkPressureRollingFileParallel(b *testing.B) {
	cases := []struct {
		name    string
		gzip    bool
		async   bool
		maxSize int64
	}{
		{name: "plain", maxSize: 4 * 1024 * 1024},
		{name: "gzip-sync", gzip: true, maxSize: 4 * 1024 * 1024},
		{name: "gzip-async", gzip: true, async: true, maxSize: 4 * 1024 * 1024},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			dir := b.TempDir()
			filePattern := filepath.Join(dir, "archive", fmt.Sprintf("%s-%%06i.log", tc.name))
			if tc.gzip {
				filePattern += ".gz"
			}
			appender, err := NewRollingFileAppender(filepath.Join(dir, "app.log"),
				WithRollingFileLayout(JSONLayout{}),
				WithRollingFileBufferSize(256*1024),
				WithRollingMaxSize(tc.maxSize),
				WithRollingMaxBackups(8),
				WithRollingFilePattern(filePattern),
				WithRollingGzip(tc.gzip),
				WithRollingAsyncActions(tc.async),
			)
			if err != nil {
				b.Fatalf("NewRollingFileAppender() error = %v", err)
			}
			handler, err := NewHandler(Options{
				Appenders: []Appender{appender},
				Root:      RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"rollingFile"}},
			})
			if err != nil {
				b.Fatalf("NewHandler() error = %v", err)
			}
			logger, err := NewNativeLogger(handler, "goark.pressure.rolling")
			if err != nil {
				b.Fatalf("NewNativeLogger() error = %v", err)
			}
			benchmarkNativeLoggerParallel3(b, logger)
			if err := handler.Close(); err != nil {
				b.Fatalf("Close() error = %v", err)
			}
		})
	}
}

func BenchmarkPressureAsyncAppenderOverflow(b *testing.B) {
	cases := []AsyncOverflowStrategy{
		AsyncOverflowDrop,
		AsyncOverflowDropDebug,
		AsyncOverflowSyncFallback,
	}
	for _, strategy := range cases {
		b.Run(string(strategy), func(b *testing.B) {
			delegate := newGatedAppender("delegate")
			appender, err := NewAsyncAppender([]Appender{delegate},
				WithAsyncQueueSize(1),
				WithAsyncOverflowStrategy(strategy),
				WithAsyncWaitStrategy(AsyncWaitYield),
			)
			if err != nil {
				b.Fatalf("NewAsyncAppender() error = %v", err)
			}
			if err := appender.Append(context.Background(), benchmarkEvent()); err != nil {
				b.Fatalf("Append(first) error = %v", err)
			}
			<-delegate.started
			if err := appender.Append(context.Background(), benchmarkEvent()); err != nil {
				b.Fatalf("Append(second) error = %v", err)
			}
			event := benchmarkEvent()
			if strategy == AsyncOverflowDropDebug {
				event.Level = slog.LevelDebug
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := appender.Append(context.Background(), event); err != nil {
					b.Fatalf("Append() error = %v", err)
				}
			}
			b.StopTimer()
			delegate.releaseGate()
			if err := appender.Close(); err != nil {
				b.Fatalf("Close() error = %v", err)
			}
			b.ReportMetric(float64(appender.Dropped()), "dropped")
			b.ReportMetric(float64(appender.Failed()), "failed")
		})
	}
}

func BenchmarkPressureNativeLoggerWithCaller(b *testing.B) {
	handler, err := NewHandler(Options{
		Appenders: []Appender{NewJSONAppender(WithJSONAppenderWriter(io.Discard))},
		Root:      RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"json"}, IncludeLocation: true},
	})
	if err != nil {
		b.Fatalf("NewHandler() error = %v", err)
	}
	logger, err := NewNativeLogger(handler, "goark.pressure.caller", WithLoggerCaller(true))
	if err != nil {
		b.Fatalf("NewNativeLogger() error = %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := logger.LogAttrs3(context.Background(), slog.LevelInfo, "event",
			slog.String("profile", "bench"),
			slog.Int("index", i),
			slog.Duration("elapsed", 10*time.Millisecond),
		); err != nil {
			b.Fatalf("LogAttrs3() error = %v", err)
		}
	}
	b.StopTimer()
	if err := handler.Close(); err != nil {
		b.Fatalf("Close() error = %v", err)
	}
}
