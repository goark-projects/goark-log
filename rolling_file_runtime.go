package goarklog

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"time"
)

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
	a.nextRollover = nextRolloverAfter(now, a.interval, a.modulate)
	a.nextCron = nextCronRolloverAfter(now, a.cron)
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
	file, err := openLogFileWithOptions(target, a.openOptions())
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
	a.nextRollover = nextRolloverAfter(now, a.interval, a.modulate)
	a.nextCron = nextCronRolloverAfter(now, a.cron)
	return existingSize, nil
}

func (a *RollingFileAppender) openActiveLocked() (int64, error) {
	file, err := openLogFileWithOptions(a.path, a.openOptions())
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

func (a *RollingFileAppender) openOptions() logFileOpenOptions {
	return logFileOpenOptions{
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
		a.nextRollover = nextRolloverAfter(now, a.interval, a.modulate)
		a.nextCron = nextCronRolloverAfter(now, a.cron)
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

func (a *RollingFileAppender) writeHeaderLocked() (int, error) {
	writer := a.outputWriterLocked()
	if writer == nil {
		return 0, nil
	}
	return writeLayoutHeader(writer, a.layout)
}

func (a *RollingFileAppender) writeFooterLocked() (int, error) {
	writer := a.outputWriterLocked()
	if writer == nil {
		return 0, nil
	}
	return writeLayoutFooter(writer, a.layout)
}

func (a *RollingFileAppender) writeFooterErrorLocked() error {
	_, err := a.writeFooterLocked()
	return err
}

func (a *RollingFileAppender) outputWriterLocked() io.Writer {
	if a == nil {
		return nil
	}
	if a.writer != nil {
		return a.writer
	}
	return a.file
}
