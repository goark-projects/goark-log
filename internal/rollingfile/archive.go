package rollingfile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"goark.dev/log/internal/logfile"
	"goark.dev/log/internal/rolling"
)

type archiveFile struct {
	path    string
	name    string
	size    int64
	modTime time.Time
}

func (a *RollingFileAppender) nextArchivePath(now time.Time) (string, error) {
	if a.fileIndexMode == RollingFileIndexMin && a.filePattern != "" && a.maxBackups > 0 {
		return a.nextMinIndexArchivePath(now)
	}
	for attempt := 0; attempt < 1000; attempt++ {
		index := a.archiveIndex
		a.archiveIndex++
		candidate, compressedCandidate, err := a.archivePaths(now, index)
		if err != nil {
			return "", err
		}
		if exists, err := logfile.Exists(candidate); err != nil {
			return "", fmt.Errorf("goark-log: stat archive log file %q: %w", candidate, err)
		} else if exists {
			continue
		}
		if a.compress && compressedCandidate != candidate {
			if exists, err := logfile.Exists(compressedCandidate); err != nil {
				return "", fmt.Errorf("goark-log: stat archive log file %q: %w", compressedCandidate, err)
			} else if exists {
				continue
			}
		}
		if err := os.MkdirAll(filepath.Dir(candidate), 0o755); err != nil {
			return "", fmt.Errorf("goark-log: create archive directory %q: %w", filepath.Dir(candidate), err)
		}
		return candidate, nil
	}
	return "", fmt.Errorf("goark-log: cannot allocate archive name for %q", a.path)
}

func (a *RollingFileAppender) nextMinIndexArchivePath(now time.Time) (string, error) {
	for index := 1; index <= a.maxBackups; index++ {
		candidate, compressedCandidate, err := a.archivePaths(now, index)
		if err != nil {
			return "", err
		}
		if archivePathAvailable(candidate, compressedCandidate, a.compress) {
			a.archiveIndex = index + 1
			if err := os.MkdirAll(filepath.Dir(candidate), 0o755); err != nil {
				return "", fmt.Errorf("goark-log: create archive directory %q: %w", filepath.Dir(candidate), err)
			}
			return candidate, nil
		}
	}
	if err := a.rotateMinIndexArchives(now); err != nil {
		return "", err
	}
	candidate, _, err := a.archivePaths(now, 1)
	if err != nil {
		return "", err
	}
	a.archiveIndex = 2
	if err := os.MkdirAll(filepath.Dir(candidate), 0o755); err != nil {
		return "", fmt.Errorf("goark-log: create archive directory %q: %w", filepath.Dir(candidate), err)
	}
	return candidate, nil
}

func archivePathAvailable(candidate string, compressedCandidate string, compressed bool) bool {
	if exists, err := logfile.Exists(candidate); err != nil || exists {
		return false
	}
	if compressed && compressedCandidate != candidate {
		if exists, err := logfile.Exists(compressedCandidate); err != nil || exists {
			return false
		}
	}
	return true
}

func (a *RollingFileAppender) rotateMinIndexArchives(now time.Time) error {
	for index := a.maxBackups; index >= 1; index-- {
		_, currentCompressed, err := a.archivePaths(now, index)
		if err != nil {
			return err
		}
		current, _, err := a.archivePaths(now, index)
		if err != nil {
			return err
		}
		source := current
		if a.compress {
			source = currentCompressed
		}
		if index == a.maxBackups {
			if err := os.Remove(source); err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("goark-log: remove archive log file %q: %w", source, err)
			}
			continue
		}
		_, nextCompressed, err := a.archivePaths(now, index+1)
		if err != nil {
			return err
		}
		next, _, err := a.archivePaths(now, index+1)
		if err != nil {
			return err
		}
		target := next
		if a.compress {
			target = nextCompressed
		}
		if err := os.Rename(source, target); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("goark-log: rename archive log file %q to %q: %w", source, target, err)
		}
	}
	return nil
}

func (a *RollingFileAppender) archivePaths(now time.Time, index int) (string, string, error) {
	if a.filePattern == "" {
		dir := filepath.Dir(a.path)
		base := filepath.Base(a.path)
		stamp := now.Format("20060102-150405.000")
		candidate := filepath.Join(dir, fmt.Sprintf("%s.%s.%03d", base, stamp, index))
		if a.compress {
			return candidate, candidate + ".gz", nil
		}
		return candidate, candidate, nil
	}
	target, err := rolling.FormatPattern(a.filePattern, now, index)
	if err != nil {
		return "", "", err
	}
	target = filepath.Clean(target)
	if a.compress {
		if strings.HasSuffix(strings.ToLower(target), ".gz") {
			return strings.TrimSuffix(target, ".gz"), target, nil
		}
		return target, target + ".gz", nil
	}
	return target, target, nil
}

