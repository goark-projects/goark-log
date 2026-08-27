package layoutsupport

import (
	"os"
	"strings"
	"time"
)

var hostName = resolveHostName()

// HostName 返回进程启动时解析到的主机名。
func HostName() string {
	return hostName
}

// EventTime 返回事件时间；零值事件使用当前时间。
func EventTime(when time.Time) time.Time {
	if when.IsZero() {
		return time.Now()
	}
	return when
}

func resolveHostName() string {
	name, err := os.Hostname()
	if err != nil || strings.TrimSpace(name) == "" {
		return "localhost"
	}
	return strings.TrimSpace(name)
}
