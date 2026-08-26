package goarklog

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"
)

func BenchmarkLayout(b *testing.B) {
	event := benchmarkEvent()
	benchmarks := []struct {
		name   string
		layout Layout
	}{
		{name: "pattern", layout: NewDefaultLayout()},
		{name: "text", layout: TextLayout{}},
		{name: "json", layout: JSONLayout{}},
		{name: "json-template", layout: mustBenchmarkJSONTemplateLayout(b)},
	}
	for _, benchmark := range benchmarks {
		b.Run(benchmark.name, func(b *testing.B) {
			b.ReportAllocs()
			buf := bufferPool.Get().(*bytes.Buffer)
			defer releaseBuffer(buf)
			for i := 0; i < b.N; i++ {
				buf.Reset()
				if err := benchmark.layout.Format(buf, event); err != nil {
					b.Fatalf("Format() error = %v", err)
				}
			}
		})
	}
}

func BenchmarkJSONLayoutAnyFallback(b *testing.B) {
	event := benchmarkEvent()
	event.Attrs = append(event.Attrs, slog.Any("payload", map[string]any{
		"traceId": "abc-123",
		"attempt": 3,
		"tags":    []string{"core", "json", "sonic"},
	}))
	layout := JSONLayout{}
	buf := bufferPool.Get().(*bytes.Buffer)
	defer releaseBuffer(buf)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		if err := layout.Format(buf, event); err != nil {
			b.Fatalf("Format() error = %v", err)
		}
	}
}

func mustBenchmarkJSONTemplateLayout(b *testing.B) Layout {
	b.Helper()
	layout, err := NewJSONTemplateLayout("")
	if err != nil {
		b.Fatalf("NewJSONTemplateLayout() error = %v", err)
	}
	return layout
}

func BenchmarkAppender(b *testing.B) {
	b.Run("console-discard", func(b *testing.B) {
		appender := NewConsoleAppender(WithConsoleWriter(io.Discard), WithConsoleLayout(TextLayout{}))
		benchmarkAppender(b, appender)
	})
	b.Run("file", func(b *testing.B) {
		appender, err := NewFileAppender(filepath.Join(b.TempDir(), "app.log"), WithFileLayout(TextLayout{}))
		if err != nil {
			b.Fatalf("NewFileAppender() error = %v", err)
		}
		defer appender.Close()
		benchmarkAppender(b, appender)
	})
	b.Run("rolling-file", func(b *testing.B) {
		appender, err := NewRollingFileAppender(filepath.Join(b.TempDir(), "app.log"),
			WithRollingFileLayout(TextLayout{}),
			WithRollingMaxSize(64*1024*1024),
			WithRollingMaxBackups(3),
		)
		if err != nil {
			b.Fatalf("NewRollingFileAppender() error = %v", err)
		}
		defer appender.Close()
		benchmarkAppender(b, appender)
	})
}

func BenchmarkHandler(b *testing.B) {
	handler, err := NewHandler(Options{
		Appenders: []Appender{NewConsoleAppender(WithConsoleWriter(io.Discard), WithConsoleLayout(TextLayout{}))},
		Root:      RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"console"}},
	})
	if err != nil {
		b.Fatalf("NewHandler() error = %v", err)
	}
	logger := NewLogger(handler, "goark.bench")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("event", slog.String("profile", "bench"), slog.Int("index", i))
	}
}

func BenchmarkHandlerLogAttrs(b *testing.B) {
	handler, err := NewHandler(Options{
		Appenders: []Appender{NewConsoleAppender(WithConsoleWriter(io.Discard), WithConsoleLayout(TextLayout{}))},
		Root:      RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"console"}},
	})
	if err != nil {
		b.Fatalf("NewHandler() error = %v", err)
	}
	logger := NewLogger(handler, "goark.bench")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.LogAttrs(context.Background(), slog.LevelInfo, "event",
			slog.String("profile", "bench"),
			slog.Int("index", i),
		)
	}
}

func BenchmarkNativeLoggerLogAttrs(b *testing.B) {
	handler, err := NewHandler(Options{
		Appenders: []Appender{NewConsoleAppender(WithConsoleWriter(io.Discard), WithConsoleLayout(TextLayout{}))},
		Root:      RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"console"}},
	})
	if err != nil {
		b.Fatalf("NewHandler() error = %v", err)
	}
	logger, err := NewNativeLogger(handler, "goark.bench")
	if err != nil {
		b.Fatalf("NewNativeLogger() error = %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := logger.LogAttrs(context.Background(), slog.LevelInfo, "event",
			slog.String("profile", "bench"),
			slog.Int("index", i),
		); err != nil {
			b.Fatalf("LogAttrs() error = %v", err)
		}
	}
}

