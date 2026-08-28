package integration

import (
	"context"
	"io/fs"
	"path/filepath"
	"sort"
	"testing"
)

func TestDocsExamples_whenLoaded_shouldBuildOptions(t *testing.T) {
	root := filepath.Join("..", "..")
	examples, err := docsConfigExamples(filepath.Join(root, "docs", "examples"))
	if err != nil {
		t.Fatalf("docsConfigExamples() error = %v", err)
	}
	if len(examples) == 0 {
		t.Fatalf("docs config examples should not be empty")
	}
	for _, example := range examples {
		t.Run(filepath.Base(example), func(t *testing.T) {
			absPath, err := filepath.Abs(example)
			if err != nil {
				t.Fatalf("filepath.Abs(%q) error = %v", example, err)
			}
			t.Chdir(t.TempDir())
			options, _, err := LoadOptions(context.Background(), WithConfigPath(absPath))
			if err != nil {
				t.Fatalf("LoadOptions() error = %v", err)
			}
			if err := closeAppenderList(options.Appenders); err != nil {
				t.Fatalf("closeAppenderList() error = %v", err)
			}
		})
	}
}

func docsConfigExamples(root string) ([]string, error) {
	var examples []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !isConfigExample(entry.Name()) {
			return nil
		}
		examples = append(examples, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(examples)
	return examples, nil
}

func isConfigExample(name string) bool {
	switch filepath.Ext(name) {
	case ".yml", ".yaml", ".json", ".toml", ".xml", ".properties":
		return true
	default:
		return false
	}
}
