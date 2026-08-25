package goarklog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	// DefaultSpringBootPattern 是默认控制台输出格式，风格对齐 Spring Boot。
	DefaultSpringBootPattern = "%d %5level %pid --- [%thread] %logger : %msg%attrs%n"
	defaultTimeFormat        = "2006-01-02T15:04:05.000Z07:00"
)

// Layout 把日志事件编码为字节。
type Layout interface {
	Format(buf *bytes.Buffer, event Event) error
}

// NewDefaultLayout 创建默认 Spring Boot 风格布局。
func NewDefaultLayout() Layout {
	layout, _ := NewPatternLayout(DefaultSpringBootPattern)
	return layout
}

// TextLayout 输出稳定的 key=value 文本。
type TextLayout struct{}

func (TextLayout) Format(buf *bytes.Buffer, event Event) error {
	appendKeyValue(buf, "time", event.Time.Format(defaultTimeFormat))
	appendKeyValue(buf, "level", levelName(event.Level))
	appendKeyValue(buf, "logger", event.Logger)
	appendKeyValue(buf, "msg", event.Message)
	for _, attr := range event.Attrs {
		appendKeyValue(buf, attr.Key, attrValueString(attr.Value))
	}
	buf.WriteByte('\n')
	return nil
}

// JSONLayout 输出单行 JSON。
type JSONLayout struct{}

func (JSONLayout) Format(buf *bytes.Buffer, event Event) error {
	fields := make(map[string]any, len(event.Attrs)+4)
	fields["time"] = event.Time.Format(defaultTimeFormat)
	fields["level"] = levelName(event.Level)
	fields["logger"] = event.Logger
	fields["msg"] = event.Message
	for _, attr := range event.Attrs {
		fields[attr.Key] = attrValueAny(attr.Value)
	}
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(fields)
}

// PatternLayout 支持 Log4j2 风格的基础占位符子集。
type PatternLayout struct {
	tokens []patternToken
}

type patternToken struct {
	kind      patternTokenKind
	literal   string
	format    string
	key       string
	minWidth  int
	maxWidth  int
	leftAlign bool
	timeUnix  timeUnixMode
}

type patternTokenKind int

const (
	tokenLiteral patternTokenKind = iota
	tokenTime
	tokenLevel
	tokenLevelPadded
	tokenPID
	tokenThread
	tokenLogger
	tokenMessage
	tokenAttrs
	tokenAttr
	tokenError
	tokenNewline
)

type timeUnixMode uint8

const (
	timeUnixNone timeUnixMode = iota
	timeUnixSeconds
	timeUnixMillis
	timeUnixMicros
	timeUnixNanos
)

// NewPatternLayout 编译 pattern，避免热路径反复解析。
func NewPatternLayout(pattern string) (*PatternLayout, error) {
	if strings.TrimSpace(pattern) == "" {
		pattern = DefaultSpringBootPattern
	}
	tokens, err := compilePattern(pattern)
	if err != nil {
		return nil, err
	}
	return &PatternLayout{tokens: tokens}, nil
}

func (l *PatternLayout) Format(buf *bytes.Buffer, event Event) error {
	if l == nil {
		return NewDefaultLayout().Format(buf, event)
	}
	for _, token := range l.tokens {
		appendPatternToken(buf, token, event)
	}
	return nil
}

func compilePattern(pattern string) ([]patternToken, error) {
	tokens := make([]patternToken, 0, 16)
	for len(pattern) > 0 {
		index := strings.IndexByte(pattern, '%')
		if index < 0 {
			tokens = append(tokens, patternToken{kind: tokenLiteral, literal: pattern})
			break
		}
		if index > 0 {
			tokens = append(tokens, patternToken{kind: tokenLiteral, literal: pattern[:index]})
			pattern = pattern[index:]
			continue
		}
		token, size, err := readPatternToken(pattern)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
		pattern = pattern[size:]
	}
	return tokens, nil
}

