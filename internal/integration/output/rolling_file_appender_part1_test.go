package integration

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "goark.dev/log/internal/testsupport"
)

func TestRollingFileAppender_whenAsyncDeleteActionConfigured_shouldDeleteExpiredArchivesOnClose(
	t *testing.T,
) {
	now := FixedTestTime()
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	archiveDir := filepath.Join(dir, "archive")
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	expired := filepath.Join(archiveDir, "expired.log.gz")
	if err := os.WriteFile(expired, []byte("expired"), 0o644); err != nil {
		t.Fatalf("WriteFile(expired) error = %v", err)
	}
	old := now.Add(-48 * time.Hour)
	if err := os.Chtimes(expired, old, old); err != nil {
		t.Fatalf("Chtimes(expired) error = %v", err)
	}
	fresh := filepath.Join(archiveDir, "fresh.log.gz")
	if err := os.WriteFile(fresh, []byte("fresh"), 0o644); err != nil {
		t.Fatalf("WriteFile(fresh) error = %v", err)
	}
	if err := os.Chtimes(fresh, now, now); err != nil {
		t.Fatalf("Chtimes(fresh) error = %v", err)
	}

	appender, err := NewRollingFileAppender(path,
		WithRollingFileLayout(TextLayout{}),
		WithRollingMaxSize(120),
		WithRollingFilePattern(filepath.Join(archiveDir, "app-%d{yyyyMMdd}-%i.log.gz")),
		WithRollingMaxBackups(10),
		WithRollingGzip(true),
		WithRollingAsyncActions(true),
		WithRollingDeleteActions(RollingDeleteAction{
			BasePath: archiveDir,
			Glob:     "*.log.gz",
			MaxAge:   24 * time.Hour,
		}),
		WithRollingClock(func() time.Time { return now }),
	)
	if err != nil {
		t.Fatalf("NewRollingFileAppender() error = %v", err)
	}
	for index := 0; index < 2; index++ {
		if err := appender.Append(
			context.Background(),
			TestEvent("async-delete-"+strings.Repeat("x", 80), now),
		); err != nil {
			t.Fatalf("Append(%d) error = %v", index, err)
		}
	}
	if err := appender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if _, err := os.Stat(expired); !os.IsNotExist(err) {
		t.Fatalf("expired archive should be deleted, stat error = %v", err)
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Fatalf("fresh archive should remain, stat error = %v", err)
	}
	archives, err := filepath.Glob(filepath.Join(archiveDir, "app-*.gz"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(archives) != 1 {
		t.Fatalf("rolled archive should be compressed before Close returns, got %v", archives)
	}
}

func TestRollingFileAppender_whenStartupEnabled_shouldArchiveExistingFile(t *testing.T) {
	now := FixedTestTime()
	path := filepath.Join(t.TempDir(), "startup.log")
	if err := os.WriteFile(path, []byte("before-startup\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	appender, err := NewRollingFileAppender(path,
		WithRollingMaxSize(0),
		WithRolloverOnStartup(true),
		WithRollingMaxBackups(10),
		WithRollingClock(func() time.Time { return now }),
	)
	if err != nil {
		t.Fatalf("NewRollingFileAppender() error = %v", err)
	}
	if err := appender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	archives, err := filepath.Glob(path + ".*")
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(archives) != 1 {
		t.Fatalf("expected one startup archive, got %d: %v", len(archives), archives)
	}
	archiveContent, err := os.ReadFile(archives[0])
	if err != nil {
		t.Fatalf("ReadFile(archive) error = %v", err)
	}
	if string(archiveContent) != "before-startup\n" {
		t.Fatalf("startup archive content = %q", string(archiveContent))
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(active) error = %v", err)
	}
	if info.Size() != 0 {
		t.Fatalf("active file should be empty after startup rollover, size=%d", info.Size())
	}
}

func TestRollingFileAppender_whenRolloverOccurs_shouldPreserveLayoutLifecycle(t *testing.T) {
	now := FixedTestTime()
	path := filepath.Join(t.TempDir(), "lifecycle.log")
	appender, err := NewRollingFileAppender(path,
		WithRollingFileLayout(testLifecycleLayout{}),
		WithRollingMaxSize(12),
		WithRollingMaxBackups(10),
		WithRollingClock(func() time.Time { return now }),
	)
	if err != nil {
		t.Fatalf("NewRollingFileAppender() error = %v", err)
	}
	if err := appender.Append(context.Background(), TestEvent("first", now)); err != nil {
		t.Fatalf("Append(first) error = %v", err)
	}
	if err := appender.Append(context.Background(), TestEvent("second", now)); err != nil {
		t.Fatalf("Append(second) error = %v", err)
	}
	if err := appender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	archives, err := filepath.Glob(path + ".*")
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(archives) != 1 {
		t.Fatalf("expected one archive, got %d: %v", len(archives), archives)
	}
	if content := ReadTextFile(t, archives[0]); content != "[\nfirst\n]\n" {
		t.Fatalf("archive lifecycle content = %q", content)
	}
	if content := ReadTextFile(t, path); content != "[\nsecond\n]\n" {
		t.Fatalf("active lifecycle content = %q", content)
	}
}

func TestRollingFileAppender_whenAppendDisabled_shouldTruncateActiveFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "truncate.log")
	if err := os.WriteFile(path, []byte("old content\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	appender, err := NewRollingFileAppender(path,
		WithRollingFileLayout(TextLayout{}),
		WithRollingFileAppend(false),
		WithRollingMaxSize(1024),
	)
	if err != nil {
		t.Fatalf("NewRollingFileAppender() error = %v", err)
	}
	if err := appender.Append(
		context.Background(),
		TestEvent("new content", FixedTestTime()),
	); err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	if err := appender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	content := ReadTextFile(t, path)
	if strings.Contains(content, "old content") || !strings.Contains(content, "new content") {
		t.Fatalf("active file content = %q, want truncated old content and new event", content)
	}
}

func TestRollingFileAppender_whenCreateOnDemandClosedBeforeAppend_shouldNotOpenFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "logs", "lazy-complete.json")
	appender, err := NewRollingFileAppender(path,
		WithRollingFileLayout(NewJSONLayout(LayoutOptions{Complete: true})),
		WithRollingFileCreateOnDemand(true),
		WithRollingMaxSize(1024),
	)
	if err != nil {
		t.Fatalf("NewRollingFileAppender() error = %v", err)
	}
	if err := appender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("active file should not be created before first append, stat error = %v", err)
	}
}

func TestRollingFileAppender_whenPolicyInvalid_shouldReject(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid.log")
	_, err := NewRollingFileAppender(path, WithRollingMaxSize(-1))
	if err == nil {
		t.Fatalf("NewRollingFileAppender() should reject negative max size")
	}
	_, err = NewRollingFileAppender(
		path,
		WithRollingMaxSize(0),
		WithRollingInterval(0),
		WithRolloverOnStartup(false),
	)
	if err == nil {
		t.Fatalf("NewRollingFileAppender() should reject empty rolling policy")
	}
}

func (testLifecycleLayout) AppendFooter(buf *bytes.Buffer) error {
	buf.WriteString("]\n")
	return nil
}