func BenchmarkNativeLoggerDirectJSON(b *testing.B) {
	handler, err := NewHandler(Options{
		Appenders: []Appender{NewJSONAppender(WithJSONAppenderWriter(io.Discard))},
		Root:      RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"json"}},
	})
	if err != nil {
		b.Fatalf("NewHandler() error = %v", err)
	}
	logger, err := NewNativeLogger(handler, "goark.bench")
	if err != nil {
		b.Fatalf("NewNativeLogger() error = %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := logger.LogAttrs(context.Background(), slog.LevelInfo, "event",
			slog.String("profile", "bench"),
			slog.Int("index", i),
		); err != nil {
			b.Fatalf("LogAttrs() error = %v", err)
		}
	}
}

func BenchmarkNativeLoggerDirectJSON3(b *testing.B) {
	handler, err := NewHandler(Options{
		Appenders: []Appender{NewJSONAppender(WithJSONAppenderWriter(io.Discard))},
		Root:      RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"json"}},
	})
	if err != nil {
		b.Fatalf("NewHandler() error = %v", err)
	}
	logger, err := NewNativeLogger(handler, "goark.bench")
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
}

func BenchmarkNativeLoggerDirectJSONFile3(b *testing.B) {
	appender, err := NewJSONFileAppender(filepath.Join(b.TempDir(), "direct.json"),
		WithJSONAppenderBufferSize(256*1024),
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
	logger, err := NewNativeLogger(handler, "goark.bench")
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

func BenchmarkNativeLoggerBuilder(b *testing.B) {
	handler, err := NewHandler(Options{
		Appenders: []Appender{NewConsoleAppender(WithConsoleWriter(io.Discard), WithConsoleLayout(TextLayout{}))},
		Root:      RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"console"}},
	})
	if err != nil {
		b.Fatalf("NewHandler() error = %v", err)
	}
	logger, err := NewNativeLogger(handler, "goark.bench")
	if err != nil {
		b.Fatalf("NewNativeLogger() error = %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := logger.AtInfo().
			WithString("profile", "bench").
			WithInt("index", i).
			Log("event"); err != nil {
			b.Fatalf("LogBuilder.Log() error = %v", err)
		}
	}
}

func BenchmarkNativeLoggerBuilderDirectJSON(b *testing.B) {
	handler, err := NewHandler(Options{
		Appenders: []Appender{NewJSONAppender(WithJSONAppenderWriter(io.Discard))},
		Root:      RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"json"}},
	})
	if err != nil {
		b.Fatalf("NewHandler() error = %v", err)
	}
	logger, err := NewNativeLogger(handler, "goark.bench")
	if err != nil {
		b.Fatalf("NewNativeLogger() error = %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := logger.AtInfo().
			WithString("profile", "bench").
			WithInt("index", i).
			Log("event"); err != nil {
			b.Fatalf("LogBuilder.Log() error = %v", err)
		}
	}
}

func BenchmarkAsyncAppender(b *testing.B) {
	strategies := []AsyncOverflowStrategy{
		AsyncOverflowBlock,
		AsyncOverflowDrop,
		AsyncOverflowDropDebug,
		AsyncOverflowSyncFallback,
	}
	for _, strategy := range strategies {
		b.Run(string(strategy), func(b *testing.B) {
			appender, err := NewAsyncAppender([]Appender{
				NewConsoleAppender(WithConsoleWriter(io.Discard), WithConsoleLayout(TextLayout{})),
			}, WithAsyncQueueSize(1024), WithAsyncOverflowStrategy(strategy))
			if err != nil {
				b.Fatalf("NewAsyncAppender() error = %v", err)
			}
			event := benchmarkEvent()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := appender.Append(context.Background(), event); err != nil {
					b.Fatalf("Append() error = %v", err)
				}
			}
			b.StopTimer()
			if err := appender.Close(); err != nil {
				b.Fatalf("Close() error = %v", err)
			}
			b.ReportMetric(float64(appender.Dropped()), "dropped")
		})
	}
}

func BenchmarkAsyncAppenderOverflow(b *testing.B) {
	strategies := []AsyncOverflowStrategy{
		AsyncOverflowDrop,
		AsyncOverflowDropDebug,
		AsyncOverflowSyncFallback,
	}
	for _, strategy := range strategies {
		b.Run(string(strategy), func(b *testing.B) {
			delegate := newGatedAppender("delegate")
			appender, err := NewAsyncAppender([]Appender{delegate},
				WithAsyncQueueSize(1),
				WithAsyncOverflowStrategy(strategy),
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
		})
	}
}

func benchmarkAppender(b *testing.B, appender Appender) {
	event := benchmarkEvent()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := appender.Append(context.Background(), event); err != nil {
			b.Fatalf("Append() error = %v", err)
		}
	}
}

func benchmarkEvent() Event {
	return Event{
		Time:    time.Date(2026, 8, 25, 10, 15, 30, 123000000, time.FixedZone("CST", 8*3600)),
		Level:   slog.LevelInfo,
		Message: "service started",
		Logger:  "goark.bench",
		Attrs: []slog.Attr{
			slog.String("profile", "bench"),
			slog.Int("index", 42),
			slog.Duration("elapsed", 10*time.Millisecond),
		},
	}
}
