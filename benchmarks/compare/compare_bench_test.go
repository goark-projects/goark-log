package compare

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	goarklog "goark.dev/log"
)

func BenchmarkCompareDiscard(b *testing.B) {
	b.Run("goark-logattrs-json", func(b *testing.B) {
		handler, err := goarklog.NewHandler(goarklog.Options{
			Appenders: []goarklog.Appender{
				goarklog.NewConsoleAppender(
					goarklog.WithConsoleWriter(io.Discard),
					goarklog.WithConsoleLayout(goarklog.JSONLayout{}),
				),
			},
			Root: goarklog.RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"console"}},
		})
		if err != nil {
			b.Fatalf("NewHandler() error = %v", err)
		}
		defer handler.Close()
		logger := goarklog.NewLogger(handler, "bench.compare")
		benchmarkGoarkLogAttrs(b, logger)
	})

	b.Run("goark-native-json", func(b *testing.B) {
		handler, err := goarklog.NewHandler(goarklog.Options{
			Appenders: []goarklog.Appender{
				goarklog.NewConsoleAppender(
					goarklog.WithConsoleWriter(io.Discard),
					goarklog.WithConsoleLayout(goarklog.JSONLayout{}),
				),
			},
			Root: goarklog.RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"console"}},
		})
		if err != nil {
			b.Fatalf("NewHandler() error = %v", err)
		}
		defer handler.Close()
		logger, err := goarklog.NewNativeLogger(handler, "bench.compare")
		if err != nil {
			b.Fatalf("NewNativeLogger() error = %v", err)
		}
		benchmarkGoarkNative(b, logger)
	})

	b.Run("goark-native-direct-json", func(b *testing.B) {
		handler, err := goarklog.NewHandler(goarklog.Options{
			Appenders: []goarklog.Appender{
				goarklog.NewJSONAppender(goarklog.WithJSONAppenderWriter(io.Discard)),
			},
			Root: goarklog.RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"json"}},
		})
		if err != nil {
			b.Fatalf("NewHandler() error = %v", err)
		}
		defer handler.Close()
		logger, err := goarklog.NewNativeLogger(handler, "bench.compare")
		if err != nil {
			b.Fatalf("NewNativeLogger() error = %v", err)
		}
		benchmarkGoarkNative(b, logger)
	})

	b.Run("goark-native-direct-json3", func(b *testing.B) {
		handler, err := goarklog.NewHandler(goarklog.Options{
			Appenders: []goarklog.Appender{
				goarklog.NewJSONAppender(goarklog.WithJSONAppenderWriter(io.Discard)),
			},
			Root: goarklog.RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"json"}},
		})
		if err != nil {
			b.Fatalf("NewHandler() error = %v", err)
		}
		defer handler.Close()
		logger, err := goarklog.NewNativeLogger(handler, "bench.compare")
		if err != nil {
			b.Fatalf("NewNativeLogger() error = %v", err)
		}
		benchmarkGoarkNative3(b, logger)
	})

	b.Run("goark-builder-direct-json", func(b *testing.B) {
		handler, err := goarklog.NewHandler(goarklog.Options{
			Appenders: []goarklog.Appender{
				goarklog.NewJSONAppender(goarklog.WithJSONAppenderWriter(io.Discard)),
			},
			Root: goarklog.RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"json"}},
		})
		if err != nil {
			b.Fatalf("NewHandler() error = %v", err)
		}
		defer handler.Close()
		logger, err := goarklog.NewNativeLogger(handler, "bench.compare")
		if err != nil {
			b.Fatalf("NewNativeLogger() error = %v", err)
		}
		benchmarkGoarkBuilder(b, logger)
	})

	b.Run("goark-info-json", func(b *testing.B) {
		handler, err := goarklog.NewHandler(goarklog.Options{
			Appenders: []goarklog.Appender{
				goarklog.NewConsoleAppender(
					goarklog.WithConsoleWriter(io.Discard),
					goarklog.WithConsoleLayout(goarklog.JSONLayout{}),
				),
			},
			Root: goarklog.RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"console"}},
		})
		if err != nil {
			b.Fatalf("NewHandler() error = %v", err)
		}
		defer handler.Close()
		logger := goarklog.NewLogger(handler, "bench.compare")
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

