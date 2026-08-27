package goarklog

import (
	"time"

	"goark.dev/log/internal/configvalue"
	"goark.dev/log/internal/textutil"
)

// ParseByteSize 解析日志滚动大小。
func ParseByteSize(value string) (int64, error) {
	return configvalue.ByteSize(value)
}

// ParseRollingInterval 解析时间滚动间隔。
func ParseRollingInterval(value string) (time.Duration, error) {
	return configvalue.RollingInterval(value)
}

// ParseRollingMaxAge 解析滚动档案最大保留时间。
func ParseRollingMaxAge(value string) (time.Duration, error) {
	return configvalue.RollingMaxAge(value)
}

// ParseMonitorInterval 解析配置监控间隔；纯数字按秒处理。
func ParseMonitorInterval(value string) (time.Duration, error) {
	return configvalue.MonitorInterval(value)
}

func (c *fileConfig) monitorInterval() (time.Duration, error) {
	if c == nil {
		return 0, nil
	}
	return ParseMonitorInterval(textutil.FirstNonBlank(c.MonitorInterval, c.MonitorKebab))
}
