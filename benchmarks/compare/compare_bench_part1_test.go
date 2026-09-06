package compare

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"goark.dev/log"
)

func BenchmarkCompareDiscard(b *testing.B) {
	b.Run("goark-logattrs-json", func(b *testing.B) {
		handler, err := log.NewHandler(log.Options{
			Appenders: []log.Appender{
				log.NewConsoleAppender(
					log.WithConsoleWriter(io.Discard),
					log.WithConsoleLayout(log.JSONLayout{}),
				),
			},
			Root: log.RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"console"}},
		})
		if err != nil {
			b.Fatalf("NewHandler() error = %v", err)
		}
		defer handler.Close()
		logger := log.NewLogger(handler, "bench.compare")
		benchmarkGoarkLogAttrs(b, logger)
	})

	b.Run("goark-native-json", func(b *testing.B) {
		handler, err := log.NewHandler(log.Options{
			Appenders: []log.Appender{
				log.NewConsoleAppender(
					log.WithConsoleWriter(io.Discard),
					log.WithConsoleLayout(log.JSONLayout{}),
				),
			},
			Root: log.RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"console"}},
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

	b.Run("goark-native-direct-json", func(b *testing.B) {
		handler, err := log.NewHandler(log.Options{
			Appenders: []log.Appender{
				log.NewJSONAppender(log.WithJSONAppenderWriter(io.Discard)),
			},
			Root: log.RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"json"}},
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
		defer handler.Close()
		logger, err := log.NewNativeLogger(handler, "bench.compare")
		if err != nil {
			b.Fatalf("NewNativeLogger() error = %v", err)
		}
		benchmarkGoarkNative3(b, logger)
	})

	b.Run("goark-builder-direct-json", func(b *testing.B) {
		handler, err := log.NewHandler(log.Options{
			Appenders: []log.Appender{
				log.NewJSONAppender(log.WithJSONAppenderWriter(io.Discard)),
			},
			Root: log.RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"json"}},
		})
		if err != nil {
			b.Fatalf("NewHandler() error = %v", err)
		}
		defer handler.Close()
		logger, err := log.NewNativeLogger(handler, "bench.compare")
		if err != nil {
			b.Fatalf("NewNativeLogger() error = %v", err)
		}
		benchmarkGoarkBuilder(b, logger)
	})

	b.Run("goark-info-json", func(b *testing.B) {
		handler, err := log.NewHandler(log.Options{
			Appenders: []log.Appender{
				log.NewConsoleAppender(
					log.WithConsoleWriter(io.Discard),
					log.WithConsoleLayout(log.JSONLayout{}),
				),
			},
			Root: log.RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"console"}},
		})
		if err != nil {
			b.Fatalf("NewHandler() error = %v", err)
		}
		defer handler.Close()
		logger := log.NewLogger(handler, "bench.compare")
		benchmarkGoarkInfo(b, logger)
	})

	b.Run("zap-json", func(b *testing.B) {
		logger := newZapLogger(io.Discard)
		defer logger.Sync()
		benchmarkZap(b, logger)
	})

	b.Run("zerolog-json", func(b *testing.B) {
		logger := zerolog.New(io.Discard)
		benchmarkZerolog(b, logger)
	})
}

func benchmarkGoarkNativeParallel3(b *testing.B, logger *log.Logger) {
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		index := 0
		for pb.Next() {
			index++
			if err := logger.LogAttrs3(context.Background(), slog.LevelInfo, "event",
				slog.String("profile", "bench"),
				slog.Int("index", index),
				slog.Duration("elapsed", 10*time.Millisecond),
			); err != nil {
				b.Fatalf("LogAttrs3() error = %v", err)
			}
		}
	})
	b.StopTimer()
}

func benchmarkZapParallel(b *testing.B, logger *zap.Logger) {
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		index := 0
		for pb.Next() {
			index++
			logger.Info("event",
				zap.String("profile", "bench"),
				zap.Int("index", index),
				zap.Duration("elapsed", 10*time.Millisecond),
			)
		}
	})
	b.StopTimer()
}

func bufferedFileWriter(b *testing.B, name string) (io.Writer, func()) {
	b.Helper()
	file, err := os.Create(filePath(b, name))
	if err != nil {
		b.Fatalf("Create(%s) error = %v", name, err)
	}
	writer := bufio.NewWriterSize(file, 256*1024)
	return writer, func() {
		if err := writer.Flush(); err != nil {
			b.Fatalf("Flush(%s) error = %v", name, err)
		}
		if err := file.Close(); err != nil {
			b.Fatalf("Close(%s) error = %v", name, err)
		}
	}
}

func benchmarkGoarkNative3(b *testing.B, logger *log.Logger) {
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		if err := logger.LogAttrs3(context.Background(), slog.LevelInfo, "event",
			slog.String("profile", "bench"),
			slog.Int("index", index),
			slog.Duration("elapsed", 10*time.Millisecond),
		); err != nil {
			b.Fatalf("LogAttrs3() error = %v", err)
		}
	}
}

func benchmarkGoarkLogAttrs(b *testing.B, logger *slog.Logger) {
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		logger.LogAttrs(context.Background(), slog.LevelInfo, "event",
			slog.String("profile", "bench"),
			slog.Int("index", index),
			slog.Duration("elapsed", 10*time.Millisecond),
		)
	}
}

func benchmarkZap(b *testing.B, logger *zap.Logger) {
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		logger.Info("event",
			zap.String("profile", "bench"),
			zap.Int("index", index),
			zap.Duration("elapsed", 10*time.Millisecond),
		)
	}
}

func newZapLogger(writer io.Writer) *zap.Logger {
	config := zap.NewProductionEncoderConfig()
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(config),
		zapcore.AddSync(writer),
		zapcore.InfoLevel,
	)
	return zap.New(core)
}
