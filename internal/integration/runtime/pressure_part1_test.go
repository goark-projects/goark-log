package integration

import (
	"bufio"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	. "goark.dev/log/internal/testsupport"
)

func TestStressRollingFileConcurrentLineIntegrity(t *testing.T) {
	requireStressEnabled(t)

	workers := stressWorkers()
	perWorker := 384
	total := workers * perWorker
	dir := t.TempDir()
	activePath := filepath.Join(dir, "app.log")
	archivePattern := filepath.Join(dir, "archive", "app-%d{UNIX_NANOS}-%06i.log")
	appender, err := NewRollingFileAppender(activePath,
		WithRollingFileLayout(JSONLayout{}),
		WithRollingFileBufferSize(64*1024),
		WithRollingMaxSize(4096),
		WithRollingMaxBackups(total),
		WithRollingFilePattern(archivePattern),
	)
	if err != nil {
		t.Fatalf("NewRollingFileAppender() error = %v", err)
	}
	handler, err := NewHandler(Options{
		Appenders: []Appender{appender},
		Root:      RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"rollingFile"}},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	logger, err := NewNativeLogger(handler, "goark.pressure.rolling")
	if err != nil {
		t.Fatalf("NewNativeLogger() error = %v", err)
	}

	var nextID atomic.Int64
	errCh := make(chan error, workers)
	var wait sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wait.Add(1)
		go func(worker int) {
			defer wait.Done()
			for index := 0; index < perWorker; index++ {
				id := int(nextID.Add(1))
				if err := logger.LogAttrs3(context.Background(), slog.LevelInfo, "stress rolling",
					slog.Int("id", id),
					slog.Int("worker", worker),
					slog.Int("index", index),
				); err != nil {
					errCh <- err
					return
				}
			}
		}(worker)
	}
	wait.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("LogAttrs3() error = %v", err)
		}
	}
	if err := handler.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	paths := []string{activePath}
	archives, err := filepath.Glob(filepath.Join(dir, "archive", "app-*.log"))
	if err != nil {
		t.Fatalf("Glob(archives) error = %v", err)
	}
	paths = append(paths, archives...)
	assertStressJSONLogFiles(t, paths, total)
}

func TestStressRollingFileAsyncActionsClose(t *testing.T) {
	requireStressEnabled(t)

	now := time.Now()
	dir := t.TempDir()
	activePath := filepath.Join(dir, "app.log")
	archivePattern := filepath.Join(dir, "archive", "app-%d{UNIX_NANOS}-%06i.log.gz")
	appender, err := NewRollingFileAppender(activePath,
		WithRollingFileLayout(JSONLayout{}),
		WithRollingFileBufferSize(16*1024),
		WithRollingMaxSize(512),
		WithRollingMaxBackups(128),
		WithRollingFilePattern(archivePattern),
		WithRollingGzip(true),
		WithRollingAsyncActions(true),
		WithRollingActionQueueSize(2),
		WithRollingClock(func() time.Time { return now }),
	)
	if err != nil {
		t.Fatalf("NewRollingFileAppender() error = %v", err)
	}
	for index := 0; index < 128; index++ {
		if err := appender.Append(context.Background(), Event{
			Time:    now,
			Level:   slog.LevelInfo,
			Logger:  "goark.pressure.rolling",
			Message: "stress gzip",
			Attrs: []slog.Attr{
				slog.Int("id", index+1),
				slog.String("payload", strings.Repeat("x", 128)),
			},
		}); err != nil {
			t.Fatalf("Append(%d) error = %v", index, err)
		}
	}
	if err := appender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	uncompressed, err := filepath.Glob(filepath.Join(dir, "archive", "*.log"))
	if err != nil {
		t.Fatalf("Glob(uncompressed) error = %v", err)
	}
	if len(uncompressed) != 0 {
		t.Fatalf("async gzip actions left uncompressed archives: %v", uncompressed)
	}
	compressed, err := filepath.Glob(filepath.Join(dir, "archive", "*.log.gz"))
	if err != nil {
		t.Fatalf("Glob(compressed) error = %v", err)
	}
	if len(compressed) == 0 {
		t.Fatalf("expected compressed archives after async rolling actions")
	}
	assertStressGzipJSONFile(t, compressed[0])
}

func assertStressJSONLogFiles(t *testing.T, paths []string, total int) {
	t.Helper()
	seen := make([]bool, total+1)
	lines := 0
	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			t.Fatalf("Open(%s) error = %v", path, err)
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.TrimSpace(line) == "" {
				continue
			}
			id := decodeStressJSONLine(t, path, line)
			if id <= 0 || id > total {
				_ = file.Close()
				t.Fatalf("line id = %d, want range 1..%d: %s", id, total, line)
			}
			if seen[id] {
				_ = file.Close()
				t.Fatalf("duplicate JSON event id %d in %s", id, path)
			}
			seen[id] = true
			lines++
		}
		if err := scanner.Err(); err != nil {
			_ = file.Close()
			t.Fatalf("Scan(%s) error = %v", path, err)
		}
		if err := file.Close(); err != nil {
			t.Fatalf("Close(%s) error = %v", path, err)
		}
	}
	if lines != total {
		t.Fatalf("JSON line count = %d, want %d", lines, total)
	}
	for id := 1; id <= total; id++ {
		if !seen[id] {
			t.Fatalf("missing JSON event id %d", id)
		}
	}
}

func stressEventIntAttr(event Event, key string) (int64, bool) {
	value, ok := event.Attr(key)
	if !ok {
		return 0, false
	}
	value = value.Resolve()
	switch value.Kind() {
	case slog.KindInt64:
		return value.Int64(), true
	case slog.KindUint64:
		return int64(value.Uint64()), true
	default:
		return 0, false
	}
}

func stressWorkers() int {
	workers := runtime.GOMAXPROCS(0)
	if workers < 4 {
		return 4
	}
	if workers > 16 {
		return 16
	}
	return workers
}

func requireStressEnabled(t *testing.T) {
	t.Helper()
	if os.Getenv("GOARK_LOG_STRESS") != "1" {
		t.Skip("set GOARK_LOG_STRESS=1 to run long pressure tests")
	}
}
