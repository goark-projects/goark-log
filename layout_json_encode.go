package goarklog

import (
	"bytes"
	"log/slog"
	"time"
)

func appendJSONEvent(buf *bytes.Buffer, when time.Time, level slog.Level, logger string, message string, attrs []slog.Attr) {
	buf.WriteByte('{')
	appendJSONFieldTime(buf, "time", when, defaultTimeFormat, false)
	appendJSONFieldString(buf, "level", levelName(level), true)
	appendJSONFieldString(buf, "logger", logger, true)
	appendJSONFieldString(buf, "msg", message, true)
	for _, attr := range attrs {
		appendJSONFieldValue(buf, attr.Key, attr.Value, true)
	}
	buf.WriteString("}\n")
}

func appendJSONLayoutEvent(buf *bytes.Buffer, event Event, options LayoutOptions) {
	buf.WriteByte('{')
	appendJSONFieldTime(buf, "time", event.Time, defaultTimeFormat, false)
	appendJSONFieldString(buf, "level", levelName(event.Level), true)
	appendJSONFieldString(buf, "logger", event.Logger, true)
	appendJSONFieldString(buf, "msg", event.Message, true)
	if options.PropertiesAsList {
		appendJSONAttrsListField(buf, "contextMap", event.Attrs, true)
	} else {
		for _, attr := range event.Attrs {
			appendJSONFieldValue(buf, attr.Key, attr.Value, true)
		}
	}
	if event.Throwable != nil && (options.IncludeStacktrace || options.StacktraceAsString) {
		appendJSONKey(buf, "thrown", true)
		if options.StacktraceAsString {
			appendJSONString(buf, throwableStackString(event.Throwable))
		} else {
			appendThrowableJSON(buf, event.Throwable)
		}
	}
	buf.WriteByte('}')
	appendLayoutTerminator(buf, options)
}

func appendJSONFixedEvent(buf *bytes.Buffer, when time.Time, level slog.Level, logger string, message string, attrs [3]slog.Attr, count int) {
	buf.WriteByte('{')
	appendJSONFieldTime(buf, "time", when, defaultTimeFormat, false)
	appendJSONFieldString(buf, "level", levelName(level), true)
	appendJSONFieldString(buf, "logger", logger, true)
	appendJSONFieldString(buf, "msg", message, true)
	for index := 0; index < count && index < len(attrs); index++ {
		attr := attrs[index]
		appendJSONFieldValue(buf, attr.Key, attr.Value, true)
	}
	buf.WriteString("}\n")
}
