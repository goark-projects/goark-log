package integration

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	. "goark.dev/log/internal/testsupport"
)

func TestStressAsyncLoggerConcurrentClose(t *testing.T) {
	requireStressEnabled(t)

	delegate := newGatedAppender("delegate")
	t.Cleanup(delegate.releaseGate)
	handler, err := NewHandler(Options{
		Appenders: []Appender{delegate},
		Root:      RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"delegate"}},
		Async: AsyncLoggerOptions{
			Enabled:          true,
			QueueSize:        1,
			BatchSize:        1,
			OverflowStrategy: AsyncOverflowBlock,
			WaitStrategy:     AsyncWaitYield,
		},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	if err := handler.Handle(
		context.Background(),
		slog.NewRecord(time.Now(), slog.LevelInfo, "first", 0),
	); err != nil {
		t.Fatalf("Handle(first) error = %v", err)
	}
	<-delegate.started
	if err := handler.Handle(
		context.Background(),
		slog.NewRecord(time.Now(), slog.LevelInfo, "second", 0),
	); err != nil {
		t.Fatalf("Handle(second) error = %v", err)
	}

	blocked := stressWorkers() * 2
	ready := make(chan struct{}, blocked)
	appendDone := make(chan error, blocked)
	for index := 0; index < blocked; index++ {
		go func(index int) {
			ready <- struct{}{}
			record := slog.NewRecord(
				time.Now(),
				slog.LevelInfo,
				fmt.Sprintf("blocked-%d", index),
				0,
			)
			appendDone <- handler.Handle(context.Background(), record)
		}(index)
	}
	for index := 0; index < blocked; index++ {
		<-ready
	}

	closeDone := make(chan error, 1)
	go func() {
		closeDone <- handler.Close()
	}()

	for index := 0; index < blocked; index++ {
		select {
		case err := <-appendDone:
			if err == nil || !strings.Contains(err.Error(), "closed") {
				t.Fatalf("blocked Handle() error = %v, want closed error", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("blocked Handle() %d was not unblocked by Close()", index)
		}
	}

	delegate.releaseGate()
	select {
	case err := <-closeDone:
		if err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("Close() did not finish after delegate release")
	}
}

func TestStressAsyncLoggerConcurrentDrain(t *testing.T) {
	requireStressEnabled(t)

	workers := stressWorkers()
	perWorker := 512
	total := workers * perWorker
	delegate := NewRecordingAppender("memory")
	handler, err := NewHandler(Options{
		Appenders: []Appender{delegate},
		Root:      RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"memory"}},
		Async: AsyncLoggerOptions{
			Enabled:          true,
			QueueSize:        128,
			BatchSize:        32,
			OverflowStrategy: AsyncOverflowBlock,
			WaitStrategy:     AsyncWaitYield,
		},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	logger, err := NewNativeLogger(handler, "goark.pressure.async")
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
				if err := logger.LogAttrs3(context.Background(), slog.LevelInfo, "stress async",
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
	if handler.AsyncDropped() != 0 || handler.AsyncFailed() != 0 {
		t.Fatalf(
			"async counters dropped=%d failed=%d, want zero",
			handler.AsyncDropped(),
			handler.AsyncFailed(),
		)
	}
	assertStressRecordingIDs(t, delegate.Events(), total)
}

func assertStressGzipJSONFile(t *testing.T, path string) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open(%s) error = %v", path, err)
	}
	defer file.Close()
	reader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("NewReader(%s) error = %v", path, err)
	}
	defer reader.Close()
	scanner := bufio.NewScanner(reader)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			t.Fatalf("Scan(%s) error = %v", path, err)
		}
		t.Fatalf("compressed archive %s is empty", path)
	}
	var decoded struct {
		Message string `json:"msg"`
		ID      int    `json:"id"`
	}
	if err := sonic.Unmarshal(scanner.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid gzip JSON line in %s: %v\n%s", path, err, scanner.Text())
	}
	if decoded.Message != "stress gzip" || decoded.ID <= 0 {
		t.Fatalf("gzip JSON event = %+v, want stress gzip with positive id", decoded)
	}
}
func assertStressRecordingIDs(t *testing.T, events []Event, total int) {
	t.Helper()
	if len(events) != total {
		t.Fatalf("event count = %d, want %d", len(events), total)
	}
	seen := make([]bool, total+1)
	for _, event := range events {
		id, ok := stressEventIntAttr(event, "id")
		if !ok || id <= 0 || int(id) > total {
			t.Fatalf("event id = %d, ok=%v, want range 1..%d: %+v", id, ok, total, event)
		}
		if seen[int(id)] {
			t.Fatalf("duplicate event id %d", id)
		}
		seen[int(id)] = true
	}
	for id := 1; id <= total; id++ {
		if !seen[id] {
			t.Fatalf("missing event id %d", id)
		}
	}
}

func decodeStressJSONLine(t *testing.T, path string, line string) int {
	t.Helper()
	var decoded struct {
		Message string `json:"msg"`
		ID      int    `json:"id"`
	}
	if err := sonic.Unmarshal([]byte(line), &decoded); err != nil {
		t.Fatalf("invalid JSON log line in %s: %v\n%s", path, err, line)
	}
	if decoded.Message != "stress rolling" {
		t.Fatalf("JSON message in %s = %q, want stress rolling", path, decoded.Message)
	}
	return decoded.ID
}
