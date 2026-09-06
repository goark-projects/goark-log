package layout

import (
	"bytes"
	"html"

	"goark.dev/log/internal/layoutsupport"
	"goark.dev/log/internal/logvalue"
)

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

// HTMLLayout 输出 HTML 表格行，适合文件或控制台片段组合。
type HTMLLayout struct {
	options LayoutOptions
}

// NewHTMLLayout 创建可配置 HTML 布局。
func NewHTMLLayout(options LayoutOptions) HTMLLayout {
	return HTMLLayout{options: options}
}

// Format 把事件编码为 HTML 表格行。
func (l HTMLLayout) Format(buf *bytes.Buffer, event Event) error {
	buf.WriteString("<tr>")
	appendHTMLCell(buf, layoutsupport.EventTime(event.Time).Format(defaultTimeFormat))
	appendHTMLCell(buf, levelName(event.Level))
	appendHTMLCell(buf, event.Logger)
	appendHTMLCell(buf, eventThreadName(event))
	appendHTMLCell(buf, event.Message)
	if len(event.Attrs) > 0 {
		var attrs bytes.Buffer
		logvalue.AppendPatternAttrs(&attrs, event.Attrs)
		appendHTMLCell(buf, attrs.String())
	} else {
		appendHTMLCell(buf, "")
	}
	buf.WriteString("</tr>")
	appendLayoutTerminator(buf, l.options)
	return nil
}

func (l HTMLLayout) AppendHeader(buf *bytes.Buffer) error {
	appendLayoutHeader(buf, l.options)
	return nil
}

func (l HTMLLayout) AppendFooter(buf *bytes.Buffer) error {
	appendLayoutFooter(buf, l.options)
	return nil
}

func appendHTMLCell(buf *bytes.Buffer, value string) {
	buf.WriteString("<td>")
	buf.WriteString(html.EscapeString(value))
	buf.WriteString("</td>")
}
