package goarklog

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFileAppender_whenPathHasMissingParent_shouldCreateAndAppend(t *testing.T) {
	path := filepath.Join(t.TempDir(), "logs", "app.log")
	appender, err := NewFileAppender(path, WithFileLayout(TextLayout{}))
	if err != nil {
		t.Fatalf("NewFileAppender() error = %v", err)
	}
	if err := appender.Append(context.Background(), testEvent("file written", fixedTestTime())); err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	if err := appender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(content), "logger=goark.test") || !strings.Contains(string(content), "msg=\"file written\"") {
		t.Fatalf("file content should contain rendered event, got %q", string(content))
	}
}

func TestFileAppender_whenPathIsDirectory_shouldReject(t *testing.T) {
	_, err := NewFileAppender(t.TempDir())
	if err == nil {
		t.Fatalf("NewFileAppender() should reject directory path")
	}
}

func testEvent(message string, when time.Time) Event {
	return Event{
		Time:    when,
		Level:   0,
		Message: message,
		Logger:  "goark.test",
	}
}

func fixedTestTime() time.Time {
	return time.Date(2026, 8, 25, 10, 15, 30, 123000000, time.FixedZone("CST", 8*3600))
}
