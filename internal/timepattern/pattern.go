package timepattern

import (
	"strings"
	"time"
)

// DefaultLayout 是 goark-log 统一的默认时间格式。
const DefaultLayout = "2006-01-02T15:04:05.000Z07:00"

// UnixMode 标识时间格式是否需要输出 Unix 时间戳。
type UnixMode uint8

const (
	UnixNone UnixMode = iota
	UnixSeconds
	UnixMillis
	UnixMicros
	UnixNanos
)

// Normalize 把 Log4j/Java 风格时间格式映射为 Go 时间布局或 Unix 输出模式。
func Normalize(format string) (string, UnixMode) {
	switch strings.ToUpper(strings.TrimSpace(format)) {
	case "", "DEFAULT", "ISO8601", "ISO8601_OFFSET_DATE_TIME":
		return DefaultLayout, UnixNone
	case "RFC3339":
		return time.RFC3339, UnixNone
	case "RFC3339NANO":
		return time.RFC3339Nano, UnixNone
	case "UNIX", "UNIX_SECONDS":
		return "", UnixSeconds
	case "UNIX_MILLIS", "UNIX_MS":
		return "", UnixMillis
	case "UNIX_MICROS", "UNIX_US":
		return "", UnixMicros
	case "UNIX_NANOS", "UNIX_NS":
		return "", UnixNanos
	default:
		return JavaToGo(format), UnixNone
	}
}

// JavaToGo 把常用 Java 日期占位符转换为 Go reference time 布局。
func JavaToGo(format string) string {
	replacer := strings.NewReplacer(
		"yyyy", "2006",
		"yy", "06",
		"MM", "01",
		"dd", "02",
		"HH", "15",
		"mm", "04",
		"ss", "05",
		"SSSSSS", "000000",
		"SSS", "000",
		"XXX", "Z07:00",
		"XX", "-0700",
		"X", "-07",
	)
	return replacer.Replace(format)
}