func (a *RollingFileAppender) initArchiveIndex() error {
	if a.filePattern != "" {
		return a.initArchiveIndexByPattern()
	}
	dir := filepath.Dir(a.path)
	base := filepath.Base(a.path)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("goark-log: read log directory %q: %w", dir, err)
	}
	prefix := base + "."
	maxIndex := -1
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), prefix) {
			continue
		}
		index, ok := parseArchiveIndex(entry.Name(), prefix)
		if ok && index > maxIndex {
			maxIndex = index
		}
	}
	a.archiveIndex = maxIndex + 1
	return nil
}

func (a *RollingFileAppender) initArchiveIndexByPattern() error {
	glob := rolling.PatternGlob(a.filePattern, a.compress)
	matches, err := filepath.Glob(glob)
	if err != nil {
		return fmt.Errorf("goark-log: glob rolling filePattern %q: %w", a.filePattern, err)
	}
	pattern, hasIndex, err := rolling.PatternIndexRegexp(a.filePattern, a.compress)
	if err != nil {
		return err
	}
	maxIndex := -1
	for _, match := range matches {
		if !hasIndex {
			maxIndex++
			continue
		}
		parts := pattern.FindStringSubmatch(filepath.ToSlash(match))
		if len(parts) != 2 {
			continue
		}
		index, err := strconv.Atoi(parts[1])
		if err == nil && index > maxIndex {
			maxIndex = index
		}
	}
	a.archiveIndex = maxIndex + 1
	return nil
}

func parseArchiveIndex(name string, prefix string) (int, bool) {
	tail := strings.TrimPrefix(name, prefix)
	tail = strings.TrimSuffix(tail, ".gz")
	indexStart := strings.LastIndexByte(tail, '.')
	if indexStart < 0 || indexStart == len(tail)-1 {
		return 0, false
	}
	index, err := strconv.Atoi(tail[indexStart+1:])
	if err != nil {
		return 0, false
	}
	return index, true
}

func (a *RollingFileAppender) deleteExpiredArchives(now time.Time) error {
	archives, err := a.archiveFiles()
	if err != nil {
		return err
	}
	if len(archives) <= a.maxBackups && a.maxAge <= 0 && a.totalSizeCap <= 0 {
		return nil
	}
	sort.Slice(archives, func(i, j int) bool {
		return archives[i].name < archives[j].name
	})
	var joined error
	removed := make(map[string]struct{})
	deleteCount := 0
	if len(archives) > a.maxBackups {
		deleteCount = len(archives) - a.maxBackups
	}
	for _, archive := range archives[:deleteCount] {
		joined = errors.Join(joined, os.Remove(archive.path))
		removed[archive.path] = struct{}{}
	}
	if a.maxAge > 0 {
		cutoff := now.Add(-a.maxAge)
		for _, archive := range archives[deleteCount:] {
			if archive.modTime.Before(cutoff) {
				joined = errors.Join(joined, os.Remove(archive.path))
				removed[archive.path] = struct{}{}
			}
		}
	}
	if a.totalSizeCap > 0 {
		var total int64
		for _, archive := range archives {
			if _, deleted := removed[archive.path]; !deleted {
				total += archive.size
			}
		}
		for _, archive := range archives {
			if total <= a.totalSizeCap {
				break
			}
			if _, deleted := removed[archive.path]; deleted {
				continue
			}
			joined = errors.Join(joined, os.Remove(archive.path))
			total -= archive.size
		}
	}
	return joined
}

func (a *RollingFileAppender) archiveFiles() ([]archiveFile, error) {
	if a.filePattern != "" {
		matches, err := filepath.Glob(rolling.PatternGlob(a.filePattern, a.compress))
		if err != nil {
			return nil, fmt.Errorf("goark-log: glob rolling filePattern %q: %w", a.filePattern, err)
		}
		archives := make([]archiveFile, 0, len(matches))
		for _, match := range matches {
			info, err := os.Stat(match)
			if err == nil && !info.IsDir() {
				archives = append(archives, archiveFile{path: match, name: filepath.ToSlash(match), size: info.Size(), modTime: info.ModTime()})
			}
		}
		return archives, nil
	}
	dir := filepath.Dir(a.path)
	base := filepath.Base(a.path)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("goark-log: read log directory %q: %w", dir, err)
	}
	prefix := base + "."
	archives := make([]archiveFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), prefix) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("goark-log: stat archive log file %q: %w", entry.Name(), err)
		}
		archives = append(archives, archiveFile{
			path:    filepath.Join(dir, entry.Name()),
			name:    entry.Name(),
			size:    info.Size(),
			modTime: info.ModTime(),
		})
	}
	return archives, nil
}
