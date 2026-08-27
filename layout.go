package goarklog

import (
	"bytes"
	"io"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"goark.dev/log/internal/logvalue"
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

// LayoutOptions 描述通用结构化布局参数。
type LayoutOptions struct {
	// Compact 禁用默认事件换行，适合由上层协议自行分隔事件的场景。
	Compact bool
	// EventEOL 在 Compact 模式下仍然为每个事件追加换行。
	EventEOL bool
	// Complete 启用布局页眉和页脚输出，由 appender 在流生命周期内写一次。
	Complete bool
	// IncludeStacktrace 在支持异常结构的布局中输出完整异常结构。
	IncludeStacktrace bool
	// StacktraceAsString 将异常栈输出为字符串，便于兼容文本型采集器。
	StacktraceAsString bool
	// PropertiesAsList 将上下文属性输出为键值列表。
	PropertiesAsList bool
	// IncludeNullDelimiter 在事件结束后追加 NUL 字节，用于 GELF 等协议分隔。
	IncludeNullDelimiter bool
	// DisableANSI 禁用 PatternLayout 中 highlight/style 转换器的 ANSI SGR 输出。
	DisableANSI bool
	// Header 是 Complete 模式下流打开时写入的页眉。
	Header string
	// Footer 是 Complete 模式下流关闭时写入的页脚。
	Footer string
}

type lifecycleLayout interface {
	AppendHeader(buf *bytes.Buffer) error
	AppendFooter(buf *bytes.Buffer) error
}

// NewDefaultLayout 创建默认 Spring Boot 风格布局。
func NewDefaultLayout() Layout {
	layout, _ := NewPatternLayout(DefaultSpringBootPattern)
	return layout
}

// TextLayout 输出稳定的 key=value 文本。
type TextLayout struct{}

func (TextLayout) Format(buf *bytes.Buffer, event Event) error {
	logvalue.AppendKey(buf, "time")
	buf.Write(event.Time.AppendFormat(buf.AvailableBuffer(), defaultTimeFormat))
	logvalue.AppendKeyValue(buf, "level", levelName(event.Level))
	logvalue.AppendKeyValue(buf, "logger", event.Logger)
	logvalue.AppendKeyValue(buf, "msg", event.Message)
	for _, attr := range event.Attrs {
		logvalue.AppendKeyValueAttr(buf, attr.Key, attr.Value)
	}
	buf.WriteByte('\n')
	return nil
}

func appendLayoutHeader(buf *bytes.Buffer, options LayoutOptions) {
	if options.Complete && strings.TrimSpace(options.Header) != "" {
		buf.WriteString(options.Header)
	}
}

func appendLayoutFooter(buf *bytes.Buffer, options LayoutOptions) {
	if options.Complete && strings.TrimSpace(options.Footer) != "" {
		buf.WriteString(options.Footer)
	}
}

func appendLayoutTerminator(buf *bytes.Buffer, options LayoutOptions) {
	if options.EventEOL || !options.Compact {
		buf.WriteByte('\n')
	}
	if options.IncludeNullDelimiter {
		buf.WriteByte(0)
	}
}

func writeLayoutHeader(writer io.Writer, layout Layout) (int, error) {
	lifecycle, ok := layout.(lifecycleLayout)
	if !ok {
		return 0, nil
	}
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer releaseBuffer(buf)
	if err := lifecycle.AppendHeader(buf); err != nil {
		return 0, err
	}
	if buf.Len() == 0 {
		return 0, nil
	}
	return writer.Write(buf.Bytes())
}

func writeLayoutFooter(writer io.Writer, layout Layout) (int, error) {
	lifecycle, ok := layout.(lifecycleLayout)
	if !ok {
		return 0, nil
	}
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer releaseBuffer(buf)
	if err := lifecycle.AppendFooter(buf); err != nil {
		return 0, err
	}
	if buf.Len() == 0 {
		return 0, nil
	}
	return writer.Write(buf.Bytes())
}
