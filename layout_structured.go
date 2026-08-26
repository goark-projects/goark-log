package goarklog

import (
	"bytes"
	"encoding/xml"
	"time"
)

// XMLLayout 输出单事件 XML 片段。
type XMLLayout struct{}

// Format 把事件编码为 XML。
func (XMLLayout) Format(buf *bytes.Buffer, event Event) error {
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
	if thrown := eventErrorString(event); thrown != "" {
		appendXMLElement(buf, "Throwable", thrown)
	}
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
			appendXMLText(buf, attrValueString(attr.Value))
			buf.WriteString("</Entry>")
		}
		buf.WriteString("</ContextMap>")
	}
	buf.WriteString("</Event>\n")
	return nil
}

// CSVLayout 输出单行 CSV，字段顺序固定。
type CSVLayout struct{}

// Format 把事件编码为 CSV。
func (CSVLayout) Format(buf *bytes.Buffer, event Event) error {
	appendCSVField(buf, eventTime(event.Time).Format(defaultTimeFormat), false)
	appendCSVField(buf, levelName(event.Level), true)
	appendCSVField(buf, event.Logger, true)
	appendCSVField(buf, eventThreadName(event), true)
	appendCSVField(buf, event.Message, true)
	if len(event.Attrs) == 0 {
		buf.WriteByte('\n')
		return nil
	}
	var attrs bytes.Buffer
	appendPatternAttrs(&attrs, event.Attrs)
	appendCSVField(buf, attrs.String(), true)
	buf.WriteByte('\n')
	return nil
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

func appendCSVField(buf *bytes.Buffer, value string, comma bool) {
	if comma {
		buf.WriteByte(',')
	}
	if !csvNeedsQuote(value) {
		buf.WriteString(value)
		return
	}
	buf.WriteByte('"')
	for _, r := range value {
		if r == '"' {
			buf.WriteString(`""`)
			continue
		}
		buf.WriteRune(r)
	}
	buf.WriteByte('"')
}

func csvNeedsQuote(value string) bool {
	for _, r := range value {
		switch r {
		case ',', '"', '\r', '\n':
			return true
		}
	}
	return value == ""
}

func eventTime(when time.Time) time.Time {
	if when.IsZero() {
		return time.Now()
	}
	return when
}
