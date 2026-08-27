package goarklog

import (
	"bytes"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"goark.dev/log/internal/logvalue"
)

// RFC5424Layout 输出 RFC 5424 syslog 单行事件。
type RFC5424Layout struct {
	Facility  int
	AppName   string
	MessageID string
}

// SyslogLayout 是 RFC5424Layout 的语义别名。
type SyslogLayout = RFC5424Layout

// Format 把事件编码为 RFC 5424 syslog。
func (l RFC5424Layout) Format(buf *bytes.Buffer, event Event) error {
	priority := syslogPriority(l.Facility, event.Level)
	buf.WriteByte('<')
	buf.Write(strconv.AppendInt(buf.AvailableBuffer(), int64(priority), 10))
	buf.WriteString(">1 ")
	buf.WriteString(eventTime(event.Time).UTC().Format(time.RFC3339Nano))
	buf.WriteByte(' ')
	appendSyslogToken(buf, hostNameString)
	buf.WriteByte(' ')
	appendSyslogToken(buf, firstNonBlank(l.AppName, event.Logger, "goark"))
	buf.WriteByte(' ')
	appendSyslogToken(buf, processIDString)
	buf.WriteByte(' ')
	appendSyslogToken(buf, firstNonBlank(l.MessageID, "-"))
	buf.WriteByte(' ')
	appendStructuredData(buf, event)
	buf.WriteByte(' ')
	buf.WriteString(event.Message)
	buf.WriteByte('\n')
	return nil
}

func syslogPriority(facility int, level slog.Level) int {
	if facility <= 0 || facility > 23 {
		facility = 1
	}
	return facility*8 + syslogSeverity(level)
}

func syslogSeverity(level slog.Level) int {
	switch {
	case level >= slog.LevelError:
		return 3
	case level >= slog.LevelWarn:
		return 4
	case level >= slog.LevelInfo:
		return 6
	default:
		return 7
	}
}

func appendSyslogToken(buf *bytes.Buffer, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		buf.WriteByte('-')
		return
	}
	for _, r := range value {
		if r <= ' ' || r == ']' || r == '"' {
			buf.WriteByte('_')
			continue
		}
		buf.WriteRune(r)
	}
}

func appendStructuredData(buf *bytes.Buffer, event Event) {
	if len(event.Attrs) == 0 {
		buf.WriteByte('-')
		return
	}
	buf.WriteString("[goark@32473")
	for _, attr := range event.Attrs {
		if strings.TrimSpace(attr.Key) == "" {
			continue
		}
		buf.WriteByte(' ')
		buf.WriteString(attr.Key)
		buf.WriteString("=\"")
		appendStructuredDataValue(buf, logvalue.String(attr.Value))
		buf.WriteByte('"')
	}
	buf.WriteByte(']')
}

func appendStructuredDataValue(buf *bytes.Buffer, value string) {
	for _, r := range value {
		switch r {
		case '"', '\\', ']':
			buf.WriteByte('\\')
			buf.WriteRune(r)
		default:
			buf.WriteRune(r)
		}
	}
}
