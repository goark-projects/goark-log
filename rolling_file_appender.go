package goarklog

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
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
	// DefaultRollingActionQueueSize 是异步滚动动作队列默认长度。
	DefaultRollingActionQueueSize = 32
)

// RollingFileAppender 支持按大小、按时间和启动时滚动的文件 appender。
type RollingFileAppender struct {
	name              string
	path              string
	filePattern       string
	layout            Layout
	bufferSize        int
	flushOnWrite      bool
	maxSize           int64
	interval          time.Duration
	cronExpression    string
	cron              *cronSchedule
	modulate          bool
	rolloverOnStartup bool
	maxBackups        int
	maxAge            time.Duration
	compress          bool
	asyncActions      bool
	actionQueueSize   int
	deleteActions     []RollingDeleteAction
	clock             func() time.Time

	mu           sync.Mutex
	file         *os.File
	writer       *bufio.Writer
	size         int64
	nextRollover time.Time
	nextCron     time.Time
	archiveIndex int
	closed       bool

	actionMu     sync.Mutex
	actionQueue  chan func() error
	actionClosed bool
	actionErr    error
	actionWG     sync.WaitGroup
}

// RollingFileOption 调整 RollingFileAppender。
type RollingFileOption func(*RollingFileAppender)

// RollingDeleteAction 描述滚动后执行的归档删除动作。
type RollingDeleteAction struct {
	BasePath string
	MaxDepth int
	Glob     string
	MaxAge   time.Duration
	MaxCount int
	MaxSize  int64
}

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

// WithRollingFileBufferSize 设置滚动文件写缓冲大小，0 表示禁用缓冲。
func WithRollingFileBufferSize(size int) RollingFileOption {
	return func(appender *RollingFileAppender) {
		appender.bufferSize = size
	}
}

