package testsupport

import (
	"log/slog"
	"time"

	"goark.dev/log"
)

// BenchmarkEvent 创建稳定的基准事件。
func BenchmarkEvent() Event {
	event := TestEvent("service started", FixedTestTime())
	event.Logger = "goark.bench"
	event.Attrs = []slog.Attr{
		slog.String("profile", "bench"),
		slog.Int("index", 42),
		slog.Duration("elapsed", 10*time.Millisecond),
	}
	return event
}

// TestEvent 创建具有稳定字段的日志测试事件。
func TestEvent(message string, when time.Time) Event {
	return Event{
		Time:       when,
		Level:      0,
		Message:    message,
		Logger:     "goark.test",
		ThreadName: DefaultThreadName,
	}
}

// FixedTestTime 返回测试使用的固定时间。
func FixedTestTime() time.Time {
	location := time.FixedZone("CST", 8*3600)
	return time.Date(2026, 8, 25, 10, 15, 30, 123000000, location)
}

// LevelPointer 返回日志级别指针。
func LevelPointer(level slog.Level) *slog.Level {
	return &level
}

// MarkerPointer 返回标记指针。
func MarkerPointer(marker log.Marker) *log.Marker {
	return &marker
}
