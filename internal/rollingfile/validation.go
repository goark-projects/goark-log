package rollingfile

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	internallayout "goark.dev/log/internal/layout"
	"goark.dev/log/internal/rolling"
)

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
			return fmt.Errorf(
				"goark-log: rolling cron schedule %q is invalid: %w",
				a.cronExpression,
				err,
			)
		}
		a.cron = cron
	}
	if a.maxBackups < 0 {
		return fmt.Errorf("goark-log: rolling max backups must be >= 0")
	}
	if a.maxAge < 0 {
		return fmt.Errorf("goark-log: rolling max age must be >= 0")
	}
	if a.totalSizeCap < 0 {
		return fmt.Errorf("goark-log: rolling total size cap must be >= 0")
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
			return fmt.Errorf(
				"goark-log: rolling filePattern requires %%i when size policy is enabled",
			)
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
			return fmt.Errorf(
				"goark-log: rolling filePattern must not resolve to active log file %q",
				a.path,
			)
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
