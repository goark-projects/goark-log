package integration

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "goark.dev/log/internal/testsupport"
)

func TestFileAppender_whenPathHasMissingParent_shouldCreateAndAppend(t *testing.T) {
	path := filepath.Join(t.TempDir(), "logs", "app.log")
	appender, err := NewFileAppender(path, WithFileLayout(TextLayout{}))
	if err != nil {
		t.Fatalf("NewFileAppender() error = %v", err)
	}
	if err := appender.Append(
		context.Background(),
		TestEvent("file written", FixedTestTime()),
	); err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	if err := appender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(content), "logger=goark.test") ||
		!strings.Contains(string(content), "msg=\"file written\"") {
		t.Fatalf("file content should contain rendered event, got %q", string(content))
	}
}

func TestFileAppender_whenPathIsDirectory_shouldReject(t *testing.T) {
	_, err := NewFileAppender(t.TempDir())
	if err == nil {
		t.Fatalf("NewFileAppender() should reject directory path")
	}
}

func TestFileAppender_whenBuffered_shouldFlushBeforeClose(t *testing.T) {
	path := filepath.Join(t.TempDir(), "buffered.log")
	appender, err := NewFileAppender(path, WithFileLayout(TextLayout{}))
	if err != nil {
		t.Fatalf("NewFileAppender() error = %v", err)
	}
	if err := appender.Append(
		context.Background(),
		TestEvent("buffered", FixedTestTime()),
	); err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	if err := appender.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(content), "buffered") {
		t.Fatalf("flushed content is wrong: %q", string(content))
	}
	if err := appender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestFileAppender_whenCreateOnDemandClosedBeforeAppend_shouldNotOpenFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lazy-complete.json")
	appender, err := NewFileAppender(path,
		WithFileLayout(NewJSONLayout(LayoutOptions{Complete: true})),
		WithFileCreateOnDemand(true),
	)
	if err != nil {
		t.Fatalf("NewFileAppender() error = %v", err)
	}
	if err := appender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("file should not be created before first append, stat error = %v", err)
	}
}
