package goarklog

import (
	"log/slog"
	"time"

	"goark.dev/log/internal/configvalue"
	configlevel "goark.dev/log/internal/level"
)

const (
	// LevelAll 表示最低阈值，配置为 ALL 时允许所有事件进入日志管线。
	LevelAll   = configlevel.All
	LevelTrace = configlevel.Trace
	// LevelFatal 表示比 ERROR 更高的致命级别。
	LevelFatal = configlevel.Fatal
	// LevelOff 表示最高阈值，配置为 OFF 时关闭普通日志事件。
	LevelOff = configlevel.Off
)

// LevelRegistry 保存日志级别名称和数值的双向映射。
type LevelRegistry = configlevel.Registry

// NewLevelRegistry 创建包含内置级别的注册表。
func NewLevelRegistry() *LevelRegistry {
	return configlevel.NewRegistry()
}

// DefaultLevelRegistry 返回进程默认级别注册表。
func DefaultLevelRegistry() *LevelRegistry {
	return configlevel.DefaultRegistry()
}

// RegisterLevel 向默认注册表注册自定义级别。
func RegisterLevel(name string, level slog.Level) error {
	return configlevel.RegisterDefault(name, level)
}

// ParseLevel 解析日志级别名称。
func ParseLevel(value string) (slog.Level, error) {
	return configlevel.ParseDefault(value)
}

// LevelName 返回级别名称，优先返回已注册的精确名称。
func LevelName(level slog.Level) string {
	return configlevel.NameDefault(level)
}

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
