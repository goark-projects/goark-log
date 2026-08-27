package goarklog

import (
	"bytes"

	"goark.dev/log/internal/layoutsupport"
	"goark.dev/log/internal/logvalue"
)

// CSVLayout 输出单行 CSV，字段顺序固定。
type CSVLayout struct {
	options LayoutOptions
}

// NewCSVLayout 创建可配置 CSV 布局。
func NewCSVLayout(options LayoutOptions) CSVLayout {
	return CSVLayout{options: options}
}

// Format 把事件编码为 CSV。
func (l CSVLayout) Format(buf *bytes.Buffer, event Event) error {
	appendCSVField(buf, layoutsupport.EventTime(event.Time).Format(defaultTimeFormat), false)
	appendCSVField(buf, levelName(event.Level), true)
	appendCSVField(buf, event.Logger, true)
	appendCSVField(buf, eventThreadName(event), true)
	appendCSVField(buf, event.Message, true)
	if len(event.Attrs) == 0 {
		appendLayoutTerminator(buf, l.options)
		return nil
	}
	var attrs bytes.Buffer
	logvalue.AppendPatternAttrs(&attrs, event.Attrs)
	appendCSVField(buf, attrs.String(), true)
	appendLayoutTerminator(buf, l.options)
	return nil
}

func (l CSVLayout) AppendHeader(buf *bytes.Buffer) error {
	appendLayoutHeader(buf, l.options)
	return nil
}

func (l CSVLayout) AppendFooter(buf *bytes.Buffer) error {
	appendLayoutFooter(buf, l.options)
	return nil
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
