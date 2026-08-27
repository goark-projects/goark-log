package rolling

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// DeleteAction 描述归档文件删除条件。
type DeleteAction struct {
	BasePath string
	MaxDepth int
	Glob     string
	MaxAge   time.Duration
	MaxCount int
	MaxSize  int64
}

type deleteCandidate struct {
	path    string
	name    string
	modTime time.Time
	size    int64
}

// NormalizeDeleteAction 规范化并校验删除动作。
func NormalizeDeleteAction(action DeleteAction) (DeleteAction, error) {
	action.BasePath = strings.TrimSpace(action.BasePath)
	if action.BasePath == "" {
		return DeleteAction{}, fmt.Errorf("basePath is empty")
	}
	action.BasePath = filepath.Clean(action.BasePath)
	action.Glob = strings.TrimSpace(action.Glob)
	if action.Glob == "" {
		action.Glob = "*"
	}
	if _, err := filepath.Match(action.Glob, "probe"); err != nil {
		return DeleteAction{}, fmt.Errorf("glob %q is invalid: %w", action.Glob, err)
	}
	if action.MaxDepth < 0 {
		return DeleteAction{}, fmt.Errorf("maxDepth must be >= 0")
	}
	if action.MaxDepth == 0 {
		action.MaxDepth = 1
	}
	if action.MaxAge < 0 {
		return DeleteAction{}, fmt.Errorf("maxAge must be >= 0")
	}
	if action.MaxCount < 0 {
		return DeleteAction{}, fmt.Errorf("maxCount must be >= 0")
	}
	if action.MaxSize < 0 {
		return DeleteAction{}, fmt.Errorf("maxSize must be >= 0")
	}
	return action, nil
}

// DeleteArchivesByAction 根据删除条件清理归档文件。
func DeleteArchivesByAction(now time.Time, action DeleteAction) error {
	var err error
	action, err = NormalizeDeleteAction(action)
	if err != nil {
		return err
	}
	info, err := os.Stat(action.BasePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("goark-log: stat rolling delete basePath %q: %w", action.BasePath, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("goark-log: rolling delete basePath %q is not a directory", action.BasePath)
	}
	cutoff := time.Time{}
	if action.MaxAge > 0 {
		cutoff = now.Add(-action.MaxAge)
	}
	candidates := make([]deleteCandidate, 0, 16)
	if err := filepath.WalkDir(action.BasePath, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == action.BasePath {
			return nil
		}
		depth := relativeDepth(action.BasePath, path)
		if entry.IsDir() {
			if depth > action.MaxDepth {
				return filepath.SkipDir
			}
			return nil
		}
		if depth > action.MaxDepth {
			return nil
		}
		matched, err := deleteGlobMatch(action.Glob, action.BasePath, path)
		if err != nil {
			return err
		}
		if !matched {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		candidates = append(candidates, deleteCandidate{
			path:    path,
			name:    filepath.ToSlash(path),
			modTime: info.ModTime(),
			size:    info.Size(),
		})
		return nil
	}); err != nil {
		return err
	}
	return deleteArchiveCandidates(candidates, cutoff, action.MaxCount, action.MaxSize)
}

func deleteArchiveCandidates(candidates []deleteCandidate, cutoff time.Time, maxCount int, maxSize int64) error {
	if len(candidates) == 0 {
		return nil
	}
	deleteSet := make(map[string]struct{}, len(candidates))
	if !cutoff.IsZero() {
		for _, candidate := range candidates {
			if candidate.modTime.Before(cutoff) {
				deleteSet[candidate.path] = struct{}{}
			}
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].modTime.Equal(candidates[j].modTime) {
			return candidates[i].name > candidates[j].name
		}
		return candidates[i].modTime.After(candidates[j].modTime)
	})
	if maxCount > 0 {
		for index, candidate := range candidates {
			if index >= maxCount {
				deleteSet[candidate.path] = struct{}{}
			}
		}
	}
	if maxSize > 0 {
		var accumulated int64
		for _, candidate := range candidates {
			accumulated += candidate.size
			if accumulated > maxSize {
				deleteSet[candidate.path] = struct{}{}
			}
		}
	}
	if len(deleteSet) == 0 {
		return nil
	}
	var joined error
	paths := make([]string, 0, len(deleteSet))
	for path := range deleteSet {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			joined = errors.Join(joined, err)
		}
	}
	return joined
}

func relativeDepth(basePath string, path string) int {
	relative, err := filepath.Rel(basePath, path)
	if err != nil || relative == "." {
		return 0
	}
	return strings.Count(filepath.ToSlash(relative), "/") + 1
}

func deleteGlobMatch(glob string, basePath string, path string) (bool, error) {
	if matched, err := filepath.Match(glob, filepath.Base(path)); err != nil || matched {
		return matched, err
	}
	relative, err := filepath.Rel(basePath, path)
	if err != nil {
		return false, err
	}
	return filepath.Match(filepath.ToSlash(glob), filepath.ToSlash(relative))
}
