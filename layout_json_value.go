package goarklog

import (
	"bytes"
	"fmt"
	"goark.dev/log/internal/jsoncodec"
	"log/slog"
	"strconv"
	"strings"
	"time"
)

func appendJSONFieldString(buf *bytes.Buffer, key string, value string, comma bool) {
	appendJSONKey(buf, key, comma)
	appendJSONString(buf, value)
}

func appendJSONFieldValue(buf *bytes.Buffer, key string, value slog.Value, comma bool) {
	appendJSONKey(buf, key, comma)
	appendJSONValue(buf, value)
}

func appendJSONFieldTime(buf *bytes.Buffer, key string, value time.Time, layout string, comma bool) {
	appendJSONKey(buf, key, comma)
	buf.WriteByte('"')
	buf.Write(value.AppendFormat(buf.AvailableBuffer(), layout))
	buf.WriteByte('"')
}

func appendJSONKey(buf *bytes.Buffer, key string, comma bool) {
	if comma {
		buf.WriteByte(',')
	}
	appendJSONString(buf, key)
	buf.WriteByte(':')
}

func appendJSONString(buf *bytes.Buffer, value string) {
	buf.Write(strconv.AppendQuote(buf.AvailableBuffer(), value))
}

func appendJSONValue(buf *bytes.Buffer, value slog.Value) {
	value = value.Resolve()
	switch value.Kind() {
	case slog.KindString:
		appendJSONString(buf, value.String())
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
		appendJSONString(buf, value.Duration().String())
	case slog.KindTime:
		buf.WriteByte('"')
		buf.Write(value.Time().AppendFormat(buf.AvailableBuffer(), time.RFC3339Nano))
		buf.WriteByte('"')
	case slog.KindGroup:
		buf.WriteByte('{')
		for index, attr := range value.Group() {
			appendJSONFieldValue(buf, attr.Key, attr.Value, index > 0)
		}
		buf.WriteByte('}')
	case slog.KindAny:
		appendJSONAny(buf, value.Any())
	default:
		appendJSONString(buf, attrValueString(value))
	}
}

func appendJSONAny(buf *bytes.Buffer, value any) {
	switch typed := value.(type) {
	case nil:
		buf.WriteString("null")
	case string:
		appendJSONString(buf, typed)
	case error:
		appendJSONString(buf, typed.Error())
	case fmt.Stringer:
		appendJSONString(buf, typed.String())
	default:
		data, err := jsoncodec.Marshal(typed)
		if err != nil {
			appendJSONString(buf, fmt.Sprint(typed))
			return
		}
		buf.Write(data)
	}
}

func appendJSONAttrsListField(buf *bytes.Buffer, key string, attrs []slog.Attr, comma bool) {
	appendJSONKey(buf, key, comma)
	appendJSONAttrsList(buf, attrs)
}

func appendJSONAttrsList(buf *bytes.Buffer, attrs []slog.Attr) {
	buf.WriteByte('[')
	for index, attr := range attrs {
		if index > 0 {
			buf.WriteByte(',')
		}
		buf.WriteByte('{')
		appendJSONFieldString(buf, "key", attr.Key, false)
		appendJSONKey(buf, "value", true)
		appendJSONValue(buf, attr.Value)
		buf.WriteByte('}')
	}
	buf.WriteByte(']')
}

func attrValueString(value slog.Value) string {
	value = value.Resolve()
	switch value.Kind() {
	case slog.KindString:
		return value.String()
	case slog.KindBool:
		if value.Bool() {
			return "true"
		}
		return "false"
	case slog.KindInt64:
		return strconv.FormatInt(value.Int64(), 10)
	case slog.KindUint64:
		return strconv.FormatUint(value.Uint64(), 10)
	case slog.KindFloat64:
		return strconv.FormatFloat(value.Float64(), 'g', -1, 64)
	case slog.KindTime:
		return value.Time().Format(time.RFC3339Nano)
	case slog.KindDuration:
		return value.Duration().String()
	case slog.KindGroup:
		var builder strings.Builder
		for index, attr := range value.Group() {
			if index > 0 {
				builder.WriteByte(',')
			}
			builder.WriteString(attr.Key)
			builder.WriteByte('=')
			builder.WriteString(attrValueString(attr.Value))
		}
		return builder.String()
	default:
		return fmt.Sprint(value.Any())
	}
}
