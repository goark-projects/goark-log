package layout

import (
	"bytes"
	"crypto/rand"
	"strconv"
	"strings"
	"time"

	"goark.dev/log/internal/callsite"
	"goark.dev/log/internal/layoutsupport"
	configlevel "goark.dev/log/internal/level"
	"goark.dev/log/internal/logevent"
	"goark.dev/log/internal/logvalue"
	"goark.dev/log/internal/timepattern"
)

func appendPatternToken(buf *bytes.Buffer, token patternToken, event Event, caller *callsite.Cache, options LayoutOptions) {
	if token.kind == tokenLiteral {
		buf.WriteString(token.literal)
		return
	}
	if token.kind == tokenAttrs && token.minWidth == 0 && token.maxWidth == 0 {
		logvalue.AppendPatternAttrs(buf, event.Attrs)
		return
	}
	if token.kind == tokenNewline && token.minWidth == 0 && token.maxWidth == 0 {
		buf.WriteByte('\n')
		return
	}
	if token.kind == tokenTime && token.minWidth == 0 && token.maxWidth == 0 {
		appendPatternTime(buf, token, event)
		return
	}
	if token.kind == tokenPID && token.minWidth == 0 && token.maxWidth == 0 {
		buf.WriteString(processIDString)
		return
	}
	if token.kind == tokenRelative && token.minWidth == 0 && token.maxWidth == 0 {
		appendPatternRelative(buf)
		return
	}
	if token.kind == tokenHost && token.minWidth == 0 && token.maxWidth == 0 {
		buf.WriteString(layoutsupport.HostName())
		return
	}
	value := patternTokenString(token, event, caller, options)
	logvalue.AppendPadded(buf, value, token.minWidth, token.maxWidth, token.leftAlign)
}

func appendPatternTime(buf *bytes.Buffer, token patternToken, event Event) {
	when := event.Time
	if when.IsZero() {
		when = time.Now()
	}
	switch token.timeUnix {
	case timepattern.UnixSeconds:
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), when.Unix(), 10))
	case timepattern.UnixMillis:
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), when.UnixMilli(), 10))
	case timepattern.UnixMicros:
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), when.UnixMicro(), 10))
	case timepattern.UnixNanos:
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), when.UnixNano(), 10))
	default:
		buf.Write(when.AppendFormat(buf.AvailableBuffer(), token.format))
	}
}

func patternTokenString(token patternToken, event Event, caller *callsite.Cache, options LayoutOptions) string {
	switch token.kind {
	case tokenTime:
		when := event.Time
		if when.IsZero() {
			when = time.Now()
		}
		switch token.timeUnix {
		case timepattern.UnixSeconds:
			return strconv.FormatInt(when.Unix(), 10)
		case timepattern.UnixMillis:
			return strconv.FormatInt(when.UnixMilli(), 10)
		case timepattern.UnixMicros:
			return strconv.FormatInt(when.UnixMicro(), 10)
		case timepattern.UnixNanos:
			return strconv.FormatInt(when.UnixNano(), 10)
		default:
			return when.Format(token.format)
		}
	case tokenLevel:
		return levelName(event.Level)
	case tokenPID:
		return processIDString
	case tokenThread:
		return eventThreadName(event)
	case tokenLogger:
		return loggerNameWithPrecision(event.Logger, token.precision)
	case tokenMessage:
		return event.Message
	case tokenAttr:
		value, ok := event.Attr(token.key)
		if !ok {
			return ""
		}
		return logvalue.String(value)
	case tokenAttrs:
		var attrBuf bytes.Buffer
		logvalue.AppendPatternAttrs(&attrBuf, event.Attrs)
		return attrBuf.String()
	case tokenError:
		return eventErrorStringWithOption(event, token.key)
	case tokenNewline:
		return "\n"
	case tokenMarker:
		return eventMarkerString(event)
	case tokenContextStack:
		return contextStackString(event.ContextStack)
	case tokenCallerClass:
		return caller.ResolvePC(event.PC).Class
	case tokenCallerMethod:
		return caller.ResolvePC(event.PC).Method
	case tokenCallerFile:
		return caller.ResolvePC(event.PC).File
	case tokenCallerLine:
		frame := caller.ResolvePC(event.PC)
		if frame.Line == 0 {
			return ""
		}
		return strconv.Itoa(frame.Line)
	case tokenCallerLocation:
		return caller.ResolvePC(event.PC).Location()
	case tokenUUID:
		return newPatternUUID()
	case tokenRelative:
		return patternRelativeString()
	case tokenHost:
		return layoutsupport.HostName()
	case tokenSequence:
		return strconv.FormatUint(patternSequence.Add(1), 10)
	case tokenSubPattern:
		return formatChildPattern(token.child, event)
	case tokenHighlight:
		if options.DisableANSI {
			return formatChildPattern(token.child, event)
		}
		return logvalue.ApplyANSIStyle(formatChildPattern(token.child, event), logvalue.HighlightStyle(event.Level, configlevel.Fatal))
	case tokenStyle:
		if options.DisableANSI {
			return formatChildPattern(token.child, event)
		}
		return logvalue.ApplyANSIStyle(formatChildPattern(token.child, event), token.value)
	case tokenNotEmpty:
		value := formatChildPattern(token.child, event)
		if strings.TrimSpace(value) == "" {
			return ""
		}
		return value
	case tokenReplace:
		return token.regex.ReplaceAllString(formatChildPattern(token.child, event), token.repl)
	case tokenEncode:
		return logvalue.EncodePatternValue(formatChildPattern(token.child, event), token.value)
	case tokenEquals:
		value := formatChildPattern(token.child, event)
		matched := value == token.value
		if token.ignore {
			matched = strings.EqualFold(value, token.value)
		}
		if matched {
			return token.repl
		}
		return value
	case tokenMaxLen:
		return logvalue.MaxPatternLength(formatChildPattern(token.child, event), token.repeat)
	case tokenRepeat:
		return strings.Repeat(formatChildPattern(token.child, event), token.repeat)
	default:
		return ""
	}
}

