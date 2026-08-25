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
	"regexp"
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
	filePattern       string
	layout            Layout
	maxSize           int64
	interval          time.Duration
	rolloverOnStartup bool
	maxBackups        int
	maxAge            time.Duration
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

// WithRollingFilePattern 设置滚动档案路径模式，支持 %d{layout} 和 %i。
func WithRollingFilePattern(pattern string) RollingFileOption {
	return func(appender *RollingFileAppender) {
		appender.filePattern = strings.TrimSpace(pattern)
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

// WithRollingMaxAge 设置档案最大保留时间，0 表示不按时间清理。
func WithRollingMaxAge(age time.Duration) RollingFileOption {
	return func(appender *RollingFileAppender) {
		appender.maxAge = age
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
	if a.maxAge < 0 {
		return fmt.Errorf("goark-log: rolling max age must be >= 0")
	}
	if a.filePattern != "" {
		if a.maxSize > 0 && !rollingPatternHasIndex(a.filePattern) {
			return fmt.Errorf("goark-log: rolling filePattern requires %%i when size policy is enabled")
		}
		if strings.HasSuffix(strings.ToLower(a.filePattern), ".gz") {
			a.compress = true
		}
		candidate, _, err := a.archivePaths(a.now(), 0)
		if err != nil {
			return err
		}
		if filepath.Clean(candidate) == filepath.Clean(a.path) {
			return fmt.Errorf("goark-log: rolling filePattern must not resolve to active log file %q", a.path)
		}
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
		_, compressedTarget, err := a.archivePaths(now, a.archiveIndex-1)
		if err != nil {
			return err
		}
		if _, err := compressFileTo(target, compressedTarget); err != nil {
			return err
		}
	}
	if err := a.deleteExpiredArchives(now); err != nil {
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
	for attempt := 0; attempt < 1000; attempt++ {
		index := a.archiveIndex
		a.archiveIndex++
		candidate, compressedCandidate, err := a.archivePaths(now, index)
		if err != nil {
			return "", err
		}
		if exists, err := pathExists(candidate); err != nil {
			return "", fmt.Errorf("goark-log: stat archive log file %q: %w", candidate, err)
		} else if exists {
			continue
		}
		if a.compress && compressedCandidate != candidate {
			if exists, err := pathExists(compressedCandidate); err != nil {
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
	target, err := formatRollingFilePattern(a.filePattern, now, index)
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
	glob := rollingPatternGlob(a.filePattern, a.compress)
	matches, err := filepath.Glob(glob)
	if err != nil {
		return fmt.Errorf("goark-log: glob rolling filePattern %q: %w", a.filePattern, err)
	}
	pattern, hasIndex, err := rollingPatternIndexRegexp(a.filePattern, a.compress)
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

func formatRollingFilePattern(pattern string, now time.Time, index int) (string, error) {
	var builder strings.Builder
	builder.Grow(len(pattern) + 8)
	for offset := 0; offset < len(pattern); {
		if pattern[offset] != '%' {
			builder.WriteByte(pattern[offset])
			offset++
			continue
		}
		if offset+1 < len(pattern) && pattern[offset+1] == '%' {
			builder.WriteByte('%')
			offset += 2
			continue
		}
		next, err := appendRollingPatternToken(&builder, pattern, offset, now, index)
		if err != nil {
			return "", err
		}
		offset = next
	}
	return builder.String(), nil
}

func appendRollingPatternToken(builder *strings.Builder, pattern string, offset int, now time.Time, index int) (int, error) {
	cursor := offset + 1
	zeroPad := false
	width := 0
	if cursor < len(pattern) && pattern[cursor] == '0' {
		zeroPad = true
		cursor++
	}
	for cursor < len(pattern) && pattern[cursor] >= '0' && pattern[cursor] <= '9' {
		width = width*10 + int(pattern[cursor]-'0')
		cursor++
	}
	if cursor >= len(pattern) {
		return 0, fmt.Errorf("goark-log: rolling filePattern token is incomplete near %q", pattern[offset:])
	}
	switch pattern[cursor] {
	case 'i':
		value := strconv.Itoa(index)
		if zeroPad && width > len(value) {
			builder.WriteString(strings.Repeat("0", width-len(value)))
		}
		builder.WriteString(value)
		return cursor + 1, nil
	case 'd':
		option := ""
		cursor++
		if cursor < len(pattern) && pattern[cursor] == '{' {
			end := strings.IndexByte(pattern[cursor+1:], '}')
			if end < 0 {
				return 0, fmt.Errorf("goark-log: rolling filePattern date option is not closed near %q", pattern[cursor:])
			}
			option = pattern[cursor+1 : cursor+1+end]
			cursor += end + 2
		}
		if strings.TrimSpace(option) == "" {
			option = "yyyyMMdd-HHmmss"
		}
		layout, unixMode := normalizeTimePattern(option)
		switch unixMode {
		case timeUnixSeconds:
			builder.WriteString(strconv.FormatInt(now.Unix(), 10))
		case timeUnixMillis:
			builder.WriteString(strconv.FormatInt(now.UnixMilli(), 10))
		case timeUnixMicros:
			builder.WriteString(strconv.FormatInt(now.UnixMicro(), 10))
		case timeUnixNanos:
			builder.WriteString(strconv.FormatInt(now.UnixNano(), 10))
		default:
			builder.WriteString(now.Format(layout))
		}
		return cursor, nil
	default:
		return 0, fmt.Errorf("goark-log: unsupported rolling filePattern token near %q", pattern[offset:])
	}
}

func rollingPatternHasIndex(pattern string) bool {
	for offset := 0; offset < len(pattern); offset++ {
		if pattern[offset] != '%' {
			continue
		}
		cursor := offset + 1
		if cursor < len(pattern) && pattern[cursor] == '0' {
			cursor++
		}
		for cursor < len(pattern) && pattern[cursor] >= '0' && pattern[cursor] <= '9' {
			cursor++
		}
		if cursor < len(pattern) && pattern[cursor] == 'i' {
			return true
		}
	}
	return false
}

func rollingPatternGlob(pattern string, compress bool) string {
	var builder strings.Builder
	builder.Grow(len(pattern) + 8)
	for offset := 0; offset < len(pattern); {
		if pattern[offset] != '%' {
			builder.WriteByte(pattern[offset])
			offset++
			continue
		}
		if offset+1 < len(pattern) && pattern[offset+1] == '%' {
			builder.WriteByte('%')
			offset += 2
			continue
		}
		cursor := offset + 1
		if cursor < len(pattern) && pattern[cursor] == '0' {
			cursor++
		}
		for cursor < len(pattern) && pattern[cursor] >= '0' && pattern[cursor] <= '9' {
			cursor++
		}
		if cursor < len(pattern) && pattern[cursor] == 'd' {
			cursor++
			if cursor < len(pattern) && pattern[cursor] == '{' {
				end := strings.IndexByte(pattern[cursor+1:], '}')
				if end >= 0 {
					cursor += end + 2
				}
			}
			builder.WriteByte('*')
			offset = cursor
			continue
		}
		if cursor < len(pattern) && pattern[cursor] == 'i' {
			builder.WriteByte('*')
			offset = cursor + 1
			continue
		}
		builder.WriteByte('*')
		offset = cursor
	}
	glob := builder.String()
	if compress && !strings.HasSuffix(strings.ToLower(glob), ".gz") {
		glob += ".gz"
	}
	return filepath.Clean(glob)
}

func rollingPatternIndexRegexp(pattern string, compress bool) (*regexp.Regexp, bool, error) {
	var builder strings.Builder
	builder.WriteByte('^')
	hasIndex := false
	slashPattern := filepath.ToSlash(pattern)
	for offset := 0; offset < len(slashPattern); {
		if slashPattern[offset] != '%' {
			builder.WriteString(regexp.QuoteMeta(string(slashPattern[offset])))
			offset++
			continue
		}
		if offset+1 < len(slashPattern) && slashPattern[offset+1] == '%' {
			builder.WriteString(regexp.QuoteMeta("%"))
			offset += 2
			continue
		}
		cursor := offset + 1
		if cursor < len(slashPattern) && slashPattern[cursor] == '0' {
			cursor++
		}
		for cursor < len(slashPattern) && slashPattern[cursor] >= '0' && slashPattern[cursor] <= '9' {
			cursor++
		}
		if cursor < len(slashPattern) && slashPattern[cursor] == 'i' {
			builder.WriteString(`(\d+)`)
			hasIndex = true
			offset = cursor + 1
			continue
		}
		if cursor < len(slashPattern) && slashPattern[cursor] == 'd' {
			cursor++
			if cursor < len(slashPattern) && slashPattern[cursor] == '{' {
				end := strings.IndexByte(slashPattern[cursor+1:], '}')
				if end >= 0 {
					cursor += end + 2
				}
			}
			builder.WriteString(`.+?`)
			offset = cursor
			continue
		}
		builder.WriteString(`.+?`)
		offset = cursor
	}
	if compress && !strings.HasSuffix(strings.ToLower(slashPattern), ".gz") {
		builder.WriteString(regexp.QuoteMeta(".gz"))
	}
	builder.WriteByte('$')
	compiled, err := regexp.Compile(builder.String())
	if err != nil {
		return nil, false, fmt.Errorf("goark-log: compile rolling filePattern index matcher: %w", err)
	}
	return compiled, hasIndex, nil
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
	return compressFileTo(path, path+".gz")
}

func compressFileTo(path string, compressedPath string) (string, error) {
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
