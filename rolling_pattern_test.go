package goarklog

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRollingFileAppender_whenFilePatternConfigured_shouldArchiveWithDateAndIndex(t *testing.T) {
	now := fixedTestTime()
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	pattern := filepath.Join(dir, "archive", "app-%d{yyyyMMdd-HHmmss}-%03i.log.gz")
	appender, err := NewRollingFileAppender(path,
		WithRollingFileLayout(TextLayout{}),
		WithRollingMaxSize(120),
		WithRollingMaxBackups(10),
		WithRollingFilePattern(pattern),
		withRollingClock(func() time.Time { return now }),
	)
	if err != nil {
		t.Fatalf("NewRollingFileAppender() error = %v", err)
	}
	for index := 0; index < 3; index++ {
		if err := appender.Append(context.Background(), testEvent("pattern-"+strings.Repeat("x", 80), now)); err != nil {
			t.Fatalf("Append(%d) error = %v", index, err)
		}
	}
	if err := appender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	archives, err := filepath.Glob(filepath.Join(dir, "archive", "*.gz"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(archives) != 2 {
		t.Fatalf("archive count = %d, want 2: %v", len(archives), archives)
	}
	if !strings.HasSuffix(filepath.Base(archives[0]), "app-20260825-101530-000.log.gz") ||
		!strings.HasSuffix(filepath.Base(archives[1]), "app-20260825-101530-001.log.gz") {
		t.Fatalf("archive names are wrong: %v", archives)
	}
	content := readGzipFile(t, archives[0])
	if !strings.Contains(content, "pattern-") {
		t.Fatalf("archive content is wrong: %q", content)
	}
}

func TestRollingFileAppender_whenPatternRestarts_shouldContinuePatternIndex(t *testing.T) {
	now := fixedTestTime()
	dir := t.TempDir()
	path := filepath.Join(dir, "restart.log")
	pattern := filepath.Join(dir, "archive", "restart-%d{yyyyMMdd}-%i.log")
	appender, err := NewRollingFileAppender(path,
		WithRollingFileLayout(TextLayout{}),
		WithRollingMaxSize(120),
		WithRollingMaxBackups(10),
		WithRollingFilePattern(pattern),
		withRollingClock(func() time.Time { return now }),
	)
	if err != nil {
		t.Fatalf("NewRollingFileAppender(first) error = %v", err)
	}
	for index := 0; index < 2; index++ {
		if err := appender.Append(context.Background(), testEvent("before-"+strings.Repeat("x", 80), now)); err != nil {
			t.Fatalf("Append(before %d) error = %v", index, err)
		}
	}
	if err := appender.Close(); err != nil {
		t.Fatalf("Close(first) error = %v", err)
	}

	appender, err = NewRollingFileAppender(path,
		WithRollingFileLayout(TextLayout{}),
		WithRollingMaxSize(120),
		WithRollingMaxBackups(10),
		WithRollingFilePattern(pattern),
		withRollingClock(func() time.Time { return now }),
	)
	if err != nil {
		t.Fatalf("NewRollingFileAppender(second) error = %v", err)
	}
	if err := appender.Append(context.Background(), testEvent("after-"+strings.Repeat("x", 80), now)); err != nil {
		t.Fatalf("Append(after) error = %v", err)
	}
	if err := appender.Close(); err != nil {
		t.Fatalf("Close(second) error = %v", err)
	}

	archives, err := filepath.Glob(filepath.Join(dir, "archive", "restart-*"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(archives) != 2 || !strings.HasSuffix(filepath.Base(archives[1]), "-1.log") {
		t.Fatalf("restart should continue pattern index, archives=%v", archives)
	}
}

func TestNewConfigured_whenYamlRollingFilePatternUsed_shouldArchiveWithPattern(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "app.log")
	configPath := filepath.Join(dir, "goark-log.yml")
	writeConfig(t, configPath, `
appenders:
  rolling:
    type: rollingFile
    fileName: "`+filepath.ToSlash(logPath)+`"
    layout:
      type: text
    rolling:
      filePattern: "`+filepath.ToSlash(filepath.Join(dir, "archive", "yaml-%d{yyyyMMdd}-%i.log.gz"))+`"
      maxSize: 120
      maxBackups: 1
root:
  level: info
  appenderRefs: [rolling]
`)
	logger, handler, _, err := NewConfigured(context.Background(), WithConfigPath(configPath))
	if err != nil {
		t.Fatalf("NewConfigured() error = %v", err)
	}
	for index := 0; index < 3; index++ {
		logger.Info("yaml pattern " + strings.Repeat("x", 80))
	}
	if err := handler.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	archives, err := filepath.Glob(filepath.Join(dir, "archive", "*.gz"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(archives) != 1 {
		t.Fatalf("retained archive count = %d, want 1: %v", len(archives), archives)
	}
	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("active log should exist: %v", err)
	}
}

func TestRollingFileAppender_whenPatternMissingIndexWithSizePolicy_shouldReject(t *testing.T) {
	_, err := NewRollingFileAppender(filepath.Join(t.TempDir(), "app.log"),
		WithRollingMaxSize(1024),
		WithRollingFilePattern("archive-%d{yyyyMMdd}.log"),
	)
	if err == nil {
		t.Fatalf("NewRollingFileAppender() should reject size policy pattern without %%i")
	}
}
