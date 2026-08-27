package goarklog

import (
	"bytes"
	"io"
	"strings"
)

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
