package goarklog

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseMonitorInterval 解析配置监控间隔；纯数字按秒处理。
func ParseMonitorInterval(value string) (time.Duration, error) {
	text := strings.ToLower(strings.TrimSpace(value))
	switch text {
	case "", "0", "off", "none", "disabled", "false":
		return 0, nil
	}
	if seconds, err := strconv.ParseFloat(text, 64); err == nil {
		if seconds < 0 {
			return 0, fmt.Errorf("goark-log: monitorInterval must be >= 0")
		}
		return time.Duration(seconds * float64(time.Second)), nil
	}
	interval, err := time.ParseDuration(text)
	if err != nil {
		return 0, fmt.Errorf("goark-log: invalid monitorInterval %q", value)
	}
	if interval < 0 {
		return 0, fmt.Errorf("goark-log: monitorInterval must be >= 0")
	}
	return interval, nil
}

func (c *fileConfig) monitorInterval() (time.Duration, error) {
	if c == nil {
		return 0, nil
	}
	return ParseMonitorInterval(firstNonBlank(c.MonitorInterval, c.MonitorKebab))
}
