package goarklog

import (
	"errors"
	"time"
)

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
	// 异步滚动动作必须由同一个 worker 串行执行；队列满时阻塞写线程施加背压，
	// 避免压缩和保留删除并发操作同一批归档文件。
	queue <- action
	return nil
}
