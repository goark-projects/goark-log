package jsontemplate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadFile_whenLocalFileProvided_shouldReadTemplate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "event.json")
	if err := os.WriteFile(path, []byte(`{"message":{"$resolver":"message"}}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	template, err := ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(template, "$resolver") {
		t.Fatalf("template = %q, want resolver content", template)
	}
}

func TestReadFile_whenUnsupportedSchemeProvided_shouldReject(t *testing.T) {
	_, err := ReadFile("https://example.invalid/template.json")
	if err == nil || !strings.Contains(err.Error(), "scheme") {
		t.Fatalf("ReadFile(https) error = %v, want scheme rejection", err)
	}
}
