package goarklog

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRollingFileAppender_whenRestarted_shouldContinueArchiveIndex(t *testing.T) {
	now := fixedTestTime()
	path := filepath.Join(t.TempDir(), "restart.log")
	appender := newSmallRollingAppender(t, path, now)
	for index := 0; index < 3; index++ {
		if err := appender.Append(context.Background(), testEvent("before-restart-"+strings.Repeat("x", 80), now)); err != nil {
			t.Fatalf("Append(before %d) error = %v", index, err)
		}
	}
	if err := appender.Close(); err != nil {
		t.Fatalf("Close(before) error = %v", err)
	}

	appender = newSmallRollingAppender(t, path, now)
	for index := 0; index < 1; index++ {
		if err := appender.Append(context.Background(), testEvent("after-restart-"+strings.Repeat("x", 80), now)); err != nil {
			t.Fatalf("Append(after %d) error = %v", index, err)
		}
	}
	if err := appender.Close(); err != nil {
		t.Fatalf("Close(after) error = %v", err)
	}

	archives, err := filepath.Glob(path + ".*")
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(archives) != 3 {
		t.Fatalf("archive count = %d, want 3: %v", len(archives), archives)
	}
	last := filepath.Base(archives[len(archives)-1])
	if !strings.HasSuffix(last, ".002") {
		t.Fatalf("restart should continue archive index, archives=%v", archives)
	}
}

func TestAsyncAppender_whenClosedTwice_shouldDrainOnceAndRejectAppend(t *testing.T) {
	delegate := newRecordingAppender("delegate")
	appender, err := NewAsyncAppender([]Appender{delegate},
		WithAsyncQueueSize(4),
		WithAsyncCloseAppenders(true),
	)
	if err != nil {
		t.Fatalf("NewAsyncAppender() error = %v", err)
	}
	for index := 0; index < 4; index++ {
		if err := appender.Append(context.Background(), testEvent("event", fixedTestTime())); err != nil {
			t.Fatalf("Append(%d) error = %v", index, err)
		}
	}
	if err := appender.Close(); err != nil {
		t.Fatalf("Close(first) error = %v", err)
	}
	if err := appender.Close(); err != nil {
		t.Fatalf("Close(second) error = %v", err)
	}
	if err := appender.Append(context.Background(), testEvent("after-close", fixedTestTime())); err == nil {
		t.Fatalf("Append() after Close should fail")
	}
	if got := len(delegate.Events()); got != 4 {
		t.Fatalf("delegate event count = %d, want 4", got)
	}
	if delegate.CloseCount() != 1 {
		t.Fatalf("delegate CloseCount() = %d, want 1", delegate.CloseCount())
	}
}

func TestConfigReloader_whenReloadFails_shouldKeepOldRuntimeConfig(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "logs", "reload-failure.log")
	configPath := filepath.Join(dir, "goark-log.yml")
	writeFileConfig(t, configPath, logPath, "info")

	logger, handler, _, err := NewConfigured(ctx, WithConfigPath(configPath))
	if err != nil {
		t.Fatalf("NewConfigured() error = %v", err)
	}
	reloader, err := NewConfigReloader(handler, WithConfigPath(configPath))
	if err != nil {
		t.Fatalf("NewConfigReloader() error = %v", err)
	}
	writeConfig(t, configPath, `
appenders:
  file:
    type: file
    fileName: `+filepath.ToSlash(filepath.Join(dir, "logs", "invalid.log"))+`
root:
  level: info
  appenderRefs: [missing]
`)
	if _, err := reloader.Reload(ctx); err == nil {
		t.Fatalf("Reload() should reject invalid replacement config")
	}
	logger.Info("still visible after failed reload")
	if err := handler.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile(old log) error = %v", err)
	}
	if !strings.Contains(string(content), "still visible after failed reload") {
		t.Fatalf("old runtime config was not kept, content=%q", string(content))
	}
}

func TestConfigReloader_whenReloadAndLoggingConcurrent_shouldBeRaceSafe(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "logs", "race.log")
	configPath := filepath.Join(dir, "goark-log.yml")
	writeFileConfig(t, configPath, logPath, "debug")
	logger, handler, _, err := NewConfigured(ctx, WithConfigPath(configPath))
	if err != nil {
		t.Fatalf("NewConfigured() error = %v", err)
	}
	reloader, err := NewConfigReloader(handler, WithConfigPath(configPath))
	if err != nil {
		t.Fatalf("NewConfigReloader() error = %v", err)
	}

	var (
		wait     sync.WaitGroup
		configMu sync.Mutex
	)
	for worker := 0; worker < 8; worker++ {
		wait.Add(1)
		go func(worker int) {
			defer wait.Done()
			for index := 0; index < 64; index++ {
				logger.Info("event", "worker", worker, "index", index)
			}
		}(worker)
	}
	for index := 0; index < 8; index++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			level := "info"
			if index%2 == 0 {
				level = "debug"
			}
			configMu.Lock()
			err := writeFileConfigNoFatal(configPath, logPath, level)
			if err == nil {
				_, err = reloader.Reload(ctx)
			}
			configMu.Unlock()
			if err != nil {
				t.Errorf("reload worker %d error = %v", index, err)
			}
		}(index)
	}
	wait.Wait()
	if err := handler.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func newSmallRollingAppender(t *testing.T, path string, now time.Time) *RollingFileAppender {
	t.Helper()
	appender, err := NewRollingFileAppender(path,
		WithRollingFileLayout(TextLayout{}),
		WithRollingMaxSize(120),
		WithRollingMaxBackups(10),
		withRollingClock(func() time.Time { return now }),
	)
	if err != nil {
		t.Fatalf("NewRollingFileAppender() error = %v", err)
	}
	return appender
}

func writeFileConfigNoFatal(configPath string, logPath string, level string) error {
	content := `
appenders:
  file:
    type: file
    fileName: "` + filepath.ToSlash(logPath) + `"
    layout:
      type: text
root:
  level: ` + level + `
  appenderRefs: [file]
`
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(configPath, []byte(strings.TrimSpace(content)+"\n"), 0o644)
}
