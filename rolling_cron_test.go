package goarklog

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRollingFileAppender_whenCronScheduleReached_shouldRoll(t *testing.T) {
	start := time.Date(2026, 8, 25, 10, 15, 30, 0, time.FixedZone("CST", 8*3600))
	path := filepath.Join(t.TempDir(), "cron.log")
	appender, err := NewRollingFileAppender(path,
		WithRollingFileLayout(TextLayout{}),
		WithRollingMaxSize(0),
		WithRollingCronSchedule("31 15 10 * * *"),
		WithRollingMaxBackups(10),
		withRollingClock(func() time.Time { return start }),
	)
	if err != nil {
		t.Fatalf("NewRollingFileAppender() error = %v", err)
	}
	if err := appender.Append(context.Background(), testEvent("before-cron", start)); err != nil {
		t.Fatalf("Append(before) error = %v", err)
	}
	if err := appender.Append(context.Background(), testEvent("after-cron", start.Add(time.Second))); err != nil {
		t.Fatalf("Append(after) error = %v", err)
	}
	if err := appender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	archives, err := filepath.Glob(path + ".*")
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(archives) != 1 {
		t.Fatalf("archive count = %d, want 1: %v", len(archives), archives)
	}
	archiveContent, err := os.ReadFile(archives[0])
	if err != nil {
		t.Fatalf("ReadFile(archive) error = %v", err)
	}
	activeContent, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(active) error = %v", err)
	}
	if !strings.Contains(string(archiveContent), "before-cron") || !strings.Contains(string(activeContent), "after-cron") {
		t.Fatalf("cron rolling split is wrong, archive=%q active=%q", string(archiveContent), string(activeContent))
	}
}
