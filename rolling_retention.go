package goarklog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type archiveFile struct {
	path string
	name string
}

func (a *RollingFileAppender) deleteExpiredArchives(now time.Time) error {
	archives, err := a.archiveFiles()
	if err != nil {
		return err
	}
	if len(archives) <= a.maxBackups {
		if a.maxAge <= 0 {
			return nil
		}
	}
	sort.Slice(archives, func(i, j int) bool {
		return archives[i].name < archives[j].name
	})
	var joined error
	deleteCount := 0
	if len(archives) > a.maxBackups {
		deleteCount = len(archives) - a.maxBackups
	}
	for _, archive := range archives[:deleteCount] {
		joined = errors.Join(joined, os.Remove(archive.path))
	}
	if a.maxAge > 0 {
		cutoff := now.Add(-a.maxAge)
		for _, archive := range archives[deleteCount:] {
			info, err := os.Stat(archive.path)
			if err != nil {
				joined = errors.Join(joined, err)
				continue
			}
			if info.ModTime().Before(cutoff) {
				joined = errors.Join(joined, os.Remove(archive.path))
			}
		}
	}
	return joined
}

func (a *RollingFileAppender) archiveFiles() ([]archiveFile, error) {
	if a.filePattern != "" {
		matches, err := filepath.Glob(rollingPatternGlob(a.filePattern, a.compress))
		if err != nil {
			return nil, fmt.Errorf("goark-log: glob rolling filePattern %q: %w", a.filePattern, err)
		}
		archives := make([]archiveFile, 0, len(matches))
		for _, match := range matches {
			info, err := os.Stat(match)
			if err == nil && !info.IsDir() {
				archives = append(archives, archiveFile{path: match, name: filepath.ToSlash(match)})
			}
		}
		return archives, nil
	}
	dir := filepath.Dir(a.path)
	base := filepath.Base(a.path)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("goark-log: read log directory %q: %w", dir, err)
	}
	prefix := base + "."
	archives := make([]archiveFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), prefix) {
			continue
		}
		archives = append(archives, archiveFile{
			path: filepath.Join(dir, entry.Name()),
			name: entry.Name(),
		})
	}
	return archives, nil
}
