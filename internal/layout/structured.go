package layout

import (
	"bytes"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"

	"goark.dev/log/internal/logvalue"
)

// JSONLayout 输出 JSON 事件。
type JSONLayout struct {
	options LayoutOptions
	state   *jsonLayoutState
}

// NewJSONLayout 创建可配置 JSON 布局。
func NewJSONLayout(options LayoutOptions) JSONLayout {
	layout := JSONLayout{options: options}
	if options.Complete {
		layout.state = &jsonLayoutState{}
	}
	return layout
}

func (l JSONLayout) Format(buf *bytes.Buffer, event Event) error {
	appendJSONCompleteSeparator(buf, l.options, l.state)
	appendJSONLayoutEvent(buf, event, l.options)
	return nil
}

func (l JSONLayout) AppendHeader(buf *bytes.Buffer) error {
	appendJSONCompleteHeader(buf, l.options, l.state)
	return nil
}

func (l JSONLayout) AppendFooter(buf *bytes.Buffer) error {
	appendJSONCompleteFooter(buf, l.options)
	return nil
}

// CloneLayout 为每个 appender 隔离 Complete 模式的事件计数。
func (l JSONLayout) CloneLayout() Layout {
	cloned := l
	if cloned.options.Complete {
		cloned.state = &jsonLayoutState{}
	} else {
		cloned.state = nil
	}
	return cloned
}

// RequiresSynchronizedFormatting 说明 Complete JSON 流需要按实际写入顺序生成分隔符。
func (l JSONLayout) RequiresSynchronizedFormatting() bool {
	return l.options.Complete
}

// Options 返回布局输出参数快照。
func (l JSONLayout) Options() LayoutOptions {
	return l.options
}

type jsonLayoutState struct {
	events atomic.Uint64
}

func appendJSONCompleteSeparator(buf *bytes.Buffer, options LayoutOptions, state *jsonLayoutState) {
	if !options.Complete || state == nil || state.events.Add(1) <= 1 {
		return
	}
	buf.WriteByte(',')
	if options.EventEOL || !options.Compact {
		buf.WriteByte('\n')
	}
}

func appendJSONCompleteHeader(buf *bytes.Buffer, options LayoutOptions, state *jsonLayoutState) {
	if state != nil {
		state.events.Store(0)
	}
	if !options.Complete {
		return
	}
	header := options.Header
	if strings.TrimSpace(header) == "" {
		header = "["
	}
	buf.WriteString(header)
	if options.EventEOL || !options.Compact {
		buf.WriteByte('\n')
	}
}

func appendJSONCompleteFooter(buf *bytes.Buffer, options LayoutOptions) {
	if !options.Complete {
		return
	}
	footer := options.Footer
	if strings.TrimSpace(footer) == "" {
		footer = "]"
	}
	buf.WriteString(footer)
}

// AppendJSONEvent 用零额外状态编码 JSON 单行事件。
func AppendJSONEvent(buf *bytes.Buffer, when time.Time, level slog.Level, logger string, message string, attrs []slog.Attr) {
	buf.WriteByte('{')
	logvalue.AppendJSONFieldTime(buf, "time", when, defaultTimeFormat, false)
	logvalue.AppendJSONFieldString(buf, "level", levelName(level), true)
	logvalue.AppendJSONFieldString(buf, "logger", logger, true)
	logvalue.AppendJSONFieldString(buf, "msg", message, true)
	for _, attr := range attrs {
		logvalue.AppendJSONFieldValue(buf, attr.Key, attr.Value, true)
	}
	buf.WriteString("}\n")
}

func appendJSONLayoutEvent(buf *bytes.Buffer, event Event, options LayoutOptions) {
	buf.WriteByte('{')
	logvalue.AppendJSONFieldTime(buf, "time", event.Time, defaultTimeFormat, false)
	logvalue.AppendJSONFieldString(buf, "level", levelName(event.Level), true)
	logvalue.AppendJSONFieldString(buf, "logger", event.Logger, true)
	logvalue.AppendJSONFieldString(buf, "msg", event.Message, true)
	if options.PropertiesAsList {
		logvalue.AppendJSONAttrsListField(buf, "contextMap", event.Attrs, true)
	} else {
		for _, attr := range event.Attrs {
			logvalue.AppendJSONFieldValue(buf, attr.Key, attr.Value, true)
		}
	}
	if event.Throwable != nil && (options.IncludeStacktrace || options.StacktraceAsString) {
		logvalue.AppendJSONKey(buf, "thrown", true)
		if options.StacktraceAsString {
			logvalue.AppendJSONString(buf, throwableStackString(event.Throwable))
		} else {
			appendThrowableJSON(buf, event.Throwable)
		}
	}
	buf.WriteByte('}')
	appendLayoutTerminator(buf, options)
}

// AppendJSONFixedEvent 编码最多三个属性的固定数组事件，避免热路径切片分配。
func AppendJSONFixedEvent(buf *bytes.Buffer, when time.Time, level slog.Level, logger string, message string, attrs [3]slog.Attr, count int) {
	buf.WriteByte('{')
	logvalue.AppendJSONFieldTime(buf, "time", when, defaultTimeFormat, false)
	logvalue.AppendJSONFieldString(buf, "level", levelName(level), true)
	logvalue.AppendJSONFieldString(buf, "logger", logger, true)
	logvalue.AppendJSONFieldString(buf, "msg", message, true)
	for index := 0; index < count && index < len(attrs); index++ {
		attr := attrs[index]
		logvalue.AppendJSONFieldValue(buf, attr.Key, attr.Value, true)
	}
	buf.WriteString("}\n")
}
