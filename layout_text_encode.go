package goarklog

import (
	"bytes"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

func appendPadded(buf *bytes.Buffer, value string, minWidth int, maxWidth int, leftAlign bool) {
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

func isPatternDigit(value byte) bool {
	return value >= '0' && value <= '9'
}

func isPatternLetter(value byte) bool {
	return value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z'
}

func isSpaceByte(value byte) bool {
	return value == ' ' || value == '\t' || value == '\r' || value == '\n'
}

func appendPatternAttrs(buf *bytes.Buffer, attrs []slog.Attr) {
	for _, attr := range attrs {
		appendPatternKeyValueAttr(buf, attr.Key, attr.Value)
	}
}

func appendPatternKeyValueAttr(buf *bytes.Buffer, key string, value slog.Value) {
	if buf.Len() > 0 {
		data := buf.Bytes()
		if !isSpaceByte(data[len(data)-1]) {
			buf.WriteByte(' ')
		}
	}
	buf.WriteString(key)
	buf.WriteByte('=')
	appendTextValue(buf, value)
}

func appendKeyValue(buf *bytes.Buffer, key string, value string) {
	appendKey(buf, key)
	quoteValue(buf, value)
}

func appendKeyValueAttr(buf *bytes.Buffer, key string, value slog.Value) {
	appendKey(buf, key)
	appendTextValue(buf, value)
}

func appendKey(buf *bytes.Buffer, key string) {
	if buf.Len() > 0 {
		buf.WriteByte(' ')
	}
	buf.WriteString(key)
	buf.WriteByte('=')
}

func quoteValue(buf *bytes.Buffer, value string) {
	if value == "" || strings.ContainsAny(value, " \t\r\n\"=") {
		buf.Write(strconv.AppendQuote(buf.AvailableBuffer(), value))
		return
	}
	buf.WriteString(value)
}

func appendTextValue(buf *bytes.Buffer, value slog.Value) {
	value = value.Resolve()
	switch value.Kind() {
	case slog.KindString:
		quoteValue(buf, value.String())
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
		quoteValue(buf, value.Duration().String())
	case slog.KindTime:
		buf.Write(value.Time().AppendFormat(buf.AvailableBuffer(), time.RFC3339Nano))
	case slog.KindGroup:
		quoteValue(buf, attrValueString(value))
	case slog.KindAny:
		appendTextAny(buf, value.Any())
	default:
		quoteValue(buf, attrValueString(value))
	}
}

func appendTextAny(buf *bytes.Buffer, value any) {
	switch typed := value.(type) {
	case nil:
		buf.WriteString("<nil>")
	case string:
		quoteValue(buf, typed)
	case error:
		quoteValue(buf, typed.Error())
	case fmt.Stringer:
		quoteValue(buf, typed.String())
	default:
		quoteValue(buf, fmt.Sprint(typed))
	}
}
