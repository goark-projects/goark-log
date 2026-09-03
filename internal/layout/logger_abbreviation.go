package layout

import (
	"bytes"
	"strconv"
	"strings"
	"unicode/utf8"
)

type loggerAbbreviationKind uint8

const (
	loggerAbbreviationNone loggerAbbreviationKind = iota
	loggerAbbreviationRetain
	loggerAbbreviationDrop
	loggerAbbreviationPattern
	loggerAbbreviationDynamic
)

type loggerAbbreviationFragment struct {
	charCount int
	ellipsis  string
}

type loggerAbbreviator struct {
	kind      loggerAbbreviationKind
	count     int
	fragments []loggerAbbreviationFragment
}

func newLoggerAbbreviator(pattern string) loggerAbbreviator {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return loggerAbbreviator{}
	}
	if count, ok := parseDynamicLoggerPrecision(pattern); ok {
		return loggerAbbreviator{kind: loggerAbbreviationDynamic, count: count}
	}
	if count, negative, ok := parseNumericLoggerPrecision(pattern); ok {
		if negative {
			return loggerAbbreviator{kind: loggerAbbreviationDrop, count: count}
		}
		if count < 1 {
			count = 1
		}
		return loggerAbbreviator{kind: loggerAbbreviationRetain, count: count}
	}
	return loggerAbbreviator{
		kind:      loggerAbbreviationPattern,
		fragments: parseLoggerAbbreviationFragments(pattern),
	}
}

func parseDynamicLoggerPrecision(pattern string) (int, bool) {
	if !strings.HasPrefix(pattern, "1.") || !strings.HasSuffix(pattern, "*") {
		return 0, false
	}
	value := pattern[2 : len(pattern)-1]
	if value == "" || strings.IndexByte(value, '.') >= 0 {
		return 0, false
	}
	count, err := strconv.Atoi(value)
	return count, err == nil && count > 0
}

func parseNumericLoggerPrecision(pattern string) (count int, negative bool, ok bool) {
	number := pattern
	if strings.HasPrefix(number, "-") && len(number) > 1 {
		negative = true
		number = number[1:]
	}
	if number == "" {
		return 0, false, false
	}
	for index := 0; index < len(number); index++ {
		if number[index] < '0' || number[index] > '9' {
			return 0, false, false
		}
	}
	value, err := strconv.Atoi(number)
	if err != nil {
		return 0, false, false
	}
	return value, negative, true
}

func parseLoggerAbbreviationFragments(pattern string) []loggerAbbreviationFragment {
	fragments := make([]loggerAbbreviationFragment, 0, 4)
	for position := 0; position < len(pattern); {
		fragment := loggerAbbreviationFragment{}
		ellipsisPosition := position
		switch current := pattern[position]; {
		case current == '*':
			fragment.charCount = int(^uint(0) >> 1)
			ellipsisPosition++
		case current >= '0' && current <= '9':
			fragment.charCount = int(current - '0')
			ellipsisPosition++
		}
		if ellipsisPosition < len(pattern) && pattern[ellipsisPosition] != '.' {
			_, size := utf8.DecodeRuneInString(pattern[ellipsisPosition:])
			fragment.ellipsis = pattern[ellipsisPosition : ellipsisPosition+size]
		}
		fragments = append(fragments, fragment)
		next := strings.IndexByte(pattern[position:], '.')
		if next < 0 {
			break
		}
		position += next + 1
	}
	return fragments
}

func (a loggerAbbreviator) format(name string) string {
	if a.kind == loggerAbbreviationNone || name == "" {
		return name
	}
	var buf bytes.Buffer
	buf.Grow(len(name))
	a.append(&buf, name)
	return buf.String()
}

func (a loggerAbbreviator) append(buf *bytes.Buffer, name string) {
	if name == "" {
		return
	}
	switch a.kind {
	case loggerAbbreviationRetain:
		appendRetainedLoggerName(buf, name, a.count)
	case loggerAbbreviationDrop:
		appendDroppedLoggerName(buf, name, a.count)
	case loggerAbbreviationPattern:
		appendPatternLoggerName(buf, name, a.fragments)
	case loggerAbbreviationDynamic:
		appendDynamicLoggerName(buf, name, a.count)
	default:
		buf.WriteString(name)
	}
}

func appendRetainedLoggerName(buf *bytes.Buffer, name string, count int) {
	end := len(name) - 1
	for range count {
		index := strings.LastIndexByte(name[:end], '.')
		if index < 0 {
			buf.WriteString(name)
			return
		}
		end = index
	}
	buf.WriteString(name[end+1:])
}

func appendDroppedLoggerName(buf *bytes.Buffer, name string, count int) {
	start := 0
	for range count {
		index := strings.IndexByte(name[start:], '.')
		if index < 0 {
			buf.WriteString(name)
			return
		}
		start += index + 1
	}
	buf.WriteString(name[start:])
}

func appendPatternLoggerName(buf *bytes.Buffer, name string, fragments []loggerAbbreviationFragment) {
	if len(fragments) == 0 {
		buf.WriteString(name)
		return
	}
	start := 0
	fragmentIndex := 0
	for start < len(name) {
		dot := strings.IndexByte(name[start:], '.')
		if dot < 0 {
			buf.WriteString(name[start:])
			return
		}
		end := start + dot
		fragment := fragments[min(fragmentIndex, len(fragments)-1)]
		prefixEnd, abbreviated := loggerPrefixEnd(name, start, end, fragment.charCount)
		buf.WriteString(name[start:prefixEnd])
		if abbreviated {
			buf.WriteString(fragment.ellipsis)
		}
		buf.WriteByte('.')
		start = end + 1
		fragmentIndex++
	}
}

func appendDynamicLoggerName(buf *bytes.Buffer, name string, retained int) {
	components := 1 + strings.Count(name, ".")
	if retained >= components {
		buf.WriteString(name)
		return
	}
	abbreviated := components - retained
	start := 0
	for range abbreviated {
		dot := strings.IndexByte(name[start:], '.')
		if dot < 0 {
			buf.WriteString(name[start:])
			return
		}
		end := start + dot
		prefixEnd, _ := loggerPrefixEnd(name, start, end, 1)
		buf.WriteString(name[start:prefixEnd])
		buf.WriteByte('.')
		start = end + 1
	}
	buf.WriteString(name[start:])
}

func loggerPrefixEnd(value string, start, end, count int) (int, bool) {
	if count >= end-start {
		return end, false
	}
	position := start
	for range count {
		_, size := utf8.DecodeRuneInString(value[position:end])
		position += size
		if position >= end {
			return end, false
		}
	}
	return position, true
}
