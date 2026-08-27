package goarklog

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
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

// NewRollingFileAppender 创建滚动文件 appender。
func NewRollingFileAppender(path string, options ...RollingFileOption) (*RollingFileAppender, error) {
	cleanPath, err := validateLogFilePath(path)
	if err != nil {
		return nil, err
	}
	appender := &RollingFileAppender{
		name:          "rollingFile",
		path:          cleanPath,
		layout:        NewDefaultLayout(),
		bufferSize:    DefaultFileBufferSize,
		append:        true,
		permissions:   defaultLogFilePermissions,
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
		appender.permissions = defaultLogFilePermissions
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
	ctx = normalizeContext(ctx)
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
		if a.maxSize > 0 && !rollingPatternHasIndex(a.filePattern) {
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
