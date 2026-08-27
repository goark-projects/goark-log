package goarklog

import (
	"bytes"
	"crypto/rand"
	"strconv"
	"strings"
	"time"
)

func appendPatternToken(buf *bytes.Buffer, token patternToken, event Event, caller *callerCache, options LayoutOptions) {
	if token.kind == tokenLiteral {
		buf.WriteString(token.literal)
		return
	}
	if token.kind == tokenAttrs && token.minWidth == 0 && token.maxWidth == 0 {
		appendPatternAttrs(buf, event.Attrs)
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
		buf.WriteString(hostNameString)
		return
	}
	value := patternTokenString(token, event, caller, options)
	appendPadded(buf, value, token.minWidth, token.maxWidth, token.leftAlign)
}

func appendPatternTime(buf *bytes.Buffer, token patternToken, event Event) {
	when := event.Time
	if when.IsZero() {
		when = time.Now()
	}
	switch token.timeUnix {
	case timeUnixSeconds:
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), when.Unix(), 10))
	case timeUnixMillis:
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), when.UnixMilli(), 10))
	case timeUnixMicros:
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), when.UnixMicro(), 10))
	case timeUnixNanos:
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), when.UnixNano(), 10))
	default:
		buf.Write(when.AppendFormat(buf.AvailableBuffer(), token.format))
	}
}

func patternTokenString(token patternToken, event Event, caller *callerCache, options LayoutOptions) string {
	switch token.kind {
	case tokenTime:
		when := event.Time
		if when.IsZero() {
			when = time.Now()
		}
		switch token.timeUnix {
		case timeUnixSeconds:
			return strconv.FormatInt(when.Unix(), 10)
		case timeUnixMillis:
			return strconv.FormatInt(when.UnixMilli(), 10)
		case timeUnixMicros:
			return strconv.FormatInt(when.UnixMicro(), 10)
		case timeUnixNanos:
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
		return attrValueString(value)
	case tokenAttrs:
		var attrBuf bytes.Buffer
		appendPatternAttrs(&attrBuf, event.Attrs)
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
		return caller.resolve(event).class
	case tokenCallerMethod:
		return caller.resolve(event).method
	case tokenCallerFile:
		return caller.resolve(event).file
	case tokenCallerLine:
		frame := caller.resolve(event)
		if frame.line == 0 {
			return ""
		}
		return strconv.Itoa(frame.line)
	case tokenCallerLocation:
		return caller.resolve(event).location()
	case tokenUUID:
		return newPatternUUID()
	case tokenRelative:
		return patternRelativeString()
	case tokenHost:
		return hostNameString
	case tokenSequence:
		return strconv.FormatUint(patternSequence.Add(1), 10)
	case tokenSubPattern:
		return formatChildPattern(token.child, event)
	case tokenHighlight:
		if options.DisableANSI {
			return formatChildPattern(token.child, event)
		}
		return applyANSIStyle(formatChildPattern(token.child, event), highlightStyle(event.Level))
	case tokenStyle:
		if options.DisableANSI {
			return formatChildPattern(token.child, event)
		}
		return applyANSIStyle(formatChildPattern(token.child, event), token.value)
	case tokenNotEmpty:
		value := formatChildPattern(token.child, event)
		if strings.TrimSpace(value) == "" {
			return ""
		}
		return value
	case tokenReplace:
		return token.regex.ReplaceAllString(formatChildPattern(token.child, event), token.repl)
	case tokenEncode:
		return encodePatternValue(formatChildPattern(token.child, event), token.value)
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
		return maxPatternLength(formatChildPattern(token.child, event), token.repeat)
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
