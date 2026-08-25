package goarklog

import (
	"compress/gzip"
	"context"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestRollingFileAppender_whenSizeExceeded_shouldArchiveCompressAndRetain(t *testing.T) {
	now := fixedTestTime()
	path := filepath.Join(t.TempDir(), "app.log")
	appender, err := NewRollingFileAppender(path,
		WithRollingFileLayout(TextLayout{}),
		WithRollingMaxSize(120),
		WithRollingMaxBackups(1),
		WithRollingGzip(true),
		withRollingClock(func() time.Time { return now }),
	)
	if err != nil {
		t.Fatalf("NewRollingFileAppender() error = %v", err)
	}
	for index := 0; index < 4; index++ {
		message := "event-" + strconv.Itoa(index) + "-" + strings.Repeat("x", 80)
		if err := appender.Append(context.Background(), testEvent(message, now)); err != nil {
			t.Fatalf("Append(%d) error = %v", index, err)
		}
	}
	if err := appender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	archives, err := filepath.Glob(path + ".*.gz")
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(archives) != 1 {
		t.Fatalf("expected one retained gzip archive, got %d: %v", len(archives), archives)
	}
	archiveContent := readGzipFile(t, archives[0])
	activeContent, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(active) error = %v", err)
	}
	if !strings.Contains(archiveContent, "event-2-") {
		t.Fatalf("archive should keep latest rolled content, archives=%v archive=%q active=%q", archives, archiveContent, string(activeContent))
	}
	if !strings.Contains(string(activeContent), "event-3-") {
		t.Fatalf("active file should contain latest event, got %q", string(activeContent))
	}
	uncompressed, err := filepath.Glob(path + ".*")
	if err != nil {
		t.Fatalf("Glob(uncompressed) error = %v", err)
	}
	for _, candidate := range uncompressed {
		if !strings.HasSuffix(candidate, ".gz") {
			t.Fatalf("expected no uncompressed archive, found %q", candidate)
		}
	}
}

func TestRollingFileAppender_whenIntervalElapsed_shouldRollByEventTime(t *testing.T) {
	start := time.Date(2026, 8, 25, 10, 0, 0, 0, time.FixedZone("CST", 8*3600))
	path := filepath.Join(t.TempDir(), "time.log")
	appender, err := NewRollingFileAppender(path,
		WithRollingFileLayout(TextLayout{}),
		WithRollingMaxSize(0),
		WithRollingInterval(time.Hour),
		WithRollingMaxBackups(10),
		withRollingClock(func() time.Time { return start }),
	)
	if err != nil {
		t.Fatalf("NewRollingFileAppender() error = %v", err)
	}
	if err := appender.Append(context.Background(), testEvent("first", start.Add(10*time.Minute))); err != nil {
		t.Fatalf("Append(first) error = %v", err)
	}
	if err := appender.Append(context.Background(), testEvent("second", start.Add(61*time.Minute))); err != nil {
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
		t.Fatalf("expected one time archive, got %d: %v", len(archives), archives)
	}
	archiveContent, err := os.ReadFile(archives[0])
	if err != nil {
		t.Fatalf("ReadFile(archive) error = %v", err)
	}
	activeContent, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(active) error = %v", err)
	}
	if !strings.Contains(string(archiveContent), "msg=first") || !strings.Contains(string(activeContent), "msg=second") {
		t.Fatalf("time rolling split is wrong, archive=%q active=%q", string(archiveContent), string(activeContent))
	}
}

func TestRollingFileAppender_whenStartupEnabled_shouldArchiveExistingFile(t *testing.T) {
	now := fixedTestTime()
	path := filepath.Join(t.TempDir(), "startup.log")
	if err := os.WriteFile(path, []byte("before-startup\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	appender, err := NewRollingFileAppender(path,
		WithRollingMaxSize(0),
		WithRolloverOnStartup(true),
		WithRollingMaxBackups(10),
		withRollingClock(func() time.Time { return now }),
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

func TestRollingFileAppender_whenPolicyInvalid_shouldReject(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid.log")
	_, err := NewRollingFileAppender(path, WithRollingMaxSize(-1))
	if err == nil {
		t.Fatalf("NewRollingFileAppender() should reject negative max size")
	}
	_, err = NewRollingFileAppender(path, WithRollingMaxSize(0), WithRollingInterval(0), WithRolloverOnStartup(false))
	if err == nil {
		t.Fatalf("NewRollingFileAppender() should reject empty rolling policy")
	}
}

func readGzipFile(t *testing.T, path string) string {
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
	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll(%s) error = %v", path, err)
	}
	return string(content)
}
