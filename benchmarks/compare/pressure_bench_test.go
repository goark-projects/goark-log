package compare

import (
	"io"
	"log/slog"
	"path/filepath"
	"sync"
	"testing"

	"github.com/rs/zerolog"
	goarklog "goark.dev/log"
)

func BenchmarkPressureParallelFile(b *testing.B) {
	b.Run("goark-native-direct-file-json3", func(b *testing.B) {
		appender, err := goarklog.NewJSONFileAppender(
			filePath(b, "goark-pressure.json"),
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
		logger, err := goarklog.NewNativeLogger(handler, "bench.pressure")
		if err != nil {
			b.Fatalf("NewNativeLogger() error = %v", err)
		}
		benchmarkGoarkNativeParallel3(b, logger)
		if err := handler.Close(); err != nil {
			b.Fatalf("Close() error = %v", err)
		}
	})

	b.Run("goark-rolling-json", func(b *testing.B) {
		dir := b.TempDir()
		appender, err := goarklog.NewRollingFileAppender(
			filepath.Join(dir, "goark-pressure-rolling.log"),
			goarklog.WithRollingFileLayout(goarklog.JSONLayout{}),
			goarklog.WithRollingFileBufferSize(256*1024),
			goarklog.WithRollingMaxSize(4*1024*1024),
			goarklog.WithRollingMaxBackups(8),
			goarklog.WithRollingFilePattern(filepath.Join(dir, "archive", "goark-%06i.log")),
		)
		if err != nil {
			b.Fatalf("NewRollingFileAppender() error = %v", err)
		}
		handler, err := goarklog.NewHandler(goarklog.Options{
			Appenders: []goarklog.Appender{appender},
			Root:      goarklog.RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"rollingFile"}},
		})
		if err != nil {
			b.Fatalf("NewHandler() error = %v", err)
		}
		logger, err := goarklog.NewNativeLogger(handler, "bench.pressure")
		if err != nil {
			b.Fatalf("NewNativeLogger() error = %v", err)
		}
		benchmarkGoarkNativeParallel3(b, logger)
		if err := handler.Close(); err != nil {
			b.Fatalf("Close() error = %v", err)
		}
	})

	b.Run("zap-json", func(b *testing.B) {
		writer, closeWriter := lockedBufferedFileWriter(b, "zap-pressure.json")
		logger := newZapLogger(writer)
		benchmarkZapParallel(b, logger)
		if err := logger.Sync(); err != nil {
			b.Fatalf("Sync() error = %v", err)
		}
		closeWriter()
	})

	b.Run("zerolog-json", func(b *testing.B) {
		writer, closeWriter := lockedBufferedFileWriter(b, "zerolog-pressure.json")
		logger := zerolog.New(writer)
		benchmarkZerologParallel(b, logger)
		closeWriter()
	})
}

func lockedBufferedFileWriter(b *testing.B, name string) (io.Writer, func()) {
	b.Helper()
	writer, closeWriter := bufferedFileWriter(b, name)
	locked := &lockedBenchmarkWriter{writer: writer}
	return locked, func() {
		locked.mu.Lock()
		defer locked.mu.Unlock()
		closeWriter()
	}
}

type lockedBenchmarkWriter struct {
	mu     sync.Mutex
	writer io.Writer
}

func (w *lockedBenchmarkWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.writer.Write(data)
}
