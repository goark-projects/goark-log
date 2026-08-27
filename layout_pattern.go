package goarklog

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"goark.dev/log/internal/callsite"
	"goark.dev/log/internal/logvalue"
	"goark.dev/log/internal/timepattern"
)

// PatternLayout 支持常用日志 pattern 占位符。
type PatternLayout struct {
	tokens  []patternToken
	options LayoutOptions
}

type patternToken struct {
	kind      patternTokenKind
	literal   string
	format    string
	key       string
	minWidth  int
	maxWidth  int
	precision int
	repeat    int
	leftAlign bool
	timeUnix  timepattern.UnixMode
	child     *PatternLayout
	regex     *regexp.Regexp
	value     string
	repl      string
	ignore    bool
}

type patternTokenKind int

const (
	tokenLiteral patternTokenKind = iota
	tokenTime
	tokenLevel
	tokenPID
	tokenThread
	tokenLogger
	tokenMessage
	tokenAttrs
	tokenAttr
	tokenError
	tokenNewline
	tokenMarker
	tokenContextStack
	tokenCallerClass
	tokenCallerMethod
	tokenCallerFile
	tokenCallerLine
	tokenCallerLocation
	tokenUUID
	tokenRelative
	tokenHost
	tokenSubPattern
	tokenHighlight
	tokenStyle
	tokenNotEmpty
	tokenReplace
	tokenEncode
	tokenEquals
	tokenMaxLen
	tokenRepeat
	tokenSequence
)

// NewPatternLayout 编译 pattern，避免热路径反复解析。
func NewPatternLayout(pattern string) (*PatternLayout, error) {
	return NewPatternLayoutWithOptions(pattern, LayoutOptions{})
}

// NewPatternLayoutWithOptions 使用指定布局参数编译 pattern。
func NewPatternLayoutWithOptions(pattern string, options LayoutOptions) (*PatternLayout, error) {
	if strings.TrimSpace(pattern) == "" {
		pattern = DefaultSpringBootPattern
	}
	tokens, err := compilePattern(pattern, options)
	if err != nil {
		return nil, err
	}
	return &PatternLayout{tokens: tokens, options: options}, nil
}

func (l *PatternLayout) Format(buf *bytes.Buffer, event Event) error {
	if l == nil {
		return NewDefaultLayout().Format(buf, event)
	}
	var caller callsite.Cache
	for _, token := range l.tokens {
		appendPatternToken(buf, token, event, &caller, l.options)
	}
	return nil
}

func compilePattern(pattern string, options LayoutOptions) ([]patternToken, error) {
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
		token, size, err := readPatternToken(pattern, options)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
		pattern = pattern[size:]
	}
	return tokens, nil
}

func readPatternToken(pattern string, layoutOptions LayoutOptions) (patternToken, int, error) {
	if strings.HasPrefix(pattern, "%%") {
		return patternToken{kind: tokenLiteral, literal: "%"}, 2, nil
	}
	index := 1
	token := patternToken{}
	if index < len(pattern) && pattern[index] == '-' {
		token.leftAlign = true
		index++
	}
	for index < len(pattern) && logvalue.IsPatternDigit(pattern[index]) {
		token.minWidth = token.minWidth*10 + int(pattern[index]-'0')
		index++
	}
	if index < len(pattern) && pattern[index] == '.' {
		index++
		for index < len(pattern) && logvalue.IsPatternDigit(pattern[index]) {
			token.maxWidth = token.maxWidth*10 + int(pattern[index]-'0')
			index++
		}
	}
	converterStart := index
	if index < len(pattern) && pattern[index] == 'X' {
		index++
	} else {
		for index < len(pattern) && logvalue.IsPatternLetter(pattern[index]) {
			index++
		}
	}
	if converterStart == index {
		return patternToken{}, 0, fmt.Errorf("goark-log: unsupported pattern token near %q", pattern)
	}
	converter := pattern[converterStart:index]
	options := []string(nil)
	if index < len(pattern) && pattern[index] == '{' {
		for index < len(pattern) && pattern[index] == '{' {
			option, next, err := readPatternOption(pattern, index)
			if err != nil {
				return patternToken{}, 0, err
			}
			options = append(options, option)
			index = next
		}
	}
	if err := configurePatternToken(&token, converter, options, layoutOptions); err != nil {
		return patternToken{}, 0, err
	}
	return token, index, nil
}

func readPatternOption(pattern string, start int) (string, int, error) {
	depth := 0
	for index := start; index < len(pattern); index++ {
		switch pattern[index] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return pattern[start+1 : index], index + 1, nil
			}
		}
	}
	return "", 0, fmt.Errorf("goark-log: pattern option is not closed near %q", pattern[start:])
}

