package goarklog

import (
	"bytes"
	"strconv"
	"strings"
)

// GELFLayout 输出 Graylog Extended Log Format 单行 JSON。
type GELFLayout struct {
	options LayoutOptions
}

// NewGELFLayout 创建可配置 GELF 布局。
func NewGELFLayout(options LayoutOptions) GELFLayout {
	return GELFLayout{options: options}
}

// Format 把事件编码为 GELF JSON。
func (l GELFLayout) Format(buf *bytes.Buffer, event Event) error {
	when := eventTime(event.Time)
	buf.WriteByte('{')
	appendJSONFieldString(buf, "version", "1.1", false)
	appendJSONFieldString(buf, "host", hostNameString, true)
	appendJSONFieldString(buf, "short_message", event.Message, true)
	if thrown := gelfThrowableString(event, l.options); thrown != "" {
		appendJSONFieldString(buf, "full_message", thrown, true)
	}
	appendJSONKey(buf, "timestamp", true)
	buf.Write(strconv.AppendFloat(buf.AvailableBuffer(), float64(when.UnixNano())/1e9, 'f', 6, 64))
	appendJSONKey(buf, "level", true)
	buf.Write(strconv.AppendInt(buf.AvailableBuffer(), int64(syslogSeverity(event.Level)), 10))
	appendJSONFieldString(buf, "_logger", event.Logger, true)
	appendJSONFieldString(buf, "_thread", eventThreadName(event), true)
	if marker := eventMarkerString(event); marker != "" {
		appendJSONFieldString(buf, "_marker", marker, true)
	}
	for _, attr := range event.Attrs {
		key := gelfAdditionalFieldKey(attr.Key)
		if key == "" {
			continue
		}
		appendJSONFieldValue(buf, key, attr.Value, true)
	}
	buf.WriteByte('}')
	appendLayoutTerminator(buf, l.options)
	return nil
}

func (l GELFLayout) AppendHeader(buf *bytes.Buffer) error {
	appendLayoutHeader(buf, l.options)
	return nil
}

func (l GELFLayout) AppendFooter(buf *bytes.Buffer) error {
	appendLayoutFooter(buf, l.options)
	return nil
}

func gelfThrowableString(event Event, options LayoutOptions) string {
	if event.Throwable == nil {
		return eventErrorString(event)
	}
	if options.StacktraceAsString || options.IncludeStacktrace {
		return throwableStackString(event.Throwable)
	}
	return event.Throwable.String()
}

func gelfAdditionalFieldKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" || key == "id" || strings.HasPrefix(key, "_") {
		return ""
	}
	return "_" + key
}
