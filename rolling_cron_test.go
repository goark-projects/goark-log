package goarklog

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestParseCronSchedule_whenExpressionValid_shouldFindNextTime(t *testing.T) {
	schedule, err := parseCronSchedule("0/15 10-11 9 * JAN MON-FRI")
	if err != nil {
		t.Fatalf("parseCronSchedule() error = %v", err)
	}
	next, ok := schedule.next(time.Date(2026, time.January, 5, 8, 59, 59, 0, time.UTC))
	if !ok {
		t.Fatalf("next() ok = false")
	}
	want := time.Date(2026, time.January, 5, 9, 10, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Fatalf("next = %s, want %s", next, want)
	}
}

func TestParseCronSchedule_whenExpressionInvalid_shouldReject(t *testing.T) {
	if _, err := parseCronSchedule("0 0 0 * * * 2026"); err == nil {
		t.Fatalf("parseCronSchedule() should reject fixed year")
	}
	if _, err := parseCronSchedule("0 0 24 * * *"); err == nil {
		t.Fatalf("parseCronSchedule() should reject invalid hour")
	}
}

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

func TestRollingDeleteAction_whenAccumulatedLimitsSet_shouldKeepNewestWithinLimits(t *testing.T) {
	now := fixedTestTime()
	dir := t.TempDir()
	files := []struct {
		name string
		size int
		age  time.Duration
	}{
		{name: "oldest.log.gz", size: 40, age: 4 * time.Hour},
		{name: "old.log.gz", size: 40, age: 3 * time.Hour},
		{name: "new.log.gz", size: 40, age: 2 * time.Hour},
		{name: "newest.log.gz", size: 40, age: time.Hour},
	}
	for _, file := range files {
		path := filepath.Join(dir, file.name)
		if err := os.WriteFile(path, []byte(strings.Repeat("x", file.size)), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", file.name, err)
		}
		modTime := now.Add(-file.age)
		if err := os.Chtimes(path, modTime, modTime); err != nil {
			t.Fatalf("Chtimes(%s) error = %v", file.name, err)
		}
	}

	err := deleteArchivesByAction(now, RollingDeleteAction{
		BasePath: dir,
		Glob:     "*.log.gz",
		MaxCount: 3,
		MaxSize:  90,
	})
	if err != nil {
		t.Fatalf("deleteArchivesByAction() error = %v", err)
	}
	remaining := existingBaseNames(t, dir)
	if !reflect.DeepEqual(remaining, []string{"new.log.gz", "newest.log.gz"}) {
		t.Fatalf("remaining files = %v", remaining)
	}
}

func existingBaseNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	return names
}
