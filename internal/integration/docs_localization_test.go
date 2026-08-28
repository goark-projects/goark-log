package integration

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestDocsLocalization_whenPublicMarkdownExists_shouldHaveChineseCounterpart(t *testing.T) {
	root := filepath.Join("..", "..")
	englishDocs, err := publicMarkdownFiles(root)
	if err != nil {
		t.Fatalf("publicMarkdownFiles() error = %v", err)
	}
	if len(englishDocs) == 0 {
		t.Fatalf("public Markdown documentation should not be empty")
	}
	for _, english := range englishDocs {
		t.Run(english, func(t *testing.T) {
			chinese := strings.TrimSuffix(english, ".md") + ".zh-CN.md"
			englishPath := filepath.Join(root, filepath.FromSlash(english))
			chinesePath := filepath.Join(root, filepath.FromSlash(chinese))
			englishContent, err := os.ReadFile(englishPath)
			if err != nil {
				t.Fatalf("ReadFile(%q) error = %v", englishPath, err)
			}
			chineseContent, err := os.ReadFile(chinesePath)
			if err != nil {
				t.Fatalf("ReadFile(%q) error = %v", chinesePath, err)
			}
			if !strings.Contains(string(englishContent), "[简体中文](") {
				t.Fatalf("%s should link to Simplified Chinese documentation", english)
			}
			if !strings.Contains(string(chineseContent), "[English](") {
				t.Fatalf("%s should link back to English documentation", chinese)
			}
		})
	}
}

func publicMarkdownFiles(root string) ([]string, error) {
	var files []string
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !isDefaultPublicMarkdown(name) {
			continue
		}
		files = append(files, filepath.ToSlash(name))
	}
	for _, dir := range []string{"docs", "examples"} {
		base := filepath.Join(root, dir)
		if err := filepath.WalkDir(base, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || !isDefaultPublicMarkdown(entry.Name()) {
				return nil
			}
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			files = append(files, filepath.ToSlash(relative))
			return nil
		}); err != nil {
			return nil, err
		}
	}
	sort.Strings(files)
	return files, nil
}

func isDefaultPublicMarkdown(name string) bool {
	if name == "AGENTS.md" || name == "CLAUDE.md" {
		return false
	}
	return strings.HasSuffix(name, ".md") && !strings.HasSuffix(name, ".zh-CN.md")
}
