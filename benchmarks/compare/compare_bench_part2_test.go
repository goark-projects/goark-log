package compare

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"goark.dev/log"
)

func BenchmarkCompareBufferedFile(b *testing.B) {
	b.Run("goark-file-json", func(b *testing.B) {
		appender, err := log.NewFileAppender(
			filePath(b, "goark.log"),
			log.WithFileLayout(log.JSONLayout{}),
			log.WithFileBufferSize(256*1024),
		)
		if err != nil {
			b.Fatalf("NewFileAppender() error = %v", err)
		}
		defer appender.Close()
		handler, err := log.NewHandler(log.Options{
			Appenders: []log.Appender{appender},
			Root:      log.RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"file"}},
		})
		if err != nil {
			b.Fatalf("NewHandler() error = %v", err)
		}
		defer handler.Close()
		logger := log.NewLogger(handler, "bench.compare")
		benchmarkGoarkLogAttrs(b, logger)
	})

	b.Run("goark-native-file-json", func(b *testing.B) {
		appender, err := log.NewFileAppender(
			filePath(b, "goark-native.log"),
			log.WithFileLayout(log.JSONLayout{}),
			log.WithFileBufferSize(256*1024),
		)
		if err != nil {
			b.Fatalf("NewFileAppender() error = %v", err)
		}
		defer appender.Close()
		handler, err := log.NewHandler(log.Options{
			Appenders: []log.Appender{appender},
			Root:      log.RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"file"}},
		})
		if err != nil {
			b.Fatalf("NewHandler() error = %v", err)
		}
		defer handler.Close()
		logger, err := log.NewNativeLogger(handler, "bench.compare")
		if err != nil {
			b.Fatalf("NewNativeLogger() error = %v", err)
		}
		benchmarkGoarkNative(b, logger)
	})

	b.Run("goark-native-direct-file-json3", func(b *testing.B) {
		appender, err := log.NewJSONFileAppender(
			filePath(b, "goark-direct-native.log"),
			log.WithJSONAppenderBufferSize(256*1024),
		)
		if err != nil {
			b.Fatalf("NewJSONFileAppender() error = %v", err)
		}
		handler, err := log.NewHandler(log.Options{
			Appenders: []log.Appender{appender},
			Root:      log.RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"json"}},
		})
		if err != nil {
			b.Fatalf("NewHandler() error = %v", err)
		}
		defer handler.Close()
		logger, err := log.NewNativeLogger(handler, "bench.compare")
		if err != nil {
			b.Fatalf("NewNativeLogger() error = %v", err)
		}
		benchmarkGoarkNative3(b, logger)
	})

	b.Run("goark-rolling-json", func(b *testing.B) {
		appender, err := log.NewRollingFileAppender(
			filePath(b, "goark-rolling.log"),
			log.WithRollingFileLayout(log.JSONLayout{}),
			log.WithRollingFileBufferSize(256*1024),
			log.WithRollingMaxSize(1<<62),
			log.WithRollingMaxBackups(1),
		)
		if err != nil {
			b.Fatalf("NewRollingFileAppender() error = %v", err)
		}
		defer appender.Close()
		handler, err := log.NewHandler(log.Options{
			Appenders: []log.Appender{appender},
			Root:      log.RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"rollingFile"}},
		})
		if err != nil {
			b.Fatalf("NewHandler() error = %v", err)
		}
		defer handler.Close()
		logger := log.NewLogger(handler, "bench.compare")
		benchmarkGoarkLogAttrs(b, logger)
	})

	b.Run("zap-json", func(b *testing.B) {
		writer, closeWriter := bufferedFileWriter(b, "zap.log")
		defer closeWriter()
		logger := newZapLogger(writer)
		defer logger.Sync()
		benchmarkZap(b, logger)
	})

	b.Run("zerolog-json", func(b *testing.B) {
		writer, closeWriter := bufferedFileWriter(b, "zerolog.log")
		defer closeWriter()
		logger := zerolog.New(writer)
		benchmarkZerolog(b, logger)
	})
}

func BenchmarkCompareParallelDiscard(b *testing.B) {
	b.Run("goark-native-direct-json3", func(b *testing.B) {
		handler, err := log.NewHandler(log.Options{
			Appenders: []log.Appender{
				log.NewJSONAppender(log.WithJSONAppenderWriter(io.Discard)),
			},
			Root: log.RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"json"}},
		})
		if err != nil {
			b.Fatalf("NewHandler() error = %v", err)
		}
		logger, err := log.NewNativeLogger(handler, "bench.compare")
		if err != nil {
			b.Fatalf("NewNativeLogger() error = %v", err)
		}
		benchmarkGoarkNativeParallel3(b, logger)
		if err := handler.Close(); err != nil {
			b.Fatalf("Close() error = %v", err)
		}
	})

	b.Run("zap-json", func(b *testing.B) {
		logger := newZapLogger(io.Discard)
		benchmarkZapParallel(b, logger)
		if err := logger.Sync(); err != nil {
			b.Fatalf("Sync() error = %v", err)
		}
	})

	b.Run("zerolog-json", func(b *testing.B) {
		logger := zerolog.New(io.Discard)
		benchmarkZerologParallel(b, logger)
	})
}

func benchmarkZerologParallel(b *testing.B, logger zerolog.Logger) {
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		index := 0
		for pb.Next() {
			index++
			logger.Info().
				Str("profile", "bench").
				Int("index", index).
				Dur("elapsed", 10*time.Millisecond).
				Msg("event")
		}
	})
	b.StopTimer()
}

func benchmarkGoarkNative(b *testing.B, logger *log.Logger) {
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		if err := logger.LogAttrs(context.Background(), slog.LevelInfo, "event",
			slog.String("profile", "bench"),
			slog.Int("index", index),
			slog.Duration("elapsed", 10*time.Millisecond),
		); err != nil {
			b.Fatalf("LogAttrs() error = %v", err)
		}
	}
}

func benchmarkGoarkBuilder(b *testing.B, logger *log.Logger) {
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		if err := logger.AtInfo().
			WithString("profile", "bench").
			WithInt("index", index).
			WithAttr(slog.Duration("elapsed", 10*time.Millisecond)).
			Log("event"); err != nil {
			b.Fatalf("LogBuilder.Log() error = %v", err)
		}
	}
}

func benchmarkGoarkInfo(b *testing.B, logger *slog.Logger) {
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		logger.Info("event",
			slog.String("profile", "bench"),
			slog.Int("index", index),
			slog.Duration("elapsed", 10*time.Millisecond),
		)
	}
}

func benchmarkZerolog(b *testing.B, logger zerolog.Logger) {
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		logger.Info().
			Str("profile", "bench").
			Int("index", index).
			Dur("elapsed", 10*time.Millisecond).
			Msg("event")
	}
}

func filePath(b *testing.B, name string) string {
	b.Helper()
	return filepath.Join(b.TempDir(), name)
}