func readPatternToken(pattern string) (patternToken, int, error) {
	if strings.HasPrefix(pattern, "%%") {
		return patternToken{kind: tokenLiteral, literal: "%"}, 2, nil
	}
	index := 1
	token := patternToken{}
	if index < len(pattern) && pattern[index] == '-' {
		token.leftAlign = true
		index++
	}
	for index < len(pattern) && isPatternDigit(pattern[index]) {
		token.minWidth = token.minWidth*10 + int(pattern[index]-'0')
		index++
	}
	if index < len(pattern) && pattern[index] == '.' {
		index++
		for index < len(pattern) && isPatternDigit(pattern[index]) {
			token.maxWidth = token.maxWidth*10 + int(pattern[index]-'0')
			index++
		}
	}
	converterStart := index
	if index < len(pattern) && pattern[index] == 'X' {
		index++
	} else {
		for index < len(pattern) && isPatternLetter(pattern[index]) {
			index++
		}
	}
	if converterStart == index {
		return patternToken{}, 0, fmt.Errorf("goark-log: unsupported pattern token near %q", pattern)
	}
	converter := pattern[converterStart:index]
	option := ""
	if index < len(pattern) && pattern[index] == '{' {
		var err error
		option, index, err = readPatternOption(pattern, index)
		if err != nil {
			return patternToken{}, 0, err
		}
	}
	if err := configurePatternToken(&token, converter, option); err != nil {
		return patternToken{}, 0, err
	}
	return token, index, nil
}

func readPatternOption(pattern string, start int) (string, int, error) {
	end := strings.IndexByte(pattern[start+1:], '}')
	if end < 0 {
		return "", 0, fmt.Errorf("goark-log: pattern option is not closed near %q", pattern[start:])
	}
	return pattern[start+1 : start+1+end], start + end + 2, nil
}

func configurePatternToken(token *patternToken, converter string, option string) error {
	switch strings.ToLower(converter) {
	case "d", "date":
		token.kind = tokenTime
		token.format, token.timeUnix = normalizeTimePattern(option)
	case "level", "p":
		token.kind = tokenLevel
	case "pid", "processid":
		token.kind = tokenPID
	case "thread", "t":
		token.kind = tokenThread
	case "logger", "c":
		token.kind = tokenLogger
	case "msg", "message", "m":
		token.kind = tokenMessage
	case "attrs", "kvp":
		token.kind = tokenAttrs
	case "x", "mdc":
		if strings.TrimSpace(option) == "" {
			token.kind = tokenAttrs
			return nil
		}
		token.kind = tokenAttr
		token.key = strings.TrimSpace(option)
	case "ex", "throwable", "exception":
		token.kind = tokenError
	case "n":
		token.kind = tokenNewline
	default:
		return fmt.Errorf("goark-log: unsupported pattern converter %q", converter)
	}
	return nil
}

func appendPatternToken(buf *bytes.Buffer, token patternToken, event Event) {
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
	value := patternTokenString(token, event)
	appendPadded(buf, value, token.minWidth, token.maxWidth, token.leftAlign)
}

func patternTokenString(token patternToken, event Event) string {
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
	case tokenLevel, tokenLevelPadded:
		return levelName(event.Level)
	case tokenPID:
		return strconv.Itoa(os.Getpid())
	case tokenThread:
		return "main"
	case tokenLogger:
		return event.Logger
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
		return eventErrorString(event)
	case tokenNewline:
		return "\n"
	default:
		return ""
	}
}

func normalizeTimePattern(format string) (string, timeUnixMode) {
	switch strings.ToUpper(strings.TrimSpace(format)) {
	case "", "DEFAULT", "ISO8601", "ISO8601_OFFSET_DATE_TIME":
		return defaultTimeFormat, timeUnixNone
	case "RFC3339":
		return time.RFC3339, timeUnixNone
	case "RFC3339NANO":
		return time.RFC3339Nano, timeUnixNone
	case "UNIX", "UNIX_SECONDS":
		return "", timeUnixSeconds
	case "UNIX_MILLIS", "UNIX_MS":
		return "", timeUnixMillis
	case "UNIX_MICROS", "UNIX_US":
		return "", timeUnixMicros
	case "UNIX_NANOS", "UNIX_NS":
		return "", timeUnixNanos
	default:
		return javaDatePatternToGo(format), timeUnixNone
	}
}

