package logvalue

import (
	"bytes"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// AppendPadded 按 pattern 宽度规则写入字符串。
func AppendPadded(buf *bytes.Buffer, value string, minWidth int, maxWidth int, leftAlign bool) {
	if maxWidth > 0 {
		value = truncatePatternWidth(value, maxWidth)
	}
	width := patternWidth(value)
	if minWidth <= width {
		buf.WriteString(value)
		return
	}
	padding := minWidth - width
	if leftAlign {
		buf.WriteString(value)
		writeSpaces(buf, padding)
		return
	}
	writeSpaces(buf, padding)
	buf.WriteString(value)
}

func patternWidth(value string) int {
	for index := 0; index < len(value); index++ {
		if value[index] >= utf8.RuneSelf {
			return utf8.RuneCountInString(value)
		}
	}
	return len(value)
}

func truncatePatternWidth(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	if len(value) <= limit {
		return value
	}
	for index := 0; index < len(value); index++ {
		if value[index] >= utf8.RuneSelf {
			return truncatePatternRunes(value, limit)
		}
	}
	return value[:limit]
}

func truncatePatternRunes(value string, limit int) string {
	count := 0
	for index := range value {
		if count == limit {
			return value[:index]
		}
		count++
	}
	return value
}

func writeSpaces(buf *bytes.Buffer, count int) {
	for index := 0; index < count; index++ {
		buf.WriteByte(' ')
	}
}

// IsPatternDigit 判断 pattern 字节是否为 ASCII 数字。
func IsPatternDigit(value byte) bool {
	return value >= '0' && value <= '9'
}

// IsPatternLetter 判断 pattern 字节是否为 ASCII 字母。
func IsPatternLetter(value byte) bool {
	return value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z'
}

func isSpaceByte(value byte) bool {
	return value == ' ' || value == '\t' || value == '\r' || value == '\n'
}

// AppendPatternAttrs 按 pattern 的 key=value 风格写入属性集合。
func AppendPatternAttrs(buf *bytes.Buffer, attrs []slog.Attr) {
	for _, attr := range attrs {
		AppendPatternKeyValueAttr(buf, attr.Key, attr.Value)
	}
}

// AppendPatternKeyValueAttr 写入 pattern 属性键值对。
func AppendPatternKeyValueAttr(buf *bytes.Buffer, key string, value slog.Value) {
	if buf.Len() > 0 {
		data := buf.Bytes()
		if !isSpaceByte(data[len(data)-1]) {
			buf.WriteByte(' ')
		}
	}
	buf.WriteString(key)
	buf.WriteByte('=')
	AppendTextValue(buf, value)
}

// AppendKeyValue 写入文本布局中的字符串键值对。
func AppendKeyValue(buf *bytes.Buffer, key string, value string) {
	AppendKey(buf, key)
	QuoteValue(buf, value)
}

// AppendKeyValueAttr 写入文本布局中的 slog 属性键值对。
func AppendKeyValueAttr(buf *bytes.Buffer, key string, value slog.Value) {
	AppendKey(buf, key)
	AppendTextValue(buf, value)
}

// AppendKey 写入文本布局中的属性键前缀。
func AppendKey(buf *bytes.Buffer, key string) {
	if buf.Len() > 0 {
		buf.WriteByte(' ')
	}
	buf.WriteString(key)
	buf.WriteByte('=')
}

// QuoteValue 按 logfmt 近似规则写入字符串值。
func QuoteValue(buf *bytes.Buffer, value string) {
	if value == "" || strings.ContainsAny(value, " \t\r\n\"=") {
		buf.Write(strconv.AppendQuote(buf.AvailableBuffer(), value))
		return
	}
	buf.WriteString(value)
}

// AppendTextValue 写入 slog.Value 的文本表达。
func AppendTextValue(buf *bytes.Buffer, value slog.Value) {
	value = value.Resolve()
	switch value.Kind() {
	case slog.KindString:
		QuoteValue(buf, value.String())
	case slog.KindBool:
		if value.Bool() {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case slog.KindInt64:
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), value.Int64(), 10))
	case slog.KindUint64:
		buf.Write(strconv.AppendUint(buf.AvailableBuffer(), value.Uint64(), 10))
	case slog.KindFloat64:
		buf.Write(strconv.AppendFloat(buf.AvailableBuffer(), value.Float64(), 'g', -1, 64))
	case slog.KindDuration:
		QuoteValue(buf, value.Duration().String())
	case slog.KindTime:
		buf.Write(value.Time().AppendFormat(buf.AvailableBuffer(), time.RFC3339Nano))
	case slog.KindGroup:
		QuoteValue(buf, String(value))
	case slog.KindAny:
		appendTextAny(buf, value.Any())
	default:
		QuoteValue(buf, String(value))
	}
}

func appendTextAny(buf *bytes.Buffer, value any) {
	switch typed := value.(type) {
	case nil:
		buf.WriteString("<nil>")
	case string:
		QuoteValue(buf, typed)
	case error:
		QuoteValue(buf, typed.Error())
	case fmt.Stringer:
		QuoteValue(buf, typed.String())
	default:
		QuoteValue(buf, fmt.Sprint(typed))
	}
}
