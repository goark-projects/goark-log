package integration

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	. "goark.dev/log/internal/testsupport"
)

func TestRollingFileAppender_whenSizeExceeded_shouldArchiveCompressAndRetain(t *testing.T) {
	now := FixedTestTime()
	path := filepath.Join(t.TempDir(), "app.log")
	appender, err := NewRollingFileAppender(path,
		WithRollingFileLayout(TextLayout{}),
		WithRollingMaxSize(120),
		WithRollingMaxBackups(1),
		WithRollingGzip(true),
		WithRollingClock(func() time.Time { return now }),
	)
	if err != nil {
		t.Fatalf("NewRollingFileAppender() error = %v", err)
	}
	for index := 0; index < 4; index++ {
		message := "event-" + strconv.Itoa(index) + "-" + strings.Repeat("x", 80)
		if err := appender.Append(context.Background(), TestEvent(message, now)); err != nil {
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
	archiveContent := ReadGzipFile(t, archives[0])
	activeContent, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(active) error = %v", err)
	}
	if !strings.Contains(archiveContent, "event-2-") {
		t.Fatalf(
			"archive should keep latest rolled content, archives=%v archive=%q active=%q",
			archives,
			archiveContent,
			string(activeContent),
		)
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
		WithRollingClock(func() time.Time { return start }),
	)
	if err != nil {
		t.Fatalf("NewRollingFileAppender() error = %v", err)
	}
	if err := appender.Append(
		context.Background(),
		TestEvent("first", start.Add(10*time.Minute)),
	); err != nil {
		t.Fatalf("Append(first) error = %v", err)
	}
	if err := appender.Append(
		context.Background(),
		TestEvent("second", start.Add(61*time.Minute)),
	); err != nil {
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
	if !strings.Contains(string(archiveContent), "msg=first") ||
		!strings.Contains(string(activeContent), "msg=second") {
		t.Fatalf(
			"time rolling split is wrong, archive=%q active=%q",
			string(archiveContent),
			string(activeContent),
		)
	}
}

func TestNewConfigured_whenRollingCreateOnDemandAndAppendDisabled_shouldDelayAndTruncate(
	t *testing.T,
) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "logs", "configured.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(logPath, []byte("old configured\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	configPath := filepath.Join(dir, "goark-log.properties")
	WriteConfig(t, configPath, `
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
	if content := ReadTextFile(t, logPath); content != "old configured\n" {
		t.Fatalf("createOnDemand should not touch existing file before append, got %q", content)
	}
	logger.Info("configured new")
	if err := handler.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	content := ReadTextFile(t, logPath)
	if strings.Contains(content, "old configured") || !strings.Contains(content, "configured new") {
		t.Fatalf(
			"configured rolling content = %q, want truncated old content and new event",
			content,
		)
	}
}

func TestRollingFileAppender_whenCompleteJSONLayoutRolls_shouldKeepEachStreamValid(t *testing.T) {
	now := FixedTestTime()
	dir := t.TempDir()
	path := filepath.Join(dir, "complete-roll.json")
	appender, err := NewRollingFileAppender(path,
		WithRollingFileLayout(NewJSONLayout(LayoutOptions{Compact: true, Complete: true})),
		WithRollingMaxSize(180),
		WithRollingMaxBackups(10),
		WithRollingClock(func() time.Time { return now }),
	)
	if err != nil {
		t.Fatalf("NewRollingFileAppender() error = %v", err)
	}
	if err := appender.Append(
		context.Background(),
		TestEvent("roll-first-"+strings.Repeat("x", 20), now),
	); err != nil {
		t.Fatalf("Append(first) error = %v", err)
	}
	if err := appender.Append(
		context.Background(),
		TestEvent("roll-second-"+strings.Repeat("x", 20), now),
	); err != nil {
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
	AssertCompleteJSONMessages(t, archives[0], "roll-first-"+strings.Repeat("x", 20))
	AssertCompleteJSONMessages(t, path, "roll-second-"+strings.Repeat("x", 20))
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
	if err := appender.Append(
		context.Background(),
		TestEvent("lazy content", FixedTestTime()),
	); err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	if err := appender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if content := ReadTextFile(t, path); !strings.Contains(content, "lazy content") {
		t.Fatalf("lazy active file content = %q", content)
	}
}

func (testLifecycleLayout) Format(buf *bytes.Buffer, event Event) error {
	buf.WriteString(event.Message)
	buf.WriteByte('\n')
	return nil
}

func (testLifecycleLayout) AppendHeader(buf *bytes.Buffer) error {
	buf.WriteString("[\n")
	return nil
}

type testLifecycleLayout struct{}