func configurePatternToken(token *patternToken, converter string, options []string, layoutOptions LayoutOptions) error {
	normalized := strings.ToLower(converter)
	option := firstPatternOption(options)
	switch {
	case converter == "C" || normalized == "class":
		token.kind = tokenCallerClass
	case converter == "M" || normalized == "method":
		token.kind = tokenCallerMethod
	case converter == "F" || normalized == "file":
		token.kind = tokenCallerFile
	case converter == "L" || normalized == "line":
		token.kind = tokenCallerLine
	case converter == "l" || normalized == "location":
		token.kind = tokenCallerLocation
	case normalized == "d" || normalized == "date":
		token.kind = tokenTime
		token.format, token.timeUnix = timepattern.Normalize(option)
	case normalized == "level" || normalized == "p":
		token.kind = tokenLevel
	case normalized == "pid" || normalized == "processid":
		token.kind = tokenPID
	case normalized == "thread" || normalized == "t":
		token.kind = tokenThread
	case normalized == "logger" || converter == "c":
		token.kind = tokenLogger
		token.precision = parsePatternPrecision(option)
	case normalized == "msg" || normalized == "message" || converter == "m":
		token.kind = tokenMessage
	case normalized == "attrs" || normalized == "kvp" || normalized == "map":
		token.kind = tokenAttrs
	case converter == "X" || normalized == "mdc":
		if strings.TrimSpace(option) == "" {
			token.kind = tokenAttrs
			return nil
		}
		token.kind = tokenAttr
		token.key = strings.TrimSpace(option)
	case normalized == "ex" || normalized == "throwable" || normalized == "exception":
		token.kind = tokenError
		token.key = strings.ToLower(strings.TrimSpace(option))
	case normalized == "marker":
		token.kind = tokenMarker
	case normalized == "ndc" || normalized == "x":
		token.kind = tokenContextStack
	case normalized == "n":
		token.kind = tokenNewline
	case normalized == "uuid":
		token.kind = tokenUUID
	case normalized == "relative" || normalized == "r":
		token.kind = tokenRelative
	case normalized == "host" || normalized == "hostname":
		token.kind = tokenHost
	case normalized == "sequencenumber" || normalized == "sn":
		token.kind = tokenSequence
	case normalized == "highlight":
		child, err := NewPatternLayoutWithOptions(option, layoutOptions)
		if err != nil {
			return err
		}
		token.kind = tokenHighlight
		token.child = child
	case normalized == "style":
		child, err := NewPatternLayoutWithOptions(option, layoutOptions)
		if err != nil {
			return err
		}
		token.kind = tokenStyle
		token.child = child
		token.value = patternOption(options, 1)
	case normalized == "notempty":
		child, err := NewPatternLayoutWithOptions(option, layoutOptions)
		if err != nil {
			return err
		}
		token.kind = tokenNotEmpty
		token.child = child
	case normalized == "replace":
		if len(options) < 3 {
			return fmt.Errorf("goark-log: replace pattern converter requires pattern, regex and replacement")
		}
		child, err := NewPatternLayoutWithOptions(options[0], layoutOptions)
		if err != nil {
			return err
		}
		expression, err := regexp.Compile(options[1])
		if err != nil {
			return fmt.Errorf("goark-log: replace pattern regex %q is invalid: %w", options[1], err)
		}
		token.kind = tokenReplace
		token.child = child
		token.regex = expression
		token.repl = options[2]
	case normalized == "enc" || normalized == "encode":
		child, err := NewPatternLayoutWithOptions(option, layoutOptions)
		if err != nil {
			return err
		}
		token.kind = tokenEncode
		token.child = child
		token.value = strings.ToLower(strings.TrimSpace(patternOption(options, 1)))
	case normalized == "equals" || normalized == "equalsignorecase":
		if len(options) < 3 {
			return fmt.Errorf("goark-log: %s pattern converter requires pattern, test and substitution", converter)
		}
		child, err := NewPatternLayoutWithOptions(options[0], layoutOptions)
		if err != nil {
			return err
		}
		token.kind = tokenEquals
		token.child = child
		token.value = options[1]
		token.repl = options[2]
		token.ignore = normalized == "equalsignorecase"
	case normalized == "maxlen" || normalized == "maxlength":
		if len(options) < 2 {
			return fmt.Errorf("goark-log: maxLen pattern converter requires pattern and length")
		}
		child, err := NewPatternLayoutWithOptions(options[0], layoutOptions)
		if err != nil {
			return err
		}
		limit, err := strconv.Atoi(strings.TrimSpace(options[1]))
		if err != nil || limit < 0 {
			return fmt.Errorf("goark-log: maxLen pattern length %q is invalid", options[1])
		}
		token.kind = tokenMaxLen
		token.child = child
		token.repeat = limit
	case normalized == "repeat":
		if len(options) < 2 {
			return fmt.Errorf("goark-log: repeat pattern converter requires pattern and count")
		}
		child, err := NewPatternLayoutWithOptions(options[0], layoutOptions)
		if err != nil {
			return err
		}
		count, err := strconv.Atoi(strings.TrimSpace(options[1]))
		if err != nil || count < 0 {
			return fmt.Errorf("goark-log: repeat pattern count %q is invalid", options[1])
		}
		token.kind = tokenRepeat
		token.child = child
		token.repeat = count
	default:
		return fmt.Errorf("goark-log: unsupported pattern converter %q", converter)
	}
	return nil
}
