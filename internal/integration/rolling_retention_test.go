package integration

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRollingFileAppender_whenCleanHistoryOnStartEnabled_shouldApplyCountAndSizeLimits(t *testing.T) {
	directory := t.TempDir()
	for index, name := range []string{"app-001.log", "app-002.log", "app-003.log"} {
		path := filepath.Join(directory, name)
		if err := os.WriteFile(path, []byte("123456"), 0o644); err != nil {
			t.Fatalf("write archive %d: %v", index, err)
		}
	}

	appender, err := NewRollingFileAppender(
		filepath.Join(directory, "app.log"),
		WithRollingFilePattern(filepath.Join(directory, "app-%03i.log")),
		WithRollingMaxSize(1024),
		WithRollingMaxBackups(2),
		WithRollingTotalSizeCap(10),
		WithRollingCleanHistoryOnStart(true),
	)
	if err != nil {
		t.Fatalf("NewRollingFileAppender() error = %v", err)
	}
	defer appender.Close()

	matches, err := filepath.Glob(filepath.Join(directory, "app-*.log"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(matches) != 1 || filepath.Base(matches[0]) != "app-003.log" {
		t.Fatalf("remaining archives = %#v, want newest archive", matches)
	}
}

func TestRollingFileAppender_whenTotalSizeCapIsNegative_shouldReject(t *testing.T) {
	_, err := NewRollingFileAppender(
		filepath.Join(t.TempDir(), "app.log"),
		WithRollingMaxSize(1024),
		WithRollingTotalSizeCap(-1),
	)
	if err == nil {
		t.Fatal("negative total size cap should fail")
	}
}