func BenchmarkCompareBufferedFile(b *testing.B) {
	b.Run("goark-file-json", func(b *testing.B) {
		appender, err := goarklog.NewFileAppender(
			filePath(b, "goark.log"),
			goarklog.WithFileLayout(goarklog.JSONLayout{}),
			goarklog.WithFileBufferSize(256*1024),
		)
		if err != nil {
			b.Fatalf("NewFileAppender() error = %v", err)
		}
		defer appender.Close()
		handler, err := goarklog.NewHandler(goarklog.Options{
			Appenders: []goarklog.Appender{appender},
			Root:      goarklog.RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"file"}},
		})
		if err != nil {
			b.Fatalf("NewHandler() error = %v", err)
		}
		defer handler.Close()
		logger := goarklog.NewLogger(handler, "bench.compare")
		benchmarkGoarkLogAttrs(b, logger)
	})

	b.Run("goark-native-file-json", func(b *testing.B) {
		appender, err := goarklog.NewFileAppender(
			filePath(b, "goark-native.log"),
			goarklog.WithFileLayout(goarklog.JSONLayout{}),
			goarklog.WithFileBufferSize(256*1024),
		)
		if err != nil {
			b.Fatalf("NewFileAppender() error = %v", err)
		}
		defer appender.Close()
		handler, err := goarklog.NewHandler(goarklog.Options{
			Appenders: []goarklog.Appender{appender},
			Root:      goarklog.RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"file"}},
		})
		if err != nil {
			b.Fatalf("NewHandler() error = %v", err)
		}
		defer handler.Close()
		logger, err := goarklog.NewNativeLogger(handler, "bench.compare")
		if err != nil {
			b.Fatalf("NewNativeLogger() error = %v", err)
		}
		benchmarkGoarkNative(b, logger)
	})

	b.Run("goark-native-direct-file-json3", func(b *testing.B) {
		appender, err := goarklog.NewJSONFileAppender(
			filePath(b, "goark-direct-native.log"),
			goarklog.WithJSONAppenderBufferSize(256*1024),
		)
		if err != nil {
			b.Fatalf("NewJSONFileAppender() error = %v", err)
		}
		handler, err := goarklog.NewHandler(goarklog.Options{
			Appenders: []goarklog.Appender{appender},
			Root:      goarklog.RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"json"}},
		})
		if err != nil {
			b.Fatalf("NewHandler() error = %v", err)
		}
		defer handler.Close()
		logger, err := goarklog.NewNativeLogger(handler, "bench.compare")
		if err != nil {
			b.Fatalf("NewNativeLogger() error = %v", err)
		}
		benchmarkGoarkNative3(b, logger)
	})

	b.Run("goark-rolling-json", func(b *testing.B) {
		appender, err := goarklog.NewRollingFileAppender(
			filePath(b, "goark-rolling.log"),
			goarklog.WithRollingFileLayout(goarklog.JSONLayout{}),
			goarklog.WithRollingFileBufferSize(256*1024),
			goarklog.WithRollingMaxSize(1<<62),
			goarklog.WithRollingMaxBackups(1),
		)
		if err != nil {
			b.Fatalf("NewRollingFileAppender() error = %v", err)
		}
		defer appender.Close()
		handler, err := goarklog.NewHandler(goarklog.Options{
			Appenders: []goarklog.Appender{appender},
			Root:      goarklog.RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"rollingFile"}},
		})
		if err != nil {
			b.Fatalf("NewHandler() error = %v", err)
		}
		defer handler.Close()
		logger := goarklog.NewLogger(handler, "bench.compare")
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
		handler, err := goarklog.NewHandler(goarklog.Options{
			Appenders: []goarklog.Appender{
				goarklog.NewJSONAppender(goarklog.WithJSONAppenderWriter(io.Discard)),
			},
			Root: goarklog.RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"json"}},
		})
		if err != nil {
			b.Fatalf("NewHandler() error = %v", err)
		}
		logger, err := goarklog.NewNativeLogger(handler, "bench.compare")
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

func benchmarkGoarkNative(b *testing.B, logger *goarklog.Logger) {
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

func benchmarkGoarkNative3(b *testing.B, logger *goarklog.Logger) {
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

func benchmarkGoarkNativeParallel3(b *testing.B, logger *goarklog.Logger) {
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

func benchmarkGoarkBuilder(b *testing.B, logger *goarklog.Logger) {
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

func newZapLogger(writer io.Writer) *zap.Logger {
	config := zap.NewProductionEncoderConfig()
	core := zapcore.NewCore(zapcore.NewJSONEncoder(config), zapcore.AddSync(writer), zapcore.InfoLevel)
	return zap.New(core)
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

func filePath(b *testing.B, name string) string {
	b.Helper()
	return filepath.Join(b.TempDir(), name)
}
