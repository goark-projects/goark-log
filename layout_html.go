package goarklog

import (
	"bytes"
	"html"

	"goark.dev/log/internal/layoutsupport"
	"goark.dev/log/internal/logvalue"
)

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
