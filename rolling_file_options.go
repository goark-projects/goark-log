package goarklog

import (
	"errors"
	"io/fs"
	"strings"
	"time"

	"goark.dev/log/internal/rolling"
)

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

// WithRollingFileAppend 设置打开活动文件时是否追加到已有内容。
func WithRollingFileAppend(enabled bool) RollingFileOption {
	return func(appender *RollingFileAppender) {
		appender.append = enabled
	}
}

// WithRollingFileCreateOnDemand 设置是否延迟到首次写入时创建活动文件。
func WithRollingFileCreateOnDemand(enabled bool) RollingFileOption {
	return func(appender *RollingFileAppender) {
		appender.createOnDemand = enabled
	}
}

// WithRollingFilePermissions 设置新建活动文件的权限。
func WithRollingFilePermissions(permissions fs.FileMode) RollingFileOption {
	return func(appender *RollingFileAppender) {
		appender.permissions = permissions.Perm()
		appender.permissionsSet = true
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

// WithRollingFileIndexMode 设置滚动索引分配策略。
func WithRollingFileIndexMode(mode RollingFileIndexMode) RollingFileOption {
	return func(appender *RollingFileAppender) {
		appender.fileIndexMode = mode
	}
}

// WithRollingDirectWrite 设置是否直接写入 filePattern 指向的滚动文件。
func WithRollingDirectWrite(enabled bool) RollingFileOption {
	return func(appender *RollingFileAppender) {
		appender.directWrite = enabled
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
			if _, err := rolling.CompressFileTo(target, compressedTarget); err != nil {
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
	// 异步滚动动作必须由同一个 worker 串行执行；队列满时阻塞写线程施加背压，
	// 避免压缩和保留删除并发操作同一批归档文件。
	queue <- action
	return nil
}

func (a *RollingFileAppender) runDeleteActions(now time.Time) error {
	var joined error
	for _, action := range a.deleteActions {
		joined = errors.Join(joined, rolling.DeleteArchivesByAction(now, toRollingDeleteAction(action)))
	}
	return joined
}

func normalizeRollingDeleteAction(action RollingDeleteAction) (RollingDeleteAction, error) {
	normalized, err := rolling.NormalizeDeleteAction(toRollingDeleteAction(action))
	if err != nil {
		return RollingDeleteAction{}, err
	}
	return RollingDeleteAction{
		BasePath: normalized.BasePath,
		MaxDepth: normalized.MaxDepth,
		Glob:     normalized.Glob,
		MaxAge:   normalized.MaxAge,
		MaxCount: normalized.MaxCount,
		MaxSize:  normalized.MaxSize,
	}, nil
}

func toRollingDeleteAction(action RollingDeleteAction) rolling.DeleteAction {
	return rolling.DeleteAction{
		BasePath: action.BasePath,
		MaxDepth: action.MaxDepth,
		Glob:     action.Glob,
		MaxAge:   action.MaxAge,
		MaxCount: action.MaxCount,
		MaxSize:  action.MaxSize,
	}
}
