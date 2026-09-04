package jsoncodec

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"testing"
)

func TestRepositoryUsesSonicAsOnlyJSONImplementation(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imported := range file.Imports {
			name, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				return err
			}
			if name == "encoding/json" {
				t.Errorf("%s 导入了 encoding/json，JSON 实现必须统一使用字节跳动 Sonic", path)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("扫描 Go 源码失败: %v", err)
	}
}
