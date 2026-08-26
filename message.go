package goarklog

import (
	"bytes"
	"fmt"
	"log/slog"
	"strings"
)

const (
	// StructuredDataIDAttrKey 是结构化消息 ID 的标准属性键。
	StructuredDataIDAttrKey = "goark.structuredData.id"
	// StructuredDataTypeAttrKey 是结构化消息类型的标准属性键。
	StructuredDataTypeAttrKey = "goark.structuredData.type"
)

// Message 表示可被日志事件快照化的消息对象。
type Message interface {
	fmt.Stringer
}

// AttributedMessage 表示会同时贡献结构化属性的消息对象。
type AttributedMessage interface {
	Message
	Attrs() []slog.Attr
}

// SimpleMessage 是不可变字符串消息。
type SimpleMessage string

// NewSimpleMessage 创建字符串消息。
func NewSimpleMessage(text string) SimpleMessage {
	return SimpleMessage(text)
}

func (m SimpleMessage) String() string {
	return string(m)
}

// ParameterizedMessage 使用 Log4j2 风格的 {} 占位符格式化消息。
type ParameterizedMessage struct {
	pattern string
	args    []any
}

// NewParameterizedMessage 创建参数化消息，参数会被快照复制。
func NewParameterizedMessage(pattern string, args ...any) ParameterizedMessage {
	return ParameterizedMessage{
		pattern: pattern,
		args:    append([]any(nil), args...),
	}
}

func (m ParameterizedMessage) String() string {
	if len(m.args) == 0 || !strings.Contains(m.pattern, "{}") {
		return m.pattern
	}
	var builder strings.Builder
	argIndex := 0
	for index := 0; index < len(m.pattern); index++ {
		if index+2 <= len(m.pattern) && m.pattern[index] == '\\' && strings.HasPrefix(m.pattern[index+1:], "{}") {
			builder.WriteString("{}")
			index += 2
			continue
		}
		if index+1 < len(m.pattern) && m.pattern[index] == '{' && m.pattern[index+1] == '}' && argIndex < len(m.args) {
			builder.WriteString(fmt.Sprint(m.args[argIndex]))
			argIndex++
			index++
			continue
		}
		builder.WriteByte(m.pattern[index])
	}
	return builder.String()
}

// MapMessage 用属性集合表达消息，适合结构化日志。
type MapMessage struct {
	attrs []slog.Attr
}

// NewMapMessage 创建结构化属性消息。
func NewMapMessage(attrs ...slog.Attr) MapMessage {
	return MapMessage{attrs: normalizeMessageAttrs(attrs)}
}

// WithAttr 返回追加属性后的新 MapMessage。
func (m MapMessage) WithAttr(attr slog.Attr) MapMessage {
	attr = normalizeAttr(attr)
	if attr.Key == "" || attr.Key == loggerNameKey {
		return m
	}
	next := MapMessage{attrs: append([]slog.Attr(nil), m.attrs...)}
	next.attrs = append(next.attrs, attr)
	return next
}

// Attrs 返回消息贡献的属性快照。
func (m MapMessage) Attrs() []slog.Attr {
	return append([]slog.Attr(nil), m.attrs...)
}

func (m MapMessage) String() string {
	return messageAttrsString(m.attrs)
}

// StructuredDataMessage 表示 Log4j2/RFC5424 风格的结构化消息。
type StructuredDataMessage struct {
	id      string
	msgType string
	message string
	attrs   []slog.Attr
}

// NewStructuredDataMessage 创建结构化数据消息。
func NewStructuredDataMessage(id string, msgType string, message string, attrs ...slog.Attr) StructuredDataMessage {
	return StructuredDataMessage{
		id:      strings.TrimSpace(id),
		msgType: strings.TrimSpace(msgType),
		message: strings.TrimSpace(message),
		attrs:   normalizeMessageAttrs(attrs),
	}
}

// Attrs 返回结构化数据字段快照。
func (m StructuredDataMessage) Attrs() []slog.Attr {
	total := len(m.attrs)
	if m.id != "" {
		total++
	}
	if m.msgType != "" {
		total++
	}
	attrs := make([]slog.Attr, 0, total)
	if m.id != "" {
		attrs = append(attrs, slog.String(StructuredDataIDAttrKey, m.id))
	}
	if m.msgType != "" {
		attrs = append(attrs, slog.String(StructuredDataTypeAttrKey, m.msgType))
	}
	return append(attrs, m.attrs...)
}

func (m StructuredDataMessage) String() string {
	var buf bytes.Buffer
	if m.id != "" {
		buf.WriteByte('[')
		buf.WriteString(m.id)
		if m.msgType != "" {
			buf.WriteString(" type=")
			quoteValue(&buf, m.msgType)
		}
		for _, attr := range m.attrs {
			buf.WriteByte(' ')
			buf.WriteString(attr.Key)
			buf.WriteByte('=')
			appendTextValue(&buf, attr.Value)
		}
		buf.WriteByte(']')
		if m.message != "" {
			buf.WriteByte(' ')
		}
	}
	buf.WriteString(m.message)
	return buf.String()
}

func normalizeMessageAttrs(attrs []slog.Attr) []slog.Attr {
	if len(attrs) == 0 {
		return nil
	}
	out := make([]slog.Attr, 0, len(attrs))
	for _, attr := range attrs {
		attr = normalizeAttr(attr)
		if attr.Key == "" || attr.Key == loggerNameKey {
			continue
		}
		out = append(out, attr)
	}
	return out
}

func messageAttrsString(attrs []slog.Attr) string {
	var buf bytes.Buffer
	for _, attr := range attrs {
		appendPatternKeyValueAttr(&buf, attr.Key, attr.Value)
	}
	return buf.String()
}
