package goarklog

import (
	"bytes"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"goark.dev/log/internal/timepattern"
)

const (
	// DefaultSpringBootPattern 是默认控制台输出格式，风格对齐 Spring Boot。
	DefaultSpringBootPattern = "%d %5level %pid --- [%thread] %logger : %msg%attrs%n"
	defaultTimeFormat        = timepattern.DefaultLayout
)

var processIDString = strconv.Itoa(os.Getpid())
var patternSequence atomic.Uint64
var patternStartTime = time.Now()

// Layout 把日志事件编码为字节。
type Layout interface {
	Format(buf *bytes.Buffer, event Event) error
}

// NewDefaultLayout 创建默认 Spring Boot 风格布局。
func NewDefaultLayout() Layout {
	layout, _ := NewPatternLayout(DefaultSpringBootPattern)
	return layout
}

// TextLayout 输出稳定的 key=value 文本。
type TextLayout struct{}

func (TextLayout) Format(buf *bytes.Buffer, event Event) error {
	appendKey(buf, "time")
	buf.Write(event.Time.AppendFormat(buf.AvailableBuffer(), defaultTimeFormat))
	appendKeyValue(buf, "level", levelName(event.Level))
	appendKeyValue(buf, "logger", event.Logger)
	appendKeyValue(buf, "msg", event.Message)
	for _, attr := range event.Attrs {
		appendKeyValueAttr(buf, attr.Key, attr.Value)
	}
	buf.WriteByte('\n')
	return nil
}
