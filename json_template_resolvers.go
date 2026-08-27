package goarklog

import (
	"bytes"
	"log/slog"
	"strconv"
	"time"
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
	unix   timeUnixMode
}

func (r timestampJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	when := event.Time
	if when.IsZero() {
		when = time.Now()
	}
	switch r.unix {
	case timeUnixSeconds:
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), when.Unix(), 10))
	case timeUnixMillis:
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), when.UnixMilli(), 10))
	case timeUnixMicros:
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), when.UnixMicro(), 10))
	case timeUnixNanos:
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
	switch normalizeKind(r.field) {
	case "int", "integer", "value":
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), int64(event.Level), 10))
	case "severity", "syslogseverity":
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), int64(syslogSeverity(event.Level)), 10))
	default:
		appendJSONString(buf, levelName(event.Level))
	}
}

type loggerJSONResolver struct {
	precision int
}

func (r loggerJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	appendJSONString(buf, loggerNameWithPrecision(event.Logger, r.precision))
}

type messageJSONResolver struct{}

func (messageJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	appendJSONString(buf, event.Message)
}

type threadJSONResolver struct{}

func (threadJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	appendJSONString(buf, eventThreadName(event))
}

type markerJSONResolver struct{}

func (markerJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	if event.Marker == nil {
		buf.WriteString("null")
		return
	}
	appendJSONString(buf, event.Marker.String())
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
	switch normalizeKind(r.field) {
	case "", "object":
		if r.stacktraceAsString {
			appendJSONString(buf, throwableStackString(throwable))
			return
		}
		appendThrowableJSON(buf, throwable)
	case "type":
		appendJSONString(buf, throwable.Type)
	case "message":
		appendJSONString(buf, throwable.Message)
	case "string", "formatted":
		if r.stacktraceAsString {
			appendJSONString(buf, throwableStackString(throwable))
			return
		}
		appendJSONString(buf, throwable.String())
	case "rootcause":
		appendThrowableJSON(buf, rootThrowable(throwable))
	case "rootcausemessage":
		appendJSONString(buf, rootThrowable(throwable).Message)
	case "stacktrace":
		appendThrowableStackJSON(buf, throwable)
	case "stacktraceasstring", "stacktracestring":
		appendJSONString(buf, throwableStackString(throwable))
	default:
		appendThrowableJSON(buf, throwable)
	}
}

type sourceJSONResolver struct{}

func (sourceJSONResolver) AppendJSON(buf *bytes.Buffer, event Event) {
	frame := callerFrameFromPC(event.PC)
	if frame.method == "" && frame.file == "" && frame.line == 0 {
		buf.WriteString("null")
		return
	}
	buf.WriteByte('{')
	appendJSONFieldString(buf, "class", frame.class, false)
	appendJSONFieldString(buf, "method", frame.method, true)
	appendJSONFieldString(buf, "file", frame.file, true)
	appendJSONKey(buf, "line", true)
	buf.Write(strconv.AppendInt(buf.AvailableBuffer(), int64(frame.line), 10))
	appendJSONFieldString(buf, "location", frame.location(), true)
	buf.WriteByte('}')
}

type processJSONResolver struct{}

func (processJSONResolver) AppendJSON(buf *bytes.Buffer, _ Event) {
	buf.WriteByte('{')
	appendJSONKey(buf, "pid", false)
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
		appendJSONString(buf, value)
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
		appendJSONAttrsList(buf, attrs)
		return
	}
	appendJSONAttrsObject(buf, attrs)
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
	appendJSONValue(buf, value)
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
		appendJSONFieldValue(buf, attr.Key, attr.Value, index > 0)
	}
	buf.WriteByte('}')
}
