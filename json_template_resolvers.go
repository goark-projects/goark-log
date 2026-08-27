package goarklog

import (
	"bytes"
	"log/slog"
	"strconv"
	"time"

	"goark.dev/log/internal/callsite"
	"goark.dev/log/internal/logvalue"
	"goark.dev/log/internal/textutil"
	"goark.dev/log/internal/timepattern"
)

type rawJSONResolver struct {
	raw []byte
}

func (r rawJSONResolver) AppendJSON(buf *bytes.Buffer, _ Event) {
	if len(r.raw) == 0 {
		buf.WriteString("null")
		return
	}
	buf.Write(r.raw)
}

type timestampJSONResolver struct {
	layout string
	unix   timepattern.UnixMode
}

func (r timestampJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	when := event.Time
	if when.IsZero() {
		when = time.Now()
	}
	switch r.unix {
	case timepattern.UnixSeconds:
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), when.Unix(), 10))
	case timepattern.UnixMillis:
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), when.UnixMilli(), 10))
	case timepattern.UnixMicros:
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), when.UnixMicro(), 10))
	case timepattern.UnixNanos:
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), when.UnixNano(), 10))
	default:
		buf.WriteByte('"')
		buf.Write(when.AppendFormat(buf.AvailableBuffer(), r.layout))
		buf.WriteByte('"')
	}
}

type levelJSONResolver struct {
	field string
}

func (r levelJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	switch textutil.NormalizeKind(r.field) {
	case "int", "integer", "value":
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), int64(event.Level), 10))
	case "severity", "syslogseverity":
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), int64(syslogSeverity(event.Level)), 10))
	default:
		logvalue.AppendJSONString(buf, levelName(event.Level))
	}
}

type loggerJSONResolver struct {
	precision int
}

func (r loggerJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	logvalue.AppendJSONString(buf, loggerNameWithPrecision(event.Logger, r.precision))
}

type messageJSONResolver struct{}

func (messageJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	logvalue.AppendJSONString(buf, event.Message)
}

type threadJSONResolver struct{}

func (threadJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	logvalue.AppendJSONString(buf, eventThreadName(event))
}

type markerJSONResolver struct{}

func (markerJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	if event.Marker == nil {
		buf.WriteString("null")
		return
	}
	logvalue.AppendJSONString(buf, event.Marker.String())
}

type throwableJSONResolver struct {
	field              string
	stacktraceAsString bool
}

func (r throwableJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	throwable := event.Throwable
	if throwable == nil {
		buf.WriteString("null")
		return
	}
	switch textutil.NormalizeKind(r.field) {
	case "", "object":
		if r.stacktraceAsString {
			logvalue.AppendJSONString(buf, throwableStackString(throwable))
			return
		}
		appendThrowableJSON(buf, throwable)
	case "type":
		logvalue.AppendJSONString(buf, throwable.Type)
	case "message":
		logvalue.AppendJSONString(buf, throwable.Message)
	case "string", "formatted":
		if r.stacktraceAsString {
			logvalue.AppendJSONString(buf, throwableStackString(throwable))
			return
		}
		logvalue.AppendJSONString(buf, throwable.String())
	case "rootcause":
		appendThrowableJSON(buf, rootThrowable(throwable))
	case "rootcausemessage":
		logvalue.AppendJSONString(buf, rootThrowable(throwable).Message)
	case "stacktrace":
		appendThrowableStackJSON(buf, throwable)
	case "stacktraceasstring", "stacktracestring":
		logvalue.AppendJSONString(buf, throwableStackString(throwable))
	default:
		appendThrowableJSON(buf, throwable)
	}
}

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

type sourceJSONResolver struct{}

func (sourceJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	frame := callsite.FrameFromPC(event.PC)
	if frame.IsZero() {
		buf.WriteString("null")
		return
	}
	buf.WriteByte('{')
	logvalue.AppendJSONFieldString(buf, "class", frame.Class, false)
	logvalue.AppendJSONFieldString(buf, "method", frame.Method, true)
	logvalue.AppendJSONFieldString(buf, "file", frame.File, true)
	logvalue.AppendJSONKey(buf, "line", true)
	buf.Write(strconv.AppendInt(buf.AvailableBuffer(), int64(frame.Line), 10))
	logvalue.AppendJSONFieldString(buf, "location", frame.Location(), true)
	buf.WriteByte('}')
}

type processJSONResolver struct{}

func (processJSONResolver) AppendJSON(buf *bytes.Buffer, _ Event) {
	buf.WriteByte('{')
	logvalue.AppendJSONKey(buf, "pid", false)
	buf.WriteString(processIDString)
	buf.WriteByte('}')
}

type contextStackJSONResolver struct{}

func (contextStackJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	buf.WriteByte('[')
	for index, value := range event.ContextStack {
		if index > 0 {
			buf.WriteByte(',')
		}
		logvalue.AppendJSONString(buf, value)
	}
	buf.WriteByte(']')
}

type attrsJSONResolver struct {
	flatten          bool
	propertiesAsList bool
}

func (r attrsJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	attrs := event.Attrs
	if r.flatten {
		attrs = make([]slog.Attr, 0, len(event.Attrs))
		for _, attr := range event.Attrs {
			appendFlattenedJSONAttr(&attrs, "", attr)
		}
	}
	if r.propertiesAsList {
		logvalue.AppendJSONAttrsList(buf, attrs)
		return
	}
	appendJSONAttrsObject(buf, attrs)
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

type attrJSONResolver struct {
	key string
}

func (r attrJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	value, ok := event.Attr(r.key)
	if !ok {
		buf.WriteString("null")
		return
	}
	logvalue.AppendJSONValue(buf, value)
}

type endOfBatchJSONResolver struct{}

func (endOfBatchJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	if event.EndOfBatch {
		buf.WriteString("true")
		return
	}
	buf.WriteString("false")
}

func appendJSONAttrsObject(buf *bytes.Buffer, attrs []slog.Attr) {
	buf.WriteByte('{')
	for index, attr := range attrs {
		logvalue.AppendJSONFieldValue(buf, attr.Key, attr.Value, index > 0)
	}
	buf.WriteByte('}')
}