// WithRollingFileFlushOnWrite 设置每次写入后立即 flush。
func WithRollingFileFlushOnWrite(enabled bool) RollingFileOption {
	return func(appender *RollingFileAppender) {
		appender.flushOnWrite = enabled
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

// WithRollingCronSchedule 设置 cron 触发滚动表达式。
func WithRollingCronSchedule(expression string) RollingFileOption {
	return func(appender *RollingFileAppender) {
		appender.cronExpression = strings.TrimSpace(expression)
	}
}

// WithRollingTimeModulate 设置时间滚动是否对齐到时间边界。
func WithRollingTimeModulate(enabled bool) RollingFileOption {
	return func(appender *RollingFileAppender) {
		appender.modulate = enabled
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

// WithRollingAsyncActions 设置压缩和清理动作是否由后台串行执行。
func WithRollingAsyncActions(enabled bool) RollingFileOption {
	return func(appender *RollingFileAppender) {
		appender.asyncActions = enabled
	}
}

// WithRollingActionQueueSize 设置后台滚动动作队列长度。
func WithRollingActionQueueSize(size int) RollingFileOption {
	return func(appender *RollingFileAppender) {
		appender.actionQueueSize = size
	}
}

// WithRollingDeleteActions 设置滚动后的归档删除动作。
func WithRollingDeleteActions(actions ...RollingDeleteAction) RollingFileOption {
	return func(appender *RollingFileAppender) {
		appender.deleteActions = append([]RollingDeleteAction(nil), actions...)
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
		bufferSize: DefaultFileBufferSize,
		maxSize:    DefaultRollingMaxSize,
		maxBackups: DefaultRollingMaxBackups,
		modulate:   true,
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
	appender.startActionWorker()
	if err := appender.open(); err != nil {
		_ = appender.closeActionWorker()
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
	var n int
	var err error
	if a.writer != nil {
		n, err = a.writer.Write(buf.Bytes())
		if err == nil && a.flushOnWrite {
			err = a.writer.Flush()
		}
	} else {
		n, err = a.file.Write(buf.Bytes())
	}
	a.size += int64(n)
	return err
}

// Flush 把缓冲日志刷入操作系统文件缓存。
func (a *RollingFileAppender) Flush() error {
	if a == nil {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.flushLocked()
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
	flushErr := a.flushLocked()
	if a.file == nil {
		return errors.Join(flushErr, a.closeActionWorker())
	}
	err := a.file.Close()
	a.file = nil
	a.writer = nil
	actionErr := a.closeActionWorker()
	if flushErr != nil {
		return errors.Join(flushErr, actionErr)
	}
	return errors.Join(err, actionErr)
}

func (a *RollingFileAppender) validate() error {
	if strings.TrimSpace(a.name) == "" {
		return fmt.Errorf("goark-log: rolling file appender name is empty")
	}
	if a.layout == nil {
		a.layout = NewDefaultLayout()
	}
	if a.bufferSize < 0 {
		return fmt.Errorf("goark-log: rolling file buffer size must be >= 0")
	}
	if a.maxSize < 0 {
		return fmt.Errorf("goark-log: rolling max size must be >= 0")
	}
	if a.interval < 0 {
		return fmt.Errorf("goark-log: rolling interval must be >= 0")
	}
	if strings.TrimSpace(a.cronExpression) != "" {
		cron, err := parseCronSchedule(a.cronExpression)
		if err != nil {
			return fmt.Errorf("goark-log: rolling cron schedule %q is invalid: %w", a.cronExpression, err)
		}
		a.cron = cron
	}
	if a.maxBackups < 0 {
		return fmt.Errorf("goark-log: rolling max backups must be >= 0")
	}
	if a.maxAge < 0 {
		return fmt.Errorf("goark-log: rolling max age must be >= 0")
	}
	if a.actionQueueSize < 0 {
		return fmt.Errorf("goark-log: rolling action queue size must be >= 0")
	}
	if a.actionQueueSize == 0 {
		a.actionQueueSize = DefaultRollingActionQueueSize
	}
	for index, action := range a.deleteActions {
		normalized, err := normalizeRollingDeleteAction(action)
		if err != nil {
			return fmt.Errorf("goark-log: rolling delete action %d: %w", index, err)
		}
		a.deleteActions[index] = normalized
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
	if a.maxSize == 0 && a.interval == 0 && a.cron == nil && !a.rolloverOnStartup {
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
	if a.bufferSize > 0 {
		a.writer = bufio.NewWriterSize(file, a.bufferSize)
	}
	a.size = info.Size()
	a.nextRollover = nextRolloverAfter(a.now(), a.interval, a.modulate)
	a.nextCron = nextCronRolloverAfter(a.now(), a.cron)
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
	if a.cron != nil && !a.nextCron.IsZero() && !now.Before(a.nextCron) {
		return true
	}
	return a.maxSize > 0 && a.size > 0 && a.size+pendingBytes > a.maxSize
}

func (a *RollingFileAppender) rollover(now time.Time) error {
	if err := a.flushLocked(); err != nil {
		return fmt.Errorf("goark-log: flush active log file %q: %w", a.path, err)
	}
	if a.file != nil {
		if err := a.file.Close(); err != nil {
			return fmt.Errorf("goark-log: close active log file %q: %w", a.path, err)
		}
		a.file = nil
		a.writer = nil
	}
	target, err := a.nextArchivePath(now)
	if err != nil {
		return err
	}
	archiveIndex := a.archiveIndex - 1
	if err := os.Rename(a.path, target); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("goark-log: rename log file %q to %q: %w", a.path, target, err)
		}
	}
	file, err := openLogFile(a.path)
	if err != nil {
		return err
	}
	a.file = file
	if a.bufferSize > 0 {
		a.writer = bufio.NewWriterSize(file, a.bufferSize)
	}
	a.size = 0
	a.nextRollover = nextRolloverAfter(now, a.interval, a.modulate)
	a.nextCron = nextCronRolloverAfter(now, a.cron)
	return a.runRolloverActions(now, target, archiveIndex)
}

func (a *RollingFileAppender) flushLocked() error {
	if a == nil || a.writer == nil {
		return nil
	}
	return a.writer.Flush()
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

func nextRolloverAfter(now time.Time, interval time.Duration, modulate bool) time.Time {
	if interval <= 0 {
		return time.Time{}
	}
	if !modulate {
		return now.Add(interval)
	}
	if interval == 24*time.Hour {
		year, month, day := now.Date()
		return time.Date(year, month, day+1, 0, 0, 0, 0, now.Location())
	}
	if interval == time.Hour {
		return now.Truncate(time.Hour).Add(time.Hour)
	}
	if interval == time.Minute {
		return now.Truncate(time.Minute).Add(time.Minute)
	}
	truncated := now.Truncate(interval)
	if truncated.Equal(now) {
		return now.Add(interval)
	}
	return truncated.Add(interval)
}

func nextCronRolloverAfter(now time.Time, cron *cronSchedule) time.Time {
	next, ok := cron.next(now)
	if !ok {
		return time.Time{}
	}
	return next
}

func normalizeRollingDeleteAction(action RollingDeleteAction) (RollingDeleteAction, error) {
	action.BasePath = strings.TrimSpace(action.BasePath)
	if action.BasePath == "" {
		return RollingDeleteAction{}, fmt.Errorf("basePath is empty")
	}
	action.BasePath = filepath.Clean(action.BasePath)
	action.Glob = strings.TrimSpace(action.Glob)
	if action.Glob == "" {
		action.Glob = "*"
	}
	if _, err := filepath.Match(action.Glob, "probe"); err != nil {
		return RollingDeleteAction{}, fmt.Errorf("glob %q is invalid: %w", action.Glob, err)
	}
	if action.MaxDepth < 0 {
		return RollingDeleteAction{}, fmt.Errorf("maxDepth must be >= 0")
	}
	if action.MaxDepth == 0 {
		action.MaxDepth = 1
	}
	if action.MaxAge < 0 {
		return RollingDeleteAction{}, fmt.Errorf("maxAge must be >= 0")
	}
	if action.MaxCount < 0 {
		return RollingDeleteAction{}, fmt.Errorf("maxCount must be >= 0")
	}
	if action.MaxSize < 0 {
		return RollingDeleteAction{}, fmt.Errorf("maxSize must be >= 0")
	}
	return action, nil
}

func (a *RollingFileAppender) startActionWorker() {
	if a == nil || !a.asyncActions {
		return
	}
	a.actionQueue = make(chan func() error, a.actionQueueSize)
	a.actionWG.Add(1)
	go a.runActionWorker()
}

func (a *RollingFileAppender) runActionWorker() {
	defer a.actionWG.Done()
	for action := range a.actionQueue {
		if action == nil {
			continue
		}
		if err := action(); err != nil {
			a.actionMu.Lock()
			a.actionErr = errors.Join(a.actionErr, err)
			a.actionMu.Unlock()
		}
	}
}

func (a *RollingFileAppender) closeActionWorker() error {
	if a == nil {
		return nil
	}
	a.actionMu.Lock()
	queue := a.actionQueue
	if queue == nil || a.actionClosed {
		err := a.actionErr
		a.actionMu.Unlock()
		return err
	}
	a.actionClosed = true
	close(queue)
	a.actionMu.Unlock()
	a.actionWG.Wait()
	a.actionMu.Lock()
	err := a.actionErr
	a.actionMu.Unlock()
	return err
}

func (a *RollingFileAppender) runRolloverActions(now time.Time, target string, archiveIndex int) error {
	_, compressedTarget, err := a.archivePaths(now, archiveIndex)
	if err != nil {
		return err
	}
	action := func() error {
		var joined error
		if a.compress {
			if _, err := compressFileTo(target, compressedTarget); err != nil {
				joined = errors.Join(joined, err)
			}
		}
		joined = errors.Join(joined, a.deleteExpiredArchives(now))
		joined = errors.Join(joined, a.runDeleteActions(now))
		return joined
	}
	if !a.asyncActions {
		return action()
	}
	return a.enqueueRolloverAction(action)
}

func (a *RollingFileAppender) enqueueRolloverAction(action func() error) error {
	a.actionMu.Lock()
	queue := a.actionQueue
	closed := a.actionClosed
	a.actionMu.Unlock()
	if queue == nil || closed {
		return action()
	}
	select {
	case queue <- action:
		return nil
	default:
		return action()
	}
}

func (a *RollingFileAppender) runDeleteActions(now time.Time) error {
	var joined error
	for _, action := range a.deleteActions {
		joined = errors.Join(joined, deleteArchivesByAction(now, action))
	}
	return joined
}

func deleteArchivesByAction(now time.Time, action RollingDeleteAction) error {
	var err error
	action, err = normalizeRollingDeleteAction(action)
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
		matched, err := rollingDeleteGlobMatch(action.Glob, action.BasePath, path)
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

type deleteCandidate struct {
	path    string
	name    string
	modTime time.Time
	size    int64
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

func rollingDeleteGlobMatch(glob string, basePath string, path string) (bool, error) {
	if matched, err := filepath.Match(glob, filepath.Base(path)); err != nil || matched {
		return matched, err
	}
	relative, err := filepath.Rel(basePath, path)
	if err != nil {
		return false, err
	}
	return filepath.Match(filepath.ToSlash(glob), filepath.ToSlash(relative))
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
