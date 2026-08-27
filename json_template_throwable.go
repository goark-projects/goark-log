package goarklog

import (
	"bytes"
	"log/slog"
)

func appendThrowableJSON(buf *bytes.Buffer, throwable *Throwable) {
	if throwable == nil {
		buf.WriteString("null")
		return
	}
	buf.WriteByte('{')
	appendJSONFieldString(buf, "type", throwable.Type, false)
	appendJSONFieldString(buf, "message", throwable.Message, true)
	appendJSONKey(buf, "rootCause", true)
	appendThrowableRootCauseJSON(buf, rootThrowable(throwable))
	appendJSONKey(buf, "stackTrace", true)
	appendThrowableStackJSON(buf, throwable)
	if throwable.Cause != nil {
		appendJSONKey(buf, "cause", true)
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
	appendJSONFieldString(buf, "type", throwable.Type, false)
	appendJSONFieldString(buf, "message", throwable.Message, true)
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
		appendJSONString(buf, frame)
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
