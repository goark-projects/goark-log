package goarklog

import (
	"bytes"
	"encoding/xml"

	"goark.dev/log/internal/logvalue"
)

// XMLLayout 输出单事件 XML 片段。
type XMLLayout struct {
	options LayoutOptions
}

// NewXMLLayout 创建可配置 XML 布局。
func NewXMLLayout(options LayoutOptions) XMLLayout {
	return XMLLayout{options: options}
}

// Format 把事件编码为 XML。
func (l XMLLayout) Format(buf *bytes.Buffer, event Event) error {
	buf.WriteString("<Event")
	appendXMLAttr(buf, "time", eventTime(event.Time).Format(defaultTimeFormat))
	appendXMLAttr(buf, "level", levelName(event.Level))
	appendXMLAttr(buf, "logger", event.Logger)
	appendXMLAttr(buf, "thread", eventThreadName(event))
	if event.EndOfBatch {
		appendXMLAttr(buf, "endOfBatch", "true")
	}
	buf.WriteByte('>')
	appendXMLElement(buf, "Message", event.Message)
	if marker := eventMarkerString(event); marker != "" {
		appendXMLElement(buf, "Marker", marker)
	}
	appendXMLThrowable(buf, l.options, event)
	if len(event.ContextStack) > 0 {
		buf.WriteString("<ContextStack>")
		for _, value := range event.ContextStack {
			appendXMLElement(buf, "Value", value)
		}
		buf.WriteString("</ContextStack>")
	}
	if len(event.Attrs) > 0 {
		buf.WriteString("<ContextMap>")
		for _, attr := range event.Attrs {
			buf.WriteString("<Entry")
			appendXMLAttr(buf, "key", attr.Key)
			buf.WriteByte('>')
			appendXMLText(buf, logvalue.String(attr.Value))
			buf.WriteString("</Entry>")
		}
		buf.WriteString("</ContextMap>")
	}
	buf.WriteString("</Event>")
	appendLayoutTerminator(buf, l.options)
	return nil
}

func (l XMLLayout) AppendHeader(buf *bytes.Buffer) error {
	appendLayoutHeader(buf, l.options)
	return nil
}

func (l XMLLayout) AppendFooter(buf *bytes.Buffer) error {
	appendLayoutFooter(buf, l.options)
	return nil
}

func appendXMLThrowable(buf *bytes.Buffer, options LayoutOptions, event Event) {
	if event.Throwable == nil {
		if thrown := eventErrorString(event); thrown != "" {
			appendXMLElement(buf, "Throwable", thrown)
		}
		return
	}
	if options.StacktraceAsString {
		appendXMLElement(buf, "Throwable", throwableStackString(event.Throwable))
		return
	}
	appendXMLElement(buf, "Throwable", event.Throwable.String())
	if !options.IncludeStacktrace || len(event.Throwable.Stack) == 0 {
		return
	}
	buf.WriteString("<StackTrace>")
	for _, frame := range event.Throwable.Stack {
		appendXMLElement(buf, "Frame", frame)
	}
	buf.WriteString("</StackTrace>")
}

func appendXMLElement(buf *bytes.Buffer, name string, value string) {
	buf.WriteByte('<')
	buf.WriteString(name)
	buf.WriteByte('>')
	appendXMLText(buf, value)
	buf.WriteString("</")
	buf.WriteString(name)
	buf.WriteByte('>')
}

func appendXMLAttr(buf *bytes.Buffer, key string, value string) {
	buf.WriteByte(' ')
	buf.WriteString(key)
	buf.WriteString("=\"")
	appendXMLText(buf, value)
	buf.WriteByte('"')
}

func appendXMLText(buf *bytes.Buffer, value string) {
	_ = xml.EscapeText(buf, []byte(value))
}
