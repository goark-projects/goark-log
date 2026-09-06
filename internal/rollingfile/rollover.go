package rollingfile

import (
	"errors"
	"fmt"
	"os"
	"time"

	"goark.dev/log/internal/rolling"
)

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
