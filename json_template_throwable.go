package goarklog

import (
	"bytes"
	"log/slog"

	"goark.dev/log/internal/logvalue"
)

func appendThrowableJSON(buf *bytes.Buffer, throwable *Throwable) {
	if throwable == nil {
		buf.WriteString("null")
		return
	}
	buf.WriteByte('{')
	logvalue.AppendJSONFieldString(buf, "type", throwable.Type, false)
	logvalue.AppendJSONFieldString(buf, "message", throwable.Message, true)
	logvalue.AppendJSONKey(buf, "rootCause", true)
	appendThrowableRootCauseJSON(buf, rootThrowable(throwable))
	logvalue.AppendJSONKey(buf, "stackTrace", true)
	appendThrowableStackJSON(buf, throwable)
	if throwable.Cause != nil {
		logvalue.AppendJSONKey(buf, "cause", true)
		appendThrowableJSON(buf, throwable.Cause)
	}
	buf.WriteByte('}')
}

func appendThrowableRootCauseJSON(buf *bytes.Buffer, throwable *Throwable) {
	if throwable == nil {
		buf.WriteString("null")
		return
	}
	buf.WriteByte('{')
	logvalue.AppendJSONFieldString(buf, "type", throwable.Type, false)
	logvalue.AppendJSONFieldString(buf, "message", throwable.Message, true)
	buf.WriteByte('}')
}

func appendThrowableStackJSON(buf *bytes.Buffer, throwable *Throwable) {
	if throwable == nil || len(throwable.Stack) == 0 {
		buf.WriteString("null")
		return
	}
	buf.WriteByte('[')
	for index, frame := range throwable.Stack {
		if index > 0 {
			buf.WriteByte(',')
		}
		logvalue.AppendJSONString(buf, frame)
	}
	buf.WriteByte(']')
}

func rootThrowable(throwable *Throwable) *Throwable {
	if throwable == nil {
		return nil
	}
	for throwable.Cause != nil {
		throwable = throwable.Cause
	}
	return throwable
}

func appendFlattenedJSONAttr(attrs *[]slog.Attr, prefix string, attr slog.Attr) {
	attr = normalizeAttr(attr)
	if attr.Key == "" {
		return
	}
	key := attr.Key
	if prefix != "" {
		key = prefix + "." + key
	}
	if attr.Value.Kind() != slog.KindGroup {
		*attrs = append(*attrs, slog.Attr{Key: key, Value: attr.Value})
		return
	}
	for _, child := range attr.Value.Group() {
		appendFlattenedJSONAttr(attrs, key, child)
	}
}
