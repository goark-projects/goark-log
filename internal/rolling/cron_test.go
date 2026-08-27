package rolling

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestParseCronSchedule_whenExpressionValid_shouldFindNextTime(t *testing.T) {
	schedule, err := ParseCronSchedule("0/15 10-11 9 * JAN MON-FRI")
	if err != nil {
		t.Fatalf("ParseCronSchedule() error = %v", err)
	}
	next, ok := schedule.Next(time.Date(2026, time.January, 5, 8, 59, 59, 0, time.UTC))
	if !ok {
		t.Fatalf("Next() ok = false")
	}
	want := time.Date(2026, time.January, 5, 9, 10, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Fatalf("Next() = %s, want %s", next, want)
	}
}

func TestParseCronSchedule_whenExpressionInvalid_shouldReject(t *testing.T) {
	if _, err := ParseCronSchedule("0 0 0 * * * 2026"); err == nil {
		t.Fatalf("ParseCronSchedule() should reject fixed year")
	}
	if _, err := ParseCronSchedule("0 0 24 * * *"); err == nil {
		t.Fatalf("ParseCronSchedule() should reject invalid hour")
	}
}

func TestDeleteArchivesByAction_whenAccumulatedLimitsSet_shouldKeepNewestWithinLimits(t *testing.T) {
	now := time.Date(2026, 8, 25, 10, 15, 30, 0, time.FixedZone("CST", 8*3600))
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

	err := DeleteArchivesByAction(now, DeleteAction{
		BasePath: dir,
		Glob:     "*.log.gz",
		MaxCount: 3,
		MaxSize:  90,
	})
	if err != nil {
		t.Fatalf("DeleteArchivesByAction() error = %v", err)
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
