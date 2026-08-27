package goarklog

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func formatRollingFilePattern(pattern string, now time.Time, index int) (string, error) {
	var builder strings.Builder
	builder.Grow(len(pattern) + 8)
	for offset := 0; offset < len(pattern); {
		if pattern[offset] != '%' {
			builder.WriteByte(pattern[offset])
			offset++
			continue
		}
		if offset+1 < len(pattern) && pattern[offset+1] == '%' {
			builder.WriteByte('%')
			offset += 2
			continue
		}
		next, err := appendRollingPatternToken(&builder, pattern, offset, now, index)
		if err != nil {
			return "", err
		}
		offset = next
	}
	return builder.String(), nil
}

func appendRollingPatternToken(builder *strings.Builder, pattern string, offset int, now time.Time, index int) (int, error) {
	cursor := offset + 1
	zeroPad := false
	width := 0
	if cursor < len(pattern) && pattern[cursor] == '0' {
		zeroPad = true
		cursor++
	}
	for cursor < len(pattern) && pattern[cursor] >= '0' && pattern[cursor] <= '9' {
		width = width*10 + int(pattern[cursor]-'0')
		cursor++
	}
	if cursor >= len(pattern) {
		return 0, fmt.Errorf("goark-log: rolling filePattern token is incomplete near %q", pattern[offset:])
	}
	switch pattern[cursor] {
	case 'i':
		value := strconv.Itoa(index)
		if zeroPad && width > len(value) {
			builder.WriteString(strings.Repeat("0", width-len(value)))
		}
		builder.WriteString(value)
		return cursor + 1, nil
	case 'd':
		option := ""
		cursor++
		if cursor < len(pattern) && pattern[cursor] == '{' {
			end := strings.IndexByte(pattern[cursor+1:], '}')
			if end < 0 {
				return 0, fmt.Errorf("goark-log: rolling filePattern date option is not closed near %q", pattern[cursor:])
			}
			option = pattern[cursor+1 : cursor+1+end]
			cursor += end + 2
		}
		if strings.TrimSpace(option) == "" {
			option = "yyyyMMdd-HHmmss"
		}
		layout, unixMode := normalizeTimePattern(option)
		switch unixMode {
		case timeUnixSeconds:
			builder.WriteString(strconv.FormatInt(now.Unix(), 10))
		case timeUnixMillis:
			builder.WriteString(strconv.FormatInt(now.UnixMilli(), 10))
		case timeUnixMicros:
			builder.WriteString(strconv.FormatInt(now.UnixMicro(), 10))
		case timeUnixNanos:
			builder.WriteString(strconv.FormatInt(now.UnixNano(), 10))
		default:
			builder.WriteString(now.Format(layout))
		}
		return cursor, nil
	default:
		return 0, fmt.Errorf("goark-log: unsupported rolling filePattern token near %q", pattern[offset:])
	}
}

func rollingPatternHasIndex(pattern string) bool {
	for offset := 0; offset < len(pattern); offset++ {
		if pattern[offset] != '%' {
			continue
		}
		cursor := offset + 1
		if cursor < len(pattern) && pattern[cursor] == '0' {
			cursor++
		}
		for cursor < len(pattern) && pattern[cursor] >= '0' && pattern[cursor] <= '9' {
			cursor++
		}
		if cursor < len(pattern) && pattern[cursor] == 'i' {
			return true
		}
	}
	return false
}

func rollingPatternGlob(pattern string, compress bool) string {
	var builder strings.Builder
	builder.Grow(len(pattern) + 8)
	for offset := 0; offset < len(pattern); {
		if pattern[offset] != '%' {
			builder.WriteByte(pattern[offset])
			offset++
			continue
		}
		if offset+1 < len(pattern) && pattern[offset+1] == '%' {
			builder.WriteByte('%')
			offset += 2
			continue
		}
		cursor := offset + 1
		if cursor < len(pattern) && pattern[cursor] == '0' {
			cursor++
		}
		for cursor < len(pattern) && pattern[cursor] >= '0' && pattern[cursor] <= '9' {
			cursor++
		}
		if cursor < len(pattern) && pattern[cursor] == 'd' {
			cursor++
			if cursor < len(pattern) && pattern[cursor] == '{' {
				end := strings.IndexByte(pattern[cursor+1:], '}')
				if end >= 0 {
					cursor += end + 2
				}
			}
			builder.WriteByte('*')
			offset = cursor
			continue
		}
		if cursor < len(pattern) && pattern[cursor] == 'i' {
			builder.WriteByte('*')
			offset = cursor + 1
			continue
		}
		builder.WriteByte('*')
		offset = cursor
	}
	glob := builder.String()
	if compress && !strings.HasSuffix(strings.ToLower(glob), ".gz") {
		glob += ".gz"
	}
	return filepath.Clean(glob)
}

func rollingPatternIndexRegexp(pattern string, compress bool) (*regexp.Regexp, bool, error) {
	var builder strings.Builder
	builder.WriteByte('^')
	hasIndex := false
	slashPattern := filepath.ToSlash(pattern)
	for offset := 0; offset < len(slashPattern); {
		if slashPattern[offset] != '%' {
			builder.WriteString(regexp.QuoteMeta(string(slashPattern[offset])))
			offset++
			continue
		}
		if offset+1 < len(slashPattern) && slashPattern[offset+1] == '%' {
			builder.WriteString(regexp.QuoteMeta("%"))
			offset += 2
			continue
		}
		cursor := offset + 1
		if cursor < len(slashPattern) && slashPattern[cursor] == '0' {
			cursor++
		}
		for cursor < len(slashPattern) && slashPattern[cursor] >= '0' && slashPattern[cursor] <= '9' {
			cursor++
		}
		if cursor < len(slashPattern) && slashPattern[cursor] == 'i' {
			builder.WriteString(`(\d+)`)
			hasIndex = true
			offset = cursor + 1
			continue
		}
		if cursor < len(slashPattern) && slashPattern[cursor] == 'd' {
			cursor++
			if cursor < len(slashPattern) && slashPattern[cursor] == '{' {
				end := strings.IndexByte(slashPattern[cursor+1:], '}')
				if end >= 0 {
					cursor += end + 2
				}
			}
			builder.WriteString(`.+?`)
			offset = cursor
			continue
		}
		builder.WriteString(`.+?`)
		offset = cursor
	}
	if compress && !strings.HasSuffix(strings.ToLower(slashPattern), ".gz") {
		builder.WriteString(regexp.QuoteMeta(".gz"))
	}
	builder.WriteByte('$')
	compiled, err := regexp.Compile(builder.String())
	if err != nil {
		return nil, false, fmt.Errorf("goark-log: compile rolling filePattern index matcher: %w", err)
	}
	return compiled, hasIndex, nil
}
