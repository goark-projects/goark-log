package integration

import (
	"bytes"
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

func TestRollingFileAppender_whenAsyncDeleteActionConfigured_shouldDeleteExpiredArchivesOnClose(t *testing.T) {
	now := fixedTestTime()
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
		withRollingClock(func() time.Time { return now }),
	)
	if err != nil {
		t.Fatalf("NewRollingFileAppender() error = %v", err)
	}
	for index := 0; index < 2; index++ {
		if err := appender.Append(context.Background(), testEvent("async-delete-"+strings.Repeat("x", 80), now)); err != nil {
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
	if err := appender.Append(context.Background(), testEvent("new content", fixedTestTime())); err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	if err := appender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	content := readTextFile(t, path)
	if strings.Contains(content, "old content") || !strings.Contains(content, "new content") {
		t.Fatalf("active file content = %q, want truncated old content and new event", content)
	}
}

func TestRollingFileAppender_whenCreateOnDemandEnabled_shouldDelayActiveFileCreation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "logs", "lazy.log")
	appender, err := NewRollingFileAppender(path,
		WithRollingFileLayout(TextLayout{}),
		WithRollingFileCreateOnDemand(true),
		WithRollingMaxSize(1024),
	)
	if err != nil {
		t.Fatalf("NewRollingFileAppender() error = %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("active file should not be created before first append, stat error = %v", err)
	}
	if err := appender.Append(context.Background(), testEvent("lazy content", fixedTestTime())); err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	if err := appender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if content := readTextFile(t, path); !strings.Contains(content, "lazy content") {
		t.Fatalf("lazy active file content = %q", content)
	}
}

func TestRollingFileAppender_whenRolloverOccurs_shouldPreserveLayoutLifecycle(t *testing.T) {
	now := fixedTestTime()
	path := filepath.Join(t.TempDir(), "lifecycle.log")
	appender, err := NewRollingFileAppender(path,
		WithRollingFileLayout(testLifecycleLayout{}),
		WithRollingMaxSize(12),
		WithRollingMaxBackups(10),
		withRollingClock(func() time.Time { return now }),
	)
	if err != nil {
		t.Fatalf("NewRollingFileAppender() error = %v", err)
	}
	if err := appender.Append(context.Background(), testEvent("first", now)); err != nil {
		t.Fatalf("Append(first) error = %v", err)
	}
	if err := appender.Append(context.Background(), testEvent("second", now)); err != nil {
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
	if content := readTextFile(t, archives[0]); content != "[\nfirst\n]\n" {
		t.Fatalf("archive lifecycle content = %q", content)
	}
	if content := readTextFile(t, path); content != "[\nsecond\n]\n" {
		t.Fatalf("active lifecycle content = %q", content)
	}
}

func TestRollingFileAppender_whenCompleteJSONLayoutRolls_shouldKeepEachStreamValid(t *testing.T) {
	now := fixedTestTime()
	dir := t.TempDir()
	path := filepath.Join(dir, "complete-roll.json")
	appender, err := NewRollingFileAppender(path,
		WithRollingFileLayout(NewJSONLayout(LayoutOptions{Compact: true, Complete: true})),
		WithRollingMaxSize(180),
		WithRollingMaxBackups(10),
		withRollingClock(func() time.Time { return now }),
	)
	if err != nil {
		t.Fatalf("NewRollingFileAppender() error = %v", err)
	}
	if err := appender.Append(context.Background(), testEvent("roll-first-"+strings.Repeat("x", 20), now)); err != nil {
		t.Fatalf("Append(first) error = %v", err)
	}
	if err := appender.Append(context.Background(), testEvent("roll-second-"+strings.Repeat("x", 20), now)); err != nil {
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
	assertCompleteJSONMessages(t, archives[0], "roll-first-"+strings.Repeat("x", 20))
	assertCompleteJSONMessages(t, path, "roll-second-"+strings.Repeat("x", 20))
}

func TestNewConfigured_whenRollingFileCreateOnDemandAndAppendDisabledConfigured_shouldDelayAndTruncate(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "logs", "configured.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(logPath, []byte("old configured\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	configPath := filepath.Join(dir, "goark-log.properties")
	writeConfig(t, configPath, `
appender.rolling.type=rollingFile
appender.rolling.fileName=`+filepath.ToSlash(logPath)+`
appender.rolling.append=false
appender.rolling.createOnDemand=true
appender.rolling.filePermissions=rw-------
appender.rolling.rolling.maxSize=1MB
appender.rolling.layout.type=text
rootLogger.level=info
rootLogger.appenderRefs=rolling
`)
	logger, handler, _, err := NewConfigured(context.Background(), WithConfigPath(configPath))
	if err != nil {
		t.Fatalf("NewConfigured() error = %v", err)
	}
	if content := readTextFile(t, logPath); content != "old configured\n" {
		t.Fatalf("createOnDemand should not touch existing file before append, got %q", content)
	}
	logger.Info("configured new")
	if err := handler.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	content := readTextFile(t, logPath)
	if strings.Contains(content, "old configured") || !strings.Contains(content, "configured new") {
		t.Fatalf("configured rolling content = %q, want truncated old content and new event", content)
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

type testLifecycleLayout struct{}

func (testLifecycleLayout) Format(buf *bytes.Buffer, event Event) error {
	buf.WriteString(event.Message)
	buf.WriteByte('\n')
	return nil
}

func (testLifecycleLayout) AppendHeader(buf *bytes.Buffer) error {
	buf.WriteString("[\n")
	return nil
}

func (testLifecycleLayout) AppendFooter(buf *bytes.Buffer) error {
	buf.WriteString("]\n")
	return nil
}
