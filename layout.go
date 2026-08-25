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
	DefaultSpringBootPattern = "%d %-5level %pid --- [%thread] %logger : %msg%attrs%n"
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
	kind    patternTokenKind
	literal string
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
	tokenNewline
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
		switch token.kind {
		case tokenLiteral:
			buf.WriteString(token.literal)
		case tokenTime:
			buf.WriteString(event.Time.Format(defaultTimeFormat))
		case tokenLevel:
			buf.WriteString(levelName(event.Level))
		case tokenLevelPadded:
			buf.WriteString(leftPad(levelName(event.Level), 5))
		case tokenPID:
			buf.WriteString(strconv.Itoa(os.Getpid()))
		case tokenThread:
			buf.WriteString("main")
		case tokenLogger:
			buf.WriteString(event.Logger)
		case tokenMessage:
			buf.WriteString(event.Message)
		case tokenAttrs:
			appendPatternAttrs(buf, event.Attrs)
		case tokenNewline:
			buf.WriteByte('\n')
		}
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
	for _, item := range []struct {
		prefix string
		kind   patternTokenKind
	}{
		{"%%", tokenLiteral},
		{"%-5level", tokenLevelPadded},
		{"%level", tokenLevel},
		{"%d", tokenTime},
		{"%pid", tokenPID},
		{"%thread", tokenThread},
		{"%logger", tokenLogger},
		{"%msg", tokenMessage},
		{"%attrs", tokenAttrs},
		{"%n", tokenNewline},
	} {
		if strings.HasPrefix(pattern, item.prefix) {
			token := patternToken{kind: item.kind}
			if item.prefix == "%%" {
				token.literal = "%"
			}
			return token, len(item.prefix), nil
		}
	}
	return patternToken{}, 0, fmt.Errorf("goark-log: unsupported pattern token near %q", pattern)
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
