package goarklog

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultRollingMaxSize 是 RollingFileAppender 默认按大小滚动阈值。
	DefaultRollingMaxSize int64 = 10 * 1024 * 1024
	// DefaultRollingMaxBackups 是 RollingFileAppender 默认保留档案数量。
	DefaultRollingMaxBackups = 7
)

// RollingFileAppender 支持按大小、按时间和启动时滚动的文件 appender。
type RollingFileAppender struct {
	name              string
	path              string
	layout            Layout
	maxSize           int64
	interval          time.Duration
	rolloverOnStartup bool
	maxBackups        int
	compress          bool
	clock             func() time.Time

	mu           sync.Mutex
	file         *os.File
	size         int64
	nextRollover time.Time
	archiveIndex int
	closed       bool
}

// RollingFileOption 调整 RollingFileAppender。
type RollingFileOption func(*RollingFileAppender)

// WithRollingFileName 设置 appender 名称。
func WithRollingFileName(name string) RollingFileOption {
	return func(appender *RollingFileAppender) {
		appender.name = name
	}
}

// WithRollingFileLayout 设置日志布局。
func WithRollingFileLayout(layout Layout) RollingFileOption {
	return func(appender *RollingFileAppender) {
		appender.layout = layout
	}
}

// WithRollingMaxSize 设置按大小滚动阈值，0 表示禁用按大小滚动。
func WithRollingMaxSize(bytes int64) RollingFileOption {
	return func(appender *RollingFileAppender) {
		appender.maxSize = bytes
	}
}

// WithRollingInterval 设置按固定时间间隔滚动，0 表示禁用按时间滚动。
func WithRollingInterval(interval time.Duration) RollingFileOption {
	return func(appender *RollingFileAppender) {
		appender.interval = interval
	}
}

// WithRolloverOnStartup 设置启动时滚动已有文件。
func WithRolloverOnStartup(enabled bool) RollingFileOption {
	return func(appender *RollingFileAppender) {
		appender.rolloverOnStartup = enabled
	}
}

// WithRollingMaxBackups 设置保留档案数量，0 表示不保留历史档案。
func WithRollingMaxBackups(maxBackups int) RollingFileOption {
	return func(appender *RollingFileAppender) {
		appender.maxBackups = maxBackups
	}
}

// WithRollingGzip 设置是否对滚动档案执行 gzip 压缩。
func WithRollingGzip(enabled bool) RollingFileOption {
	return func(appender *RollingFileAppender) {
		appender.compress = enabled
	}
}

func withRollingClock(clock func() time.Time) RollingFileOption {
	return func(appender *RollingFileAppender) {
		appender.clock = clock
	}
}

// NewRollingFileAppender 创建滚动文件 appender。
func NewRollingFileAppender(path string, options ...RollingFileOption) (*RollingFileAppender, error) {
	cleanPath, err := validateLogFilePath(path)
	if err != nil {
		return nil, err
	}
	appender := &RollingFileAppender{
		name:       "rollingFile",
		path:       cleanPath,
		layout:     NewDefaultLayout(),
		maxSize:    DefaultRollingMaxSize,
		maxBackups: DefaultRollingMaxBackups,
		clock:      time.Now,
	}
	for _, option := range options {
		if option != nil {
			option(appender)
		}
	}
	if err := appender.validate(); err != nil {
		return nil, err
	}
	if err := appender.open(); err != nil {
		return nil, err
	}
	if appender.rolloverOnStartup && appender.size > 0 {
		if err := appender.rollover(appender.now()); err != nil {
			_ = appender.Close()
			return nil, err
		}
	}
	return appender, nil
}

func (a *RollingFileAppender) Name() string {
	if a == nil || a.name == "" {
		return "rollingFile"
	}
	return a.name
}

