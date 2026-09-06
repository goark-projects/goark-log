package rollingfile

import (
	"bufio"
	"fmt"
	"time"

	"goark.dev/log/internal/logfile"
	"goark.dev/log/internal/rolling"
)

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
