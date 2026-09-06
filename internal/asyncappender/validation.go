package asyncappender

import (
	"fmt"
	"strings"

	"goark.dev/log/internal/asyncruntime"
)

func (a *Appender) validate(appenders []Sink) error {
	if strings.TrimSpace(a.name) == "" {
		return fmt.Errorf("goark-log: async appender name is empty")
	}
	if a.queueSize <= 0 {
		return fmt.Errorf("goark-log: async queue size must be > 0")
	}
	if a.batchSize <= 0 {
		return fmt.Errorf("goark-log: async appender batch size must be > 0")
	}
	if _, err := asyncruntime.ParseOverflowStrategy(string(a.strategy)); err != nil {
		return err
	}
	if err := asyncruntime.ValidateWaitOptions(a.waitOptions); err != nil {
		return err
	}
	if len(appenders) == 0 {
		return fmt.Errorf("goark-log: async appender requires at least one delegate appender")
	}
	for _, appender := range appenders {
		if appender == nil {
			return fmt.Errorf("goark-log: async delegate appender is nil")
		}
	}
	return nil
}
