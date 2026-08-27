package rolling

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompressFile_whenGzipCreateFails_shouldKeepOriginalArchive(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "app.log.20260825-101530.123.000")
	if err := os.WriteFile(source, []byte("archive-content\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(source) error = %v", err)
	}
	if err := os.WriteFile(source+".gz", []byte("exists"), 0o644); err != nil {
		t.Fatalf("WriteFile(gz) error = %v", err)
	}

	if _, err := CompressFile(source); err == nil {
		t.Fatalf("CompressFile() should fail when gzip archive already exists")
	}
	content, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("ReadFile(source) error = %v", err)
	}
	if string(content) != "archive-content\n" {
		t.Fatalf("source archive content changed: %q", string(content))
	}
}
