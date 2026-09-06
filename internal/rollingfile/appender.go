package rollingfile

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
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
	totalSizeCap      int64
	cleanOnStart      bool
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
func NewRollingFileAppender(
	path string,
	options ...RollingFileOption,
) (*RollingFileAppender, error) {
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
	if appender.cleanOnStart {
		if err := appender.deleteExpiredArchives(appender.now()); err != nil {
			return nil, fmt.Errorf("goark-log: clean rolling history on start: %w", err)
		}
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

func (a *RollingFileAppender) now() time.Time {
	return a.clock()
}