func firstPatternOption(options []string) string {
	return patternOption(options, 0)
}

func patternOption(options []string, index int) string {
	if index < 0 || index >= len(options) {
		return ""
	}
	return options[index]
}

func parsePatternPrecision(option string) int {
	value, err := strconv.Atoi(strings.TrimSpace(option))
	if err != nil || value <= 0 {
		return 0
	}
	return value
}

func loggerNameWithPrecision(name string, precision int) string {
	if precision <= 0 || name == "" {
		return name
	}
	parts := strings.Split(name, ".")
	if precision >= len(parts) {
		return name
	}
	return strings.Join(parts[len(parts)-precision:], ".")
}

func formatChildPattern(layout *PatternLayout, event Event) string {
	if layout == nil {
		return ""
	}
	var buf bytes.Buffer
	if err := layout.Format(&buf, event); err != nil {
		return ""
	}
	return buf.String()
}

func appendPatternRelative(buf *bytes.Buffer) {
	buf.Write(strconv.AppendInt(buf.AvailableBuffer(), patternRelativeMillis(), 10))
}

func patternRelativeString() string {
	return strconv.FormatInt(patternRelativeMillis(), 10)
}

func patternRelativeMillis() int64 {
	elapsed := time.Since(patternStartTime).Milliseconds()
	if elapsed < 0 {
		return 0
	}
	return elapsed
}

func newPatternUUID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80
	var out [36]byte
	hex := "0123456789abcdef"
	source := 0
	for index := 0; index < len(out); index++ {
		switch index {
		case 8, 13, 18, 23:
			out[index] = '-'
		default:
			if index == 14 {
				out[index] = '4'
				source++
				continue
			}
			b := value[source/2]
			if source%2 == 0 {
				out[index] = hex[b>>4]
			} else {
				out[index] = hex[b&0x0f]
			}
			source++
		}
	}
	return string(out[:])
}

func eventMarkerString(event Event) string {
	if event.Marker != nil {
		return event.Marker.String()
	}
	for _, key := range []string{"marker", "goark.marker"} {
		value, ok := event.Attr(key)
		if ok {
			return logvalue.String(value)
		}
	}
	return ""
}

func eventThreadName(event Event) string {
	if strings.TrimSpace(event.ThreadName) != "" {
		return strings.TrimSpace(event.ThreadName)
	}
	for _, key := range []string{"goark.thread", "thread", "goroutine"} {
		value, ok := event.Attr(key)
		if ok {
			name := strings.TrimSpace(logvalue.String(value))
			if name != "" {
				return name
			}
		}
	}
	return defaultThreadName
}

func eventErrorString(event Event) string {
	return eventErrorStringWithOption(event, "")
}

func eventErrorStringWithOption(event Event, option string) string {
	option = strings.ToLower(strings.TrimSpace(option))
	if option == "none" {
		return ""
	}
	if event.Throwable != nil {
		return throwableStringWithPatternOption(event.Throwable, option)
	}
	if throwable := logevent.ThrowableFromAttrs(event.Attrs); throwable != nil {
		return throwableStringWithPatternOption(throwable, option)
	}
	for _, key := range []string{"error", "err"} {
		value, ok := event.Attr(key)
		if ok {
			return logvalue.String(value)
		}
	}
	return ""
}

func throwableStringWithPatternOption(throwable *Throwable, option string) string {
	if throwable == nil {
		return ""
	}
	switch option {
	case "none":
		return ""
	case "short":
		return throwable.Message
	case "full":
		return logevent.ThrowableStackString(throwable)
	default:
		return throwable.String()
	}
}
