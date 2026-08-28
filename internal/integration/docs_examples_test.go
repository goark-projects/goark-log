package integration

import (
	"context"
	"path/filepath"
	"testing"
)

func TestDocsExamples_whenLoaded_shouldBuildOptions(t *testing.T) {
	examples := []string{
		"console.yml",
		"json-stdout.yml",
		"production-rolling.yml",
		"split-audit.yml",
		"async-appender.yml",
		"rewrite-routing.yml",
		"goark-log.properties",
		"log4j2-style.xml",
	}
	for _, example := range examples {
		t.Run(example, func(t *testing.T) {
			root := filepath.Join("..", "..")
			path := filepath.Join(root, "docs", "examples", example)
			absPath, err := filepath.Abs(path)
			if err != nil {
				t.Fatalf("filepath.Abs() error = %v", err)
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