func (a *RollingFileAppender) Append(ctx context.Context, event Event) error {
	if a == nil {
		return fmt.Errorf("goark-log: rolling file appender is nil")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer releaseBuffer(buf)
	if err := a.layout.Format(buf, event); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed || a.file == nil {
		return fmt.Errorf("goark-log: rolling file appender %q is closed", a.Name())
	}
	now := event.Time
	if now.IsZero() {
		now = a.now()
	}
	if a.shouldRollover(now, int64(buf.Len())) {
		if err := a.rollover(now); err != nil {
			return err
		}
	}
	n, err := a.file.Write(buf.Bytes())
	a.size += int64(n)
	return err
}

func (a *RollingFileAppender) Close() error {
	if a == nil {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed {
		return nil
	}
	a.closed = true
	if a.file == nil {
		return nil
	}
	err := a.file.Close()
	a.file = nil
	return err
}

func (a *RollingFileAppender) validate() error {
	if strings.TrimSpace(a.name) == "" {
		return fmt.Errorf("goark-log: rolling file appender name is empty")
	}
	if a.layout == nil {
		a.layout = NewDefaultLayout()
	}
	if a.maxSize < 0 {
		return fmt.Errorf("goark-log: rolling max size must be >= 0")
	}
	if a.interval < 0 {
		return fmt.Errorf("goark-log: rolling interval must be >= 0")
	}
	if a.maxBackups < 0 {
		return fmt.Errorf("goark-log: rolling max backups must be >= 0")
	}
	if a.maxSize == 0 && a.interval == 0 && !a.rolloverOnStartup {
		return fmt.Errorf("goark-log: rolling policy is empty")
	}
	if a.clock == nil {
		a.clock = time.Now
	}
	return nil
}

func (a *RollingFileAppender) open() error {
	file, err := openLogFile(a.path)
	if err != nil {
		return err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return fmt.Errorf("goark-log: stat log file %q: %w", a.path, err)
	}
	a.file = file
	a.size = info.Size()
	a.nextRollover = nextRolloverAfter(a.now(), a.interval)
	if err := a.initArchiveIndex(); err != nil {
		_ = file.Close()
		a.file = nil
		return err
	}
	return nil
}

func (a *RollingFileAppender) now() time.Time {
	return a.clock()
}

func (a *RollingFileAppender) shouldRollover(now time.Time, pendingBytes int64) bool {
	if a.interval > 0 && !a.nextRollover.IsZero() && !now.Before(a.nextRollover) {
		return true
	}
	return a.maxSize > 0 && a.size > 0 && a.size+pendingBytes > a.maxSize
}

func (a *RollingFileAppender) rollover(now time.Time) error {
	if a.file != nil {
		if err := a.file.Close(); err != nil {
			return fmt.Errorf("goark-log: close active log file %q: %w", a.path, err)
		}
		a.file = nil
	}
	target, err := a.nextArchivePath(now)
	if err != nil {
		return err
	}
	if err := os.Rename(a.path, target); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("goark-log: rename log file %q to %q: %w", a.path, target, err)
		}
	}
	if a.compress {
		if _, err := compressFile(target); err != nil {
			return err
		}
	}
	if err := a.deleteExpiredArchives(); err != nil {
		return err
	}
	file, err := openLogFile(a.path)
	if err != nil {
		return err
	}
	a.file = file
	a.size = 0
	a.nextRollover = nextRolloverAfter(now, a.interval)
	return nil
}

func (a *RollingFileAppender) nextArchivePath(now time.Time) (string, error) {
	dir := filepath.Dir(a.path)
	base := filepath.Base(a.path)
	stamp := now.Format("20060102-150405.000")
	for attempt := 0; attempt < 1000; attempt++ {
		index := a.archiveIndex
		a.archiveIndex++
		name := fmt.Sprintf("%s.%s.%03d", base, stamp, index)
		candidate := filepath.Join(dir, name)
		if exists, err := pathExists(candidate); err != nil {
			return "", fmt.Errorf("goark-log: stat archive log file %q: %w", candidate, err)
		} else if exists {
			continue
		}
		if a.compress {
			compressedCandidate := candidate + ".gz"
			if exists, err := pathExists(compressedCandidate); err != nil {
				return "", fmt.Errorf("goark-log: stat archive log file %q: %w", compressedCandidate, err)
			} else if exists {
				continue
			}
		}
		return candidate, nil
	}
	return "", fmt.Errorf("goark-log: cannot allocate archive name for %q", a.path)
}

func (a *RollingFileAppender) initArchiveIndex() error {
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

func pathExists(path string) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return true, nil
	} else if errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else {
		return false, err
	}
}

func (a *RollingFileAppender) deleteExpiredArchives() error {
	dir := filepath.Dir(a.path)
	base := filepath.Base(a.path)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("goark-log: read log directory %q: %w", dir, err)
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
	if len(archives) <= a.maxBackups {
		return nil
	}
	sort.Slice(archives, func(i, j int) bool {
		return archives[i].name < archives[j].name
	})
	var joined error
	for _, archive := range archives[:len(archives)-a.maxBackups] {
		joined = errors.Join(joined, os.Remove(archive.path))
	}
	return joined
}

type archiveFile struct {
	path string
	name string
}

func nextRolloverAfter(now time.Time, interval time.Duration) time.Time {
	if interval <= 0 {
		return time.Time{}
	}
	truncated := now.Truncate(interval)
	if truncated.Equal(now) {
		return now.Add(interval)
	}
	return truncated.Add(interval)
}

func compressFile(path string) (string, error) {
	source, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("goark-log: open archive log file %q: %w", path, err)
	}
	sourceClosed := false
	defer func() {
		if !sourceClosed {
			_ = source.Close()
		}
	}()
	compressedPath := path + ".gz"
	target, err := os.OpenFile(compressedPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return "", fmt.Errorf("goark-log: create gzip archive %q: %w", compressedPath, err)
	}
	removeCompressed := true
	defer func() {
		if removeCompressed {
			_ = os.Remove(compressedPath)
		}
	}()
	gzipWriter := gzip.NewWriter(target)
	if _, err := io.Copy(gzipWriter, source); err != nil {
		_ = gzipWriter.Close()
		_ = target.Close()
		return "", fmt.Errorf("goark-log: gzip archive %q: %w", path, err)
	}
	if err := gzipWriter.Close(); err != nil {
		_ = target.Close()
		return "", fmt.Errorf("goark-log: close gzip archive %q: %w", compressedPath, err)
	}
	if err := source.Close(); err != nil {
		_ = target.Close()
		return "", fmt.Errorf("goark-log: close archive log file %q: %w", path, err)
	}
	sourceClosed = true
	if err := target.Close(); err != nil {
		return "", fmt.Errorf("goark-log: close gzip file %q: %w", compressedPath, err)
	}
	if err := os.Remove(path); err != nil {
		return "", fmt.Errorf("goark-log: remove uncompressed archive %q: %w", path, err)
	}
	removeCompressed = false
	return compressedPath, nil
}
