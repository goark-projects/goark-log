package rollingfile

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	internalfileappender "goark.dev/log/internal/fileappender"
	internallayout "goark.dev/log/internal/layout"
	"goark.dev/log/internal/logevent"
	"goark.dev/log/internal/logfile"
	"goark.dev/log/internal/rolling"
)

const (
	// DefaultRollingMaxSize 是 RollingFileAppender 默认按大小滚动阈值。
	DefaultRollingMaxSize int64 = 10 * 1024 * 1024
	// DefaultRollingMaxBackups 是 RollingFileAppender 默认保留档案数量。
	DefaultRollingMaxBackups = 7
	// DefaultRollingActionQueueSize 是异步滚动动作队列默认长度。
	DefaultRollingActionQueueSize = 32
)

// Event 是滚动文件输出端处理的事件快照。
type Event = logevent.Event

// Layout 是滚动文件输出端依赖的布局接口。
type Layout = internallayout.Layout

// RollingFileIndexMode 定义 filePattern 中 %i 的分配策略。
type RollingFileIndexMode string

const (
	RollingFileIndexNoMax RollingFileIndexMode = "nomax"
	RollingFileIndexMax   RollingFileIndexMode = "max"
	RollingFileIndexMin   RollingFileIndexMode = "min"
)

// RollingFileAppender 支持按大小、按时间和启动时滚动的文件 appender。
type RollingFileAppender struct {
	name              string
	path              string
	filePattern       string
	fileIndexMode     RollingFileIndexMode
	directWrite       bool
	layout            Layout
	bufferSize        int
	flushOnWrite      bool
	append            bool
	createOnDemand    bool
	permissions       fs.FileMode
	permissionsSet    bool
	maxSize           int64
	interval          time.Duration
	cronExpression    string
	cron              *rolling.CronSchedule
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

// NewRollingFileAppender 创建滚动文件 appender。
func NewRollingFileAppender(path string, options ...RollingFileOption) (*RollingFileAppender, error) {
	cleanPath, err := logfile.ValidatePath(path)
	if err != nil {
		return nil, err
	}
	appender := &RollingFileAppender{
		name:          "rollingFile",
		path:          cleanPath,
		layout:        internallayout.NewDefaultLayout(),
		bufferSize:    internalfileappender.DefaultFileBufferSize,
		append:        true,
		permissions:   logfile.DefaultPermissions,
		maxSize:       DefaultRollingMaxSize,
		maxBackups:    DefaultRollingMaxBackups,
		fileIndexMode: RollingFileIndexNoMax,
		modulate:      true,
		clock:         time.Now,
	}
	for _, option := range options {
		if option != nil {
			option(appender)
		}
	}
	if err := appender.validate(); err != nil {
		return nil, err
	}
	if !appender.permissionsSet && appender.permissions == 0 {
		appender.permissions = logfile.DefaultPermissions
	}
	appender.startActionWorker()
	if !appender.createOnDemand {
		existingSize, err := appender.openAt(appender.now())
		if err != nil {
			_ = appender.closeActionWorker()
			return nil, err
		}
		if appender.rolloverOnStartup && existingSize > 0 {
			if err := appender.rollover(appender.now()); err != nil {
				_ = appender.Close()
				return nil, err
			}
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
	ctx = logevent.NormalizeContext(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}
	if internallayout.RequiresSynchronizedFormatting(a.layout) {
		return a.appendSynchronized(event)
	}
	buf := acquireBuffer()
	defer releaseBuffer(buf)
	if err := a.layout.Format(buf, event); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed {
		return fmt.Errorf("goark-log: rolling file appender %q is closed", a.Name())
	}
	now := event.Time
	if now.IsZero() {
		now = a.now()
	}
	if a.file == nil {
		existingSize, err := a.openAt(now)
		if err != nil {
			return err
		}
		if a.rolloverOnStartup && existingSize > 0 {
			if err := a.rollover(now); err != nil {
				return err
			}
		}
	}
	if a.shouldRollover(now, int64(buf.Len())) {
		if err := a.rollover(now); err != nil {
			return err
		}
	}
	return a.writeBytesLocked(buf.Bytes())
}

func (a *RollingFileAppender) appendSynchronized(event Event) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed {
		return fmt.Errorf("goark-log: rolling file appender %q is closed", a.Name())
	}
	now := event.Time
	if now.IsZero() {
		now = a.now()
	}
	if a.file == nil {
		existingSize, err := a.openAt(now)
		if err != nil {
			return err
		}
		if a.rolloverOnStartup && existingSize > 0 {
			if err := a.rollover(now); err != nil {
				return err
			}
		}
	}
	if a.shouldRollover(now, 0) {
		if err := a.rollover(now); err != nil {
			return err
		}
	}
	buf := acquireBuffer()
	defer releaseBuffer(buf)
	if err := a.layout.Format(buf, event); err != nil {
		return err
	}
	if a.shouldRollover(now, int64(buf.Len())) {
		if err := a.rollover(now); err != nil {
			return err
		}
		buf.Reset()
		if err := a.layout.Format(buf, event); err != nil {
			return err
		}
	}
	return a.writeBytesLocked(buf.Bytes())
}

func (a *RollingFileAppender) writeBytesLocked(data []byte) error {
	var n int
	var err error
	if a.writer != nil {
		n, err = a.writer.Write(data)
		if err == nil && a.flushOnWrite {
			err = a.writer.Flush()
		}
	} else {
		n, err = a.file.Write(data)
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
	_, footerErr := a.writeFooterLocked()
	flushErr := a.flushLocked()
	if a.file == nil {
		return errors.Join(footerErr, flushErr, a.closeActionWorker())
	}
	err := a.file.Close()
	a.file = nil
	a.writer = nil
	actionErr := a.closeActionWorker()
	return errors.Join(footerErr, flushErr, err, actionErr)
}

func (a *RollingFileAppender) validate() error {
	if strings.TrimSpace(a.name) == "" {
		return fmt.Errorf("goark-log: rolling file appender name is empty")
	}
	if a.layout == nil {
		a.layout = internallayout.NewDefaultLayout()
	}
	a.layout = internallayout.CloneLayout(a.layout)
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
		cron, err := rolling.ParseCronSchedule(a.cronExpression)
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
	switch a.fileIndexMode {
	case "", RollingFileIndexNoMax:
		a.fileIndexMode = RollingFileIndexNoMax
	case RollingFileIndexMax, RollingFileIndexMin:
	default:
		return fmt.Errorf("goark-log: rolling file index mode %q is invalid", a.fileIndexMode)
	}
	for index, action := range a.deleteActions {
		normalized, err := normalizeRollingDeleteAction(action)
		if err != nil {
			return fmt.Errorf("goark-log: rolling delete action %d: %w", index, err)
		}
		a.deleteActions[index] = normalized
	}
	if a.directWrite && strings.TrimSpace(a.filePattern) == "" {
		return fmt.Errorf("goark-log: direct write rollover requires filePattern")
	}
	if a.filePattern != "" {
		if a.maxSize > 0 && !rolling.PatternHasIndex(a.filePattern) {
			return fmt.Errorf("goark-log: rolling filePattern requires %%i when size policy is enabled")
		}
		if strings.HasSuffix(strings.ToLower(a.filePattern), ".gz") {
			a.compress = true
		}
		if a.directWrite && a.compress {
			return fmt.Errorf("goark-log: direct write rollover does not support gzip compression")
		}
		candidate, _, err := a.archivePaths(a.now(), 0)
		if err != nil {
			return err
		}
		if !a.directWrite && filepath.Clean(candidate) == filepath.Clean(a.path) {
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
	_, err := a.openAt(a.now())
	return err
}

func (a *RollingFileAppender) openAt(now time.Time) (int64, error) {
	if a.directWrite {
		if err := a.initArchiveIndex(); err != nil {
			return 0, err
		}
		return a.openDirect(now)
	}
	existingSize, err := a.openActiveLocked()
	if err != nil {
		return 0, err
	}
	a.nextRollover = rolling.NextRolloverAfter(now, a.interval, a.modulate)
	a.nextCron = rolling.NextCronRolloverAfter(now, a.cron)
	if err := a.initArchiveIndex(); err != nil {
		_ = a.file.Close()
		a.file = nil
		a.writer = nil
		return 0, err
	}
	return existingSize, nil
}

func (a *RollingFileAppender) openDirect(now time.Time) (int64, error) {
	target, err := a.nextArchivePath(now)
	if err != nil {
		return 0, err
	}
	file, err := logfile.OpenWithOptions(target, a.openOptions())
	if err != nil {
		return 0, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return 0, fmt.Errorf("goark-log: stat log file %q: %w", target, err)
	}
	existingSize := info.Size()
	a.path = target
	a.file = file
	if a.bufferSize > 0 {
		a.writer = bufio.NewWriterSize(file, a.bufferSize)
	}
	a.size = existingSize
	if a.size == 0 {
		n, err := a.writeHeaderLocked()
		if err != nil {
			_ = a.flushLocked()
			_ = file.Close()
			a.file = nil
			a.writer = nil
			return 0, fmt.Errorf("goark-log: write rolling file appender %q header: %w", a.Name(), err)
		}
		a.size += int64(n)
	}
	a.nextRollover = rolling.NextRolloverAfter(now, a.interval, a.modulate)
	a.nextCron = rolling.NextCronRolloverAfter(now, a.cron)
	return existingSize, nil
}

func (a *RollingFileAppender) openActiveLocked() (int64, error) {
	file, err := logfile.OpenWithOptions(a.path, a.openOptions())
	if err != nil {
		return 0, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return 0, fmt.Errorf("goark-log: stat log file %q: %w", a.path, err)
	}
	existingSize := info.Size()
	a.file = file
	if a.bufferSize > 0 {
		a.writer = bufio.NewWriterSize(file, a.bufferSize)
	}
	a.size = existingSize
	if a.size == 0 {
		n, err := a.writeHeaderLocked()
		if err != nil {
			_ = a.flushLocked()
			_ = file.Close()
			a.file = nil
			a.writer = nil
			return 0, fmt.Errorf("goark-log: write rolling file appender %q header: %w", a.Name(), err)
		}
		a.size += int64(n)
	}
	return existingSize, nil
}

func (a *RollingFileAppender) openOptions() logfile.OpenOptions {
	return logfile.OpenOptions{
		Append:         a.append,
		Permissions:    a.permissions,
		PermissionsSet: a.permissionsSet,
	}
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
	if err := errors.Join(a.writeFooterErrorLocked(), a.flushLocked()); err != nil {
		return fmt.Errorf("goark-log: flush active log file %q: %w", a.path, err)
	}
	if a.file != nil {
		if err := a.file.Close(); err != nil {
			return fmt.Errorf("goark-log: close active log file %q: %w", a.path, err)
		}
		a.file = nil
		a.writer = nil
	}
	if a.directWrite {
		if _, err := a.openDirect(now); err != nil {
			return err
		}
		a.nextRollover = rolling.NextRolloverAfter(now, a.interval, a.modulate)
		a.nextCron = rolling.NextCronRolloverAfter(now, a.cron)
		return a.runDeleteActions(now)
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
	if _, err := a.openActiveLocked(); err != nil {
		return err
	}
	a.nextRollover = rolling.NextRolloverAfter(now, a.interval, a.modulate)
	a.nextCron = rolling.NextCronRolloverAfter(now, a.cron)
	return a.runRolloverActions(now, target, archiveIndex)
}