func javaDatePatternToGo(format string) string {
	replacer := strings.NewReplacer(
		"yyyy", "2006",
		"yy", "06",
		"MM", "01",
		"dd", "02",
		"HH", "15",
		"mm", "04",
		"ss", "05",
		"SSSSSS", "000000",
		"SSS", "000",
		"XXX", "Z07:00",
		"XX", "-0700",
		"X", "-07",
	)
	return replacer.Replace(format)
}

func eventErrorString(event Event) string {
	for _, key := range []string{"error", "err"} {
		value, ok := event.Attr(key)
		if ok {
			return attrValueString(value)
		}
	}
	return ""
}

func appendPadded(buf *bytes.Buffer, value string, minWidth int, maxWidth int, leftAlign bool) {
	if maxWidth > 0 && len(value) > maxWidth {
		value = value[:maxWidth]
	}
	if minWidth <= len(value) {
		buf.WriteString(value)
		return
	}
	padding := minWidth - len(value)
	if leftAlign {
		buf.WriteString(value)
		writeSpaces(buf, padding)
		return
	}
	writeSpaces(buf, padding)
	buf.WriteString(value)
}

func writeSpaces(buf *bytes.Buffer, count int) {
	for index := 0; index < count; index++ {
		buf.WriteByte(' ')
	}
}

func isPatternDigit(value byte) bool {
	return value >= '0' && value <= '9'
}

func isPatternLetter(value byte) bool {
	return value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z'
}

func appendPatternAttrs(buf *bytes.Buffer, attrs []slog.Attr) {
	for _, attr := range attrs {
		buf.WriteByte(' ')
		appendKeyValue(buf, attr.Key, attrValueString(attr.Value))
	}
}

func appendKeyValue(buf *bytes.Buffer, key string, value string) {
	if buf.Len() > 0 {
		buf.WriteByte(' ')
	}
	buf.WriteString(key)
	buf.WriteByte('=')
	quoteValue(buf, value)
}

func quoteValue(buf *bytes.Buffer, value string) {
	if value == "" || strings.ContainsAny(value, " \t\r\n\"=") {
		buf.WriteString(strconv.Quote(value))
		return
	}
	buf.WriteString(value)
}

func attrValueString(value slog.Value) string {
	value = value.Resolve()
	switch value.Kind() {
	case slog.KindString:
		return value.String()
	case slog.KindTime:
		return value.Time().Format(time.RFC3339Nano)
	case slog.KindDuration:
		return value.Duration().String()
	case slog.KindGroup:
		var builder strings.Builder
		for index, attr := range value.Group() {
			if index > 0 {
				builder.WriteByte(',')
			}
			builder.WriteString(attr.Key)
			builder.WriteByte('=')
			builder.WriteString(attrValueString(attr.Value))
		}
		return builder.String()
	default:
		return fmt.Sprint(value.Any())
	}
}

func attrValueAny(value slog.Value) any {
	value = value.Resolve()
	switch value.Kind() {
	case slog.KindString:
		return value.String()
	case slog.KindBool:
		return value.Bool()
	case slog.KindInt64:
		return value.Int64()
	case slog.KindUint64:
		return value.Uint64()
	case slog.KindFloat64:
		return value.Float64()
	case slog.KindDuration:
		return value.Duration().String()
	case slog.KindTime:
		return value.Time().Format(time.RFC3339Nano)
	case slog.KindGroup:
		group := value.Group()
		out := make(map[string]any, len(group))
		for _, attr := range group {
			out[attr.Key] = attrValueAny(attr.Value)
		}
		return out
	default:
		return value.Any()
	}
}

func leftPad(value string, width int) string {
	if len(value) >= width {
		return value
	}
	return strings.Repeat(" ", width-len(value)) + value
}
