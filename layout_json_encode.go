package goarklog

import (
	"bytes"
	"log/slog"
	"time"

	"goark.dev/log/internal/logvalue"
)

func appendJSONEvent(buf *bytes.Buffer, when time.Time, level slog.Level, logger string, message string, attrs []slog.Attr) {
	buf.WriteByte('{')
	logvalue.AppendJSONFieldTime(buf, "time", when, defaultTimeFormat, false)
	logvalue.AppendJSONFieldString(buf, "level", levelName(level), true)
	logvalue.AppendJSONFieldString(buf, "logger", logger, true)
	logvalue.AppendJSONFieldString(buf, "msg", message, true)
	for _, attr := range attrs {
		logvalue.AppendJSONFieldValue(buf, attr.Key, attr.Value, true)
	}
	buf.WriteString("}\n")
}

func appendJSONLayoutEvent(buf *bytes.Buffer, event Event, options LayoutOptions) {
	buf.WriteByte('{')
	logvalue.AppendJSONFieldTime(buf, "time", event.Time, defaultTimeFormat, false)
	logvalue.AppendJSONFieldString(buf, "level", levelName(event.Level), true)
	logvalue.AppendJSONFieldString(buf, "logger", event.Logger, true)
	logvalue.AppendJSONFieldString(buf, "msg", event.Message, true)
	if options.PropertiesAsList {
		logvalue.AppendJSONAttrsListField(buf, "contextMap", event.Attrs, true)
	} else {
		for _, attr := range event.Attrs {
			logvalue.AppendJSONFieldValue(buf, attr.Key, attr.Value, true)
		}
	}
	if event.Throwable != nil && (options.IncludeStacktrace || options.StacktraceAsString) {
		logvalue.AppendJSONKey(buf, "thrown", true)
		if options.StacktraceAsString {
			logvalue.AppendJSONString(buf, throwableStackString(event.Throwable))
		} else {
			appendThrowableJSON(buf, event.Throwable)
		}
	}
	buf.WriteByte('}')
	appendLayoutTerminator(buf, options)
}

func appendJSONFixedEvent(buf *bytes.Buffer, when time.Time, level slog.Level, logger string, message string, attrs [3]slog.Attr, count int) {
	buf.WriteByte('{')
	logvalue.AppendJSONFieldTime(buf, "time", when, defaultTimeFormat, false)
	logvalue.AppendJSONFieldString(buf, "level", levelName(level), true)
	logvalue.AppendJSONFieldString(buf, "logger", logger, true)
	logvalue.AppendJSONFieldString(buf, "msg", message, true)
	for index := 0; index < count && index < len(attrs); index++ {
		attr := attrs[index]
		logvalue.AppendJSONFieldValue(buf, attr.Key, attr.Value, true)
	}
	buf.WriteString("}\n")
}
