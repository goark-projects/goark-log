package layout

import (
	"fmt"
	"regexp"
	"strings"

	"goark.dev/log/internal/logvalue"
	"goark.dev/log/internal/timepattern"
)

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
	if index < len(pattern) && pattern[index] == '0' {
		token.zeroPad = true
	}
	for index < len(pattern) && logvalue.IsPatternDigit(pattern[index]) {
		token.minWidth = token.minWidth*10 + int(pattern[index]-'0')
		index++
	}
	if index < len(pattern) && pattern[index] == '.' {
		index++
		if index < len(pattern) && pattern[index] == '-' {
			token.truncateFromEnd = true
			index++
		}
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
		return patternToken{}, 0, fmt.Errorf(
			"goark-log: unsupported pattern token near %q",
			pattern,
		)
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

type patternToken struct {
	kind            patternTokenKind
	literal         string
	format          string
	key             string
	minWidth        int
	maxWidth        int
	logger          loggerAbbreviator
	repeat          int
	leftAlign       bool
	truncateFromEnd bool
	zeroPad         bool
	timeUnix        timepattern.UnixMode
	child           *PatternLayout
	regex           *regexp.Regexp
	value           string
	repl            string
	ignore          bool
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

// PatternLayout 支持常用日志 pattern 占位符。
type PatternLayout struct {
	tokens  []patternToken
	options LayoutOptions
}

// NewPatternLayout 编译 pattern，避免热路径反复解析。
func NewPatternLayout(pattern string) (*PatternLayout, error) {
	return NewPatternLayoutWithOptions(pattern, LayoutOptions{})
}
