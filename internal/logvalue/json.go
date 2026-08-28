package logvalue

import (
	"bytes"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"time"

	"goark.dev/log/internal/jsoncodec"
)

// AppendJSONFieldString 写入 JSON 字符串字段。
func AppendJSONFieldString(buf *bytes.Buffer, key string, value string, comma bool) {
	AppendJSONKey(buf, key, comma)
	AppendJSONString(buf, value)
}

// AppendJSONFieldValue 写入 JSON slog.Value 字段。
func AppendJSONFieldValue(buf *bytes.Buffer, key string, value slog.Value, comma bool) {
	AppendJSONKey(buf, key, comma)
	AppendJSONValue(buf, value)
}

// AppendJSONFieldTime 写入 JSON 时间字段。
func AppendJSONFieldTime(buf *bytes.Buffer, key string, value time.Time, layout string, comma bool) {
	AppendJSONKey(buf, key, comma)
	buf.WriteByte('"')
	buf.Write(value.AppendFormat(buf.AvailableBuffer(), layout))
	buf.WriteByte('"')
}

// AppendJSONKey 写入 JSON 对象键。
func AppendJSONKey(buf *bytes.Buffer, key string, comma bool) {
	if comma {
		buf.WriteByte(',')
	}
	AppendJSONString(buf, key)
	buf.WriteByte(':')
}

// AppendJSONString 写入 JSON 字符串值。
func AppendJSONString(buf *bytes.Buffer, value string) {
	buf.Write(strconv.AppendQuote(buf.AvailableBuffer(), value))
}

// AppendJSONValue 写入 slog.Value 的 JSON 表达。
func AppendJSONValue(buf *bytes.Buffer, value slog.Value) {
	value = value.Resolve()
	switch value.Kind() {
	case slog.KindString:
		AppendJSONString(buf, value.String())
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
		float := value.Float64()
		if math.IsNaN(float) || math.IsInf(float, 0) {
			AppendJSONString(buf, strconv.FormatFloat(float, 'g', -1, 64))
			return
		}
		buf.Write(strconv.AppendFloat(buf.AvailableBuffer(), float, 'g', -1, 64))
	case slog.KindDuration:
		AppendJSONString(buf, value.Duration().String())
	case slog.KindTime:
		buf.WriteByte('"')
		buf.Write(value.Time().AppendFormat(buf.AvailableBuffer(), time.RFC3339Nano))
		buf.WriteByte('"')
	case slog.KindGroup:
		buf.WriteByte('{')
		for index, attr := range value.Group() {
			AppendJSONFieldValue(buf, attr.Key, attr.Value, index > 0)
		}
		buf.WriteByte('}')
	case slog.KindAny:
		appendJSONAny(buf, value.Any())
	default:
		AppendJSONString(buf, String(value))
	}
}

func appendJSONAny(buf *bytes.Buffer, value any) {
	switch typed := value.(type) {
	case nil:
		buf.WriteString("null")
	case string:
		AppendJSONString(buf, typed)
	case error:
		AppendJSONString(buf, typed.Error())
	case fmt.Stringer:
		AppendJSONString(buf, typed.String())
	default:
		data, err := jsoncodec.Marshal(typed)
		if err != nil {
			AppendJSONString(buf, fmt.Sprint(typed))
			return
		}
		buf.Write(data)
	}
}

// AppendJSONAttrsListField 写入 Log4j2 propertiesAsList 风格字段。
func AppendJSONAttrsListField(buf *bytes.Buffer, key string, attrs []slog.Attr, comma bool) {
	AppendJSONKey(buf, key, comma)
	AppendJSONAttrsList(buf, attrs)
}

// AppendJSONAttrsList 写入 Log4j2 propertiesAsList 风格属性数组。
func AppendJSONAttrsList(buf *bytes.Buffer, attrs []slog.Attr) {
	buf.WriteByte('[')
	for index, attr := range attrs {
		if index > 0 {
			buf.WriteByte(',')
		}
		buf.WriteByte('{')
		AppendJSONFieldString(buf, "key", attr.Key, false)
		AppendJSONKey(buf, "value", true)
		AppendJSONValue(buf, attr.Value)
		buf.WriteByte('}')
	}
	buf.WriteByte(']')
}
